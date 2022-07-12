package lib

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

func isEmty(s types.Container) bool {
	return len(s.ID) == 0
}

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

	err = client.ContainerStart(context.Background(), Id, types.ContainerStartOptions{
		CheckpointID:  "",
		CheckpointDir: "",
	})
	if err != nil {
		return err
	}

	return nil
}

func waitExitOrRemoved(c *Context, containerID string, waitRemove bool) <-chan int {
	client := c.Client

	condition := container.WaitConditionNextExit
	if waitRemove {
		condition = container.WaitConditionRemoved
	}

	resultC, errC := client.ContainerWait(context.Background(), containerID, condition)

	statusC := make(chan int)
	go func() {
		select {
		case result := <-resultC:
			if result.Error != nil {
				statusC <- 125
			} else {
				statusC <- int(result.StatusCode)
			}
		case err := <-errC:
			c.Log.Errorf("error waiting for container: %v", err)
			statusC <- 125
		}
	}()

	return statusC
}

func ContainerLogs(c *Context, Id string) error {
	client, err := c.GetClient()
	if err != nil {
		return err
	}

	go func() {
		out, err := client.ContainerLogs(context.Background(), Id, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Timestamps: false,
		})
		_ = err
		defer out.Close()
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	return nil
}

func CreateContainer(c *Context) (types.Container, error) {

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
		container, err = CreateContainer(c)
		if err != nil {
			c.Log.Errorf("failed to CreateContainer : %s\n", err)
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
	}

	if c.Logs {
		err = ContainerLogs(c, inspect.ID)
	}
	return err
}

func WaitFinished(c *Context) error {
	statusC := waitExitOrRemoved(c, c.Id, c.Rm)
	exitCode := <-statusC
	if exitCode != 0 {
		return fmt.Errorf("container stopped with error value %d", exitCode)
	}
	return nil
}

func StopContainer(c *Context) error {
	if len(c.Id) == 0 {
		return nil
	}

	client, err := c.GetClient()
	if err != nil {
		return err
	}
	return client.ContainerStop(context.Background(), c.Id, nil)
}
