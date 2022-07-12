package lib

import (
	"os/exec"

	"github.com/docker/docker/client"
)

type Context struct {
	Args       []string
	Cgroups    []string
	AllCgroups bool
	Logs       bool
	Notify     bool
	Name       string
	Env        bool
	Rm         bool
	Id         string
	Cmd        *exec.Cmd
	Pid        int
	PidFile    string
	Notifier   *Notifier
	Client     *client.Client
	Monitor    *Monitor
	Log        *logger
}

func (c *Context) GetClient() (*client.Client, error) {
	var err error
	if c.Client == nil {
		c.Client, err = client.NewClientWithOpts(client.FromEnv)
	}
	return c.Client, err
}
