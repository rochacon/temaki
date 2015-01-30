package main

import (
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"net"
	"time"
)

func LaunchContainer(name string, service Service, container chan<- *docker.Container, quit <-chan bool, finished chan<- bool) {
	dcli, err := docker.NewClient("tcp://127.0.0.1:2375")
	if err != nil {
		container <- nil
		return
	}

	fmt.Printf("[%s] Creating container\n", name)
	c, err := dcli.CreateContainer(docker.CreateContainerOptions{
		"",
		&docker.Config{
			Image:     service.Image,
			PortSpecs: []string{service.Port + "/tcp"},
		},
		nil,
	})
	if err != nil {
		fmt.Printf("[%s] %s\n", name, err.Error())
		container <- nil
		return
	}

	if err := dcli.StartContainer(c.ID, &docker.HostConfig{}); err != nil {
		fmt.Printf("[%s] %s\n", name, err.Error())
		container <- nil
		return
	}

	c, err = dcli.InspectContainer(c.ID)
	if err != nil {
		fmt.Printf("[%s] %s\n", name, err.Error())
		container <- nil
		return
	}

	if cmds, ok := service.Hooks["pre-run"]; ok {
		for _, cmd := range cmds {
			fmt.Printf("[%s] TODO: exec pre-run hook: %s\n", name, cmd)
		}
	}

	// Wait for port listen
	for x := 0; ; x++ {
		addr := fmt.Sprintf("%s:%s", c.NetworkSettings.IPAddress, firstPort(c.NetworkSettings.Ports))
		if conn, err := net.Dial("tcp", addr); err == nil {
			conn.Close()
			break
		}
		time.Sleep(1 * time.Second)

		if x == 10 {
			fmt.Printf("[%s] max connection retries exceeded\n", name)
			container <- nil
			return
		}
	}

	fmt.Printf("[%s] Created\n", name)
	container <- c

	// Waiting for quit signal
	<-quit

	if cmds, ok := service.Hooks["post-run"]; ok {
		for _, cmd := range cmds {
			fmt.Printf("[%s] TODO: exec post-run hook: %s\n", name, cmd)
		}
	}

	fmt.Printf("[%s] Removing container\n", name)
	dcli.StopContainer(c.ID, 10)
	dcli.RemoveContainer(docker.RemoveContainerOptions{ID: c.ID})

	finished <- true
}

func Exec(cID string, cfg docker.ExecProcessConfig) error {
	return nil
}

func firstPort(ports map[docker.Port][]docker.PortBinding) string {
	for p := range ports {
		return p.Port()
	}
	return ""
}
