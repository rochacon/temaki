package main

import (
	docker "github.com/fsouza/go-dockerclient"
	"log"
	"time"
)

func LaunchContainer(name string, service Service, container chan<- *docker.Container, quit <-chan bool, finished chan<- bool) {
	dcli, err := docker.NewClient("tcp://127.0.0.1:2375")
	if err != nil {
		container <- nil
		return
	}

	log.Printf("[%s] Creating container\n", name)
	c, err := dcli.CreateContainer(docker.CreateContainerOptions{
		"",
		&docker.Config{
			Image:     service.Image,
			PortSpecs: []string{service.Port + "/tcp"},
		},
		nil,
	})
	if err != nil {
		log.Println("DEBUG", err.Error())
		container <- nil
		return
	}

	log.Printf("[%s] Starting container\n", name)
	if err := dcli.StartContainer(c.ID, &docker.HostConfig{}); err != nil {
		container <- nil
		return
	}

	log.Printf("[%s] Inspecting container\n", name)
	c, err = dcli.InspectContainer(c.ID)
	if err != nil {
		container <- nil
		return
	}

	if cmds, ok := service.Hooks["pre-run"]; ok {
		for _, cmd := range cmds {
			log.Println("// TODO(rochacon): exec pre-run hook:", cmd)
		}
	}

	// FIXME(rochacon): wait for port listen instead of time
	time.Sleep(10 * time.Second)

	container <- c

	log.Printf("[%s] Waiting for quit signal\n", name)
	<-quit

	if cmds, ok := service.Hooks["post-run"]; ok {
		for _, cmd := range cmds {
			log.Println("// TODO(rochacon): exec post-run hook:", cmd)
		}
	}

	log.Printf("[%s] Stopping container\n", name)
	dcli.StopContainer(c.ID, 10)

	log.Printf("[%s] Removing container\n", name)
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
