package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/flynn/go-shlex"
	"github.com/fsouza/go-dockerclient"
)

func dockerHost() string {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		return "unix://var/run/docker.sock"
	}
	return host
}

// Build builds a Dockerfile
func Build(name, dockerfile string, output io.Writer) error {
	dcli, err := docker.NewClient(dockerHost())
	if err != nil {
		return err
	}
	opts := docker.BuildImageOptions{
		ContextDir:          filepath.Dir(dockerfile),
		Dockerfile:          filepath.Base(dockerfile),
		Name:                name,
		ForceRmTmpContainer: true,
		OutputStream:        output,
	}
	if err := dcli.BuildImage(opts); err != nil {
		return err
	}
	return nil
}

func LaunchContainer(name string, service Service, container chan<- *docker.Container, quit <-chan bool, finished chan<- bool) {
	dcli, err := docker.NewClient(dockerHost())
	if err != nil {
		container <- nil
		return
	}

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

	// Execute pre-run hooks
	if cmds, ok := service.Hooks["pre-run"]; ok {
		for _, cmd := range cmds {
			if err := Exec(dcli, c.ID, cmd); err != nil {
				fmt.Printf("[%s] pre-run: %s\n", name, err.Error())
				container <- nil
				return
			}
		}
	}

	container <- c

	// Waiting for quit signal
	<-quit

	// Execute post-run hooks
	if cmds, ok := service.Hooks["post-run"]; ok {
		for _, cmd := range cmds {
			if err := Exec(dcli, c.ID, cmd); err != nil {
				fmt.Printf("[%s] post-run: %s\n", name, err.Error())
				container <- nil
				return
			}
		}
	}

	dcli.StopContainer(c.ID, 10)
	dcli.RemoveContainer(docker.RemoveContainerOptions{ID: c.ID})

	finished <- true
}

func Exec(dcli *docker.Client, cID, cmd string) error {
	cmdsplt, err := shlex.Split(cmd)
	if err != nil {
		return err
	}
	exec, err := dcli.CreateExec(docker.CreateExecOptions{
		Cmd:       cmdsplt,
		Container: cID,
	})
	if err != nil {
		return err
	}
	return dcli.StartExec(exec.ID, docker.StartExecOptions{})
}

func firstPort(ports map[docker.Port][]docker.PortBinding) string {
	for p := range ports {
		return p.Port()
	}
	return ""
}
