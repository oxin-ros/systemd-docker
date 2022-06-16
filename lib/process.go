package lib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

func IsPidRunning(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return !os.IsNotExist(err)
}

func CreatePidFile(pidfile string, pid int) error {
	err := ioutil.WriteFile(pidfile, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return err
	}

	return nil
}

func PidFile(c *Context) error {
	if len(c.PidFile) == 0 || c.Pid <= 0 {
		return nil
	}

	if !IsPidRunning(c.Pid) {
		return errors.New("pid file not created container not running")
	}

	err := CreatePidFile(c.PidFile, c.Pid)
	if err != nil {
		return err
	}

	return nil
}
