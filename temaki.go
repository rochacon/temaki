//
// temaki - Test environment wrapper
//
// Examples:
//     temaki # read command from .temaki.yml cmd entry
//     temaki gradle test
//     temaki mvn test
//     temaki py.test ./tests/
//
package main

import (
	"bytes"
	"fmt"
	"github.com/flynn/go-shlex"
	docker "github.com/fsouza/go-dockerclient"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

func main() {
	conf, err := ConfigFromFile()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cmd_env := []string{"GOPATH=/home/rochacon/dev"}
	finished := make(chan bool, len(conf.Env))
	quit := make(chan bool, len(conf.Env))

	for envvar, service := range conf.Env {
		// launch Docker container and pipe output with envvar prefix

		new_container := make(chan *docker.Container)
		go LaunchContainer(envvar, service, new_container, quit, finished)
		container := <-new_container
		if container == nil {
			log.Fatalf("[%s] container not created.\n", envvar)
		}

		var fmt_tmpl = template.Must(template.New(envvar).Parse(service.Format))
		formatted := bytes.NewBuffer([]byte{})
		if err := fmt_tmpl.Execute(formatted, &struct {
			Host  string
			Port0 string
		}{
			Host:  container.NetworkSettings.IPAddress,
			Port0: firstPort(container.NetworkSettings.Ports),
		}); err != nil {
			panic(err)
		}

		cmd_env = append(cmd_env, fmt.Sprintf("%s=%s", envvar, formatted.String()))
	}

	log.Println("---> Starting command:", conf.Cmd)
	log.Println("     Environment:", strings.Join(cmd_env, " "))

	// exec command
	cmd_splitted, _ := shlex.Split(conf.Cmd)
	cmd := exec.Command(cmd_splitted[0], cmd_splitted[1:]...)
	cmd.Env = cmd_env
	cmd.Stdout = &PrefixWriter{w: os.Stdout, Prefix: "cmd: "}
	cmd.Stderr = &PrefixWriter{w: os.Stderr, Prefix: "cmd: "}
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
	}

	for x := 0; x < len(conf.Env); x++ {
		quit <- true
	}

	log.Println("Waiting for shutdown of containers")
	for x := 0; x < len(conf.Env); x++ {
		<-finished
	}
}
