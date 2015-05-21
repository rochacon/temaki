//
// temaki - Test environment wrapper
//
// Examples:
//     temaki # read command from temaki.yml cmd entry
//     temaki gradle test
//     temaki mvn test
//     temaki py.test ./tests/
//
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/fsouza/go-dockerclient"
)

func main() {
	conf, err := ConfigFromFile()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Build app container image
	fmt.Println("---> Building container")
	if err := Build(conf.Name, conf.Dockerfile, os.Stdout); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create test environment/services
	fmt.Println("---> Creating test environment")

	cmd_env := []string{}
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
			Host string
			Port string
		}{
			Host: container.NetworkSettings.IPAddress,
			Port: firstPort(container.NetworkSettings.Ports),
		}); err != nil {
			panic(err)
		}

		fmt.Printf("     %s=%q\n", envvar, formatted.String())
		cmd_env = append(cmd_env, fmt.Sprintf("%s=%s", envvar, formatted.String()))
	}

	// Run test suite in container
	fmt.Println("---> Starting test suite:", conf.Cmd, "\n")
	if err := RunTestSuite(conf.Name, conf.Cmd, os.Stdout, os.Stderr); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("") // Break line after test suite output

	// Destroy test environment/services
	fmt.Println("---> Cleaning test environment")

	// Send quit signal to test services
	for x := 0; x < len(conf.Env); x++ {
		quit <- true
	}

	// Wait finished notification from test services
	for x := 0; x < len(conf.Env); x++ {
		<-finished
	}
}
