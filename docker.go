package main

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
		RmTmpContainer:      true,
		ForceRmTmpContainer: true,
		OutputStream:        output,
	}
	if err := dcli.BuildImage(opts); err != nil {
		return err
	}
	return nil
}

// LaunchService launches a Service
func LaunchService(name string, service Service, container chan<- *docker.Container, quit <-chan bool, finished chan<- bool) {
	dcli, err := docker.NewClient(dockerHost())
	if err != nil {
		container <- nil
		return
	}

	c, err := dcli.CreateContainer(docker.CreateContainerOptions{
		"",
		&docker.Config{
			Env: service.Env,
			ExposedPorts: map[docker.Port]struct{}{
				docker.Port(service.Port + "/tcp"): struct{}{},
			},
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

	err = dcli.StartContainer(c.ID, &docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port(service.Port + "/tcp"): []docker.PortBinding{
				docker.PortBinding{},
			},
		},
	})
	if err != nil {
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

	// Get exposed Host and Port
	host, port, err := getExposedHostAndPort(service.Port, c.NetworkSettings.Ports)
	if err != nil {
		fmt.Printf("[%s] Unable to retrieve container Host and Port: %s\n", name, err.Error())
		container <- nil
		return
	}

	// Wait for port listen
	for x := 0; ; x++ {
		addr := fmt.Sprintf("%s:%s", host, port)
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

func RunTestSuite(image, command string, env []string, stdout, stderr io.Writer) error {
	dcli, err := docker.NewClient(dockerHost())
	if err != nil {
		return err
	}
	cmd, err := shlex.Split(command)
	if err != nil {
		return err
	}
	c, err := dcli.CreateContainer(docker.CreateContainerOptions{
		"",
		&docker.Config{
			Cmd:        cmd[1:],
			Entrypoint: []string{cmd[0]},
			Env:        env,
			Image:      image,
		},
		nil,
	})
	if err != nil {
		return err
	}
	defer dcli.RemoveContainer(docker.RemoveContainerOptions{ID: c.ID})

	if err := dcli.StartContainer(c.ID, &docker.HostConfig{}); err != nil {
		return err
	}
	defer dcli.StopContainer(c.ID, 10)

	logOpts := docker.LogsOptions{
		Container:    c.ID,
		Follow:       true,
		Stdout:       true,
		OutputStream: stdout,
		Stderr:       true,
		ErrorStream:  stderr,
	}
	if err := dcli.Logs(logOpts); err != nil {
		return err
	}

	return nil
}

// fixHostIfRemoteDaemon checks if DOCKER_HOST env is a remote host and
// replaces the 0.0.0.0 with the remote host
// If DOCKER_HOST is unix socket, return the received host
func fixHostIfRemoteDaemon(host string) string {
	dockerHost := dockerHost()
	if host == "0.0.0.0" && strings.HasPrefix(dockerHost, "tcp://") {
		u, err := url.Parse(dockerHost)
		if err != nil {
			return host
		}
		h, _, _ := net.SplitHostPort(u.Host)
		return h
	}
	return host
}

// getExposedHostAndPort returns the exposed HostIP and HostPort for a given port binding
func getExposedHostAndPort(port string, ports map[docker.Port][]docker.PortBinding) (string, string, error) {
	for p, exposed := range ports {
		if p.Port() != port {
			continue
		}
		return fixHostIfRemoteDaemon(exposed[0].HostIP), exposed[0].HostPort, nil
	}
	return "", "", fmt.Errorf("Exposed Host/Port not found.")
}
