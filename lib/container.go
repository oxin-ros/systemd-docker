package lib

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func lookupByName(containers []types.Container, name string) types.Container {
	searchName := "/" + name
	for _, container := range containers {
		if container.Names[0] == searchName {
			return container
		}
	}
	return types.Container{}
}

func lookupNamedContainer(c *Context, name string) (types.Container, error) {

	filterArgs := filters.NewArgs()
	filterArgs.Add("name", name)

	client, err := c.GetClient()
	if err != nil {
		return types.Container{}, err
	}

	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return types.Container{}, err
	}

	container := lookupByName(containers, c.Name)
	return container, nil
}

func getDockerCommand() string {
	dockerCommand := os.Getenv("DOCKER_COMMAND")
	if len(dockerCommand) == 0 {
		dockerCommand = "docker"
	}
	return dockerCommand
}

func removeContainer(c *Context, Id string) error {
	client, err := c.GetClient()
	if err != nil {
		return err
	}

	return client.ContainerRemove(context.Background(), Id, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: false,
		RemoveLinks:   false,
	})
}

func startContainer(c *Context, Id string) error {
	client, err := c.GetClient()
	if err != nil {
		return err
	}

	return client.ContainerStart(context.Background(), Id, types.ContainerStartOptions{
		CheckpointID:  "",
		CheckpointDir: "",
	})
}

func createContainer(c *Context) (types.Container, error) {

	client, err := c.GetClient()
	if err != nil {
		return types.Container{}, err
	}

	args := append([]string{"create"}, c.Args...)
	dockerCommand := getDockerCommand()

	c.Cmd = exec.Command(dockerCommand, args...)

	errorPipe, err := c.Cmd.StderrPipe()
	if err != nil {
		return types.Container{}, err
	}

	outputPipe, err := c.Cmd.StdoutPipe()
	if err != nil {
		return types.Container{}, err
	}

	err = c.Cmd.Start()
	if err != nil {
		return types.Container{}, err
	}

	go func() {
		_, _ = io.Copy(os.Stderr, errorPipe)
	}()

	bytes, err := ioutil.ReadAll(outputPipe)
	if err != nil {
		return types.Container{}, err
	}

	Id := strings.TrimSpace(string(bytes))

	err = c.Cmd.Wait()
	if err != nil {
		return types.Container{}, err
	}

	if !c.Cmd.ProcessState.Success() {
		return types.Container{}, err
	}

	filterArgs := filters.NewArgs()
	filterArgs.Add("id", Id)

	var containers []types.Container
	containers, err = client.ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return types.Container{}, err
	}

	return containers[0], nil
}

func isEmty(s types.Container) bool {
	return len(s.ID) == 0
}

func RunContainer(c *Context) error {

	var container types.Container
	var err error
	if len(c.Name) > 0 {
		container, err = lookupNamedContainer(c, c.Name)
		if err != nil {
			c.Log.Errorf("failed to lookupNamedContainer: %s\n", err)
			return err
		}

		if !isEmty(container) && c.Rm {
			err = removeContainer(c, container.ID)
			if err != nil {
				c.Log.Errorf("failed to removeContainer : %s\n", err)
				return err
			}
			container.ID = ""
		}
	}

	if isEmty(container) {
		container, err = createContainer(c)
		if err != nil {
			c.Log.Errorf("failed to createContainer : %s\n", err)
			return err
		}
	}

	inspect, err := c.Client.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		return err
	}

	if inspect.ContainerJSONBase.State.Running {
		c.Id = inspect.ID
		c.Pid = inspect.State.Pid
		return nil
	} else {
		err = startContainer(c, inspect.ID)
		if err != nil {
			c.Log.Errorf("failed to startContainer : %s\n", err)
			return err
		}

		inspect, err = c.Client.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			return err
		}

		c.Id = inspect.ID
		c.Pid = inspect.State.Pid
		return nil
	}
}
