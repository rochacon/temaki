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
	if err := Build(conf.Image, conf.Dockerfile, os.Stdout); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create test environment/services
	fmt.Println("---> Creating test environment")

	testEnv := []string{}
	finished := make(chan bool, len(conf.Services))
	quit := make(chan bool, len(conf.Services))

	for envvar, service := range conf.Services {
		// launch Docker container and pipe output with envvar prefix

		new_container := make(chan *docker.Container)
		go LaunchContainer(envvar, service, new_container, quit, finished)
		container := <-new_container
		if container == nil {
			log.Fatalf("[%s] container not created.\n", envvar)
		}
		host, port, _ := getExposedHostAndPort(service.Port, container.NetworkSettings.Ports)

		var fmt_tmpl = template.Must(template.New(envvar).Parse(service.Format))
		formatted := bytes.NewBuffer([]byte{})
		if err := fmt_tmpl.Execute(formatted, &struct {
			Host string
			Port string
		}{
			Host: host,
			Port: port,
		}); err != nil {
			panic(err)
		}

		fmt.Printf("     %s=%q\n", envvar, formatted.String())
		testEnv = append(testEnv, fmt.Sprintf("%s=%s", envvar, formatted.String()))
	}

	// Run test suite in container
	fmt.Println("---> Starting test suite:", conf.Cmd, "\n")
	if err := RunTestSuite(conf.Image, conf.Cmd, testEnv, os.Stdout, os.Stderr); err != nil {
		fmt.Println(err)
	}
	fmt.Println("") // Break line after test suite output

	// Destroy test environment/services
	fmt.Println("---> Cleaning test environment")

	// Send quit signal to test services
	for x := 0; x < len(conf.Services); x++ {
		quit <- true
	}

	// Wait finished notification from test services
	for x := 0; x < len(conf.Services); x++ {
		<-finished
	}
}
