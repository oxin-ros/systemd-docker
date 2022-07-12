// Copyright © 2021 Joel Baranick <jbaranick@gmail.com>
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
// 	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	SYSFS       string = "/sys/fs/cgroup"
	PROCS       string = "cgroup.procs"
	CGROUP_PROC string = "/proc/%d/cgroup"
)

func constructCgroupPath(cgroupName string, cgroupPath string) string {
	return path.Join(SYSFS, strings.TrimPrefix(cgroupName, "name="), cgroupPath, PROCS)
}

func writePid(pid string, path string) error {
	return ioutil.WriteFile(path, []byte(pid), 0644)
}

func getCgroupPids(cgroupName string, cgroupPath string) ([]string, error) {
	ret := []string{}

	file, err := os.Open(constructCgroupPath(cgroupName, cgroupPath))
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ret = append(ret, strings.TrimSpace(scanner.Text()))
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

func getCgroupsForPid(pid int) (map[string]string, error) {
	file, err := os.Open(fmt.Sprintf(CGROUP_PROC, pid))
	if err != nil {
		return nil, err
	}

	ret := map[string]string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.SplitN(scanner.Text(), ":", 3)
		if len(line) != 3 || line[1] == "" {
			continue
		}

		ret[line[1]] = line[2]
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

func MoveCgroups(c *Context) (bool, error) {
	moved := false
	currentCgroups, err := getCgroupsForPid(os.Getpid())
	if err != nil {
		return false, err
	}

	containerCgroups, err := getCgroupsForPid(c.Pid)
	if err != nil {
		return false, err
	}

	var ns []string

	if c.AllCgroups || c.Cgroups == nil || len(c.Cgroups) == 0 {
		ns = make([]string, 0, len(containerCgroups))
		for value := range containerCgroups {
			ns = append(ns, value)
		}
	} else {
		ns = c.Cgroups
	}

	for _, nsName := range ns {
		currentPath, ok := currentCgroups[nsName]
		if !ok {
			continue
		}

		containerPath, ok := containerCgroups[nsName]
		if !ok {
			continue
		}

		if currentPath == containerPath || containerPath == "/" {
			continue
		}

		pids, err := getCgroupPids(nsName, containerPath)
		if err != nil {
			return false, err
		}

		for _, pid := range pids {
			pidInt, err := strconv.Atoi(pid)
			if err != nil {
				continue
			}

			if !IsPidRunning(pidInt) {
				continue
			}

			currentFullPath := constructCgroupPath(nsName, currentPath)
			c.Log.Infof("Moving pid %s to %s\n", pid, currentFullPath)
			err = writePid(pid, currentFullPath)
			if err != nil {
				return false, err
			}

			moved = true
		}
	}

	return moved, nil
}
