# Introduction
This repository provides a wrapper that allows for a better integration of Docker containers as `systemd` services. 

Usually, a Docker container is launched with `docker run ...` where `docker` is the *Docker client* - a command 
line utility connected to the *Docker engine running in another process*, which executes the *image builds, running 
containers, etc. in yet other processes*. If a Docker container is started as a `systemd` service using 
an instruction like `ExecStart=docker run ...`, **`systemd` is attached to the Docker client process instead of the 
actual container process, which can lead to a bunch of odd situations**:
- the client can detach or crash while the container is fine, yet `systemd` would trigger failure handling 
- worse, the container crashes and requires care, but the client stalled - `systemd` is blind and won't trigger 
  anything
- when a container is stopped with `docker stop ...`, attached client processes exit with an error code instead of 
  0/success. This triggers `systemd`'s failure handling whereas in fact the container/service was properly shut down

The **key thing that this wrapper does is** that it moves the container process from the *cgroups set up by Docker* 
to the *service unit's cgroup* **to give `systemd` the mean to supervise the actual Docker container process**. 
It's written in Golang and allows to *leverage all the cgroup functionality of `systemd` and `systemd-notify`*.

# Repository history and credits
- the code was written by [@ibuildthecloud](https://github.com/ibuildthecloud) and his co-contributors in this [repository](https://github.com/ibuildthecloud/systemd-docker). 
The motivation is explained in this [Docker issue #6791](https://github.com/docker/docker/issues/6791) and this [mailing list thread](https://groups.google.com/d/topic/coreos-dev/wf7G6rA7Bf4/discussion).
- [@agend07](https://github.com/agend07) and co-contributors fixed outdated dependancies and did a first clean-up
- I removed all outdated and broken elements and created a new compilation docker container which can be found [here]()

# Installation
Supposing that a Go environment is available, the build instruction is `go get github.com/dontsetse/systemd-docker`. The 
executable can then be found in the Go binary directory, usually something like `$GO_ROOT/bin`. 

It can also be build using a stand-alone docker image, see [here]()

# Use
Both
- `systemctl` to manage `systemd` services, and
- the `docker` CLI

can be used and everything should stay in sync.

In the `systemd` unit files, the instruction to launch the Docker container takes the form 

`ExecStart=systemd-docker [<systemd-docker_options>] run [<docker-run_options>] <image_name> [<container_parameters>]`

where
- `<systemd-docker_options>` are explained below in the section Options
- `<docker-run_options>` are the usual flags defined by `docker run` - a few restriction apply, see section 
  Docker restrictions
- `<image_name>` is the name of the Docker image to run
- `<container_parameters>` are the parameters provided to the container when it's started  

Note: like any executable, `systemd-docker` should be in folder that is part of `$PATH` to be able to use it globally, 
      otherwise use a absolute path in the instruction like f.ex. `ExecStart=/opt/bin/systemd-docker ...` 

Here's an unit file example to run a Nginx container:
```ini
[Unit]
Description=Nginx
After=docker.service
Requires=docker.service

[Service]
ExecStart=systemd-docker run --rm --name %n nginx
Restart=always
RestartSec=10s
Type=notify
NotifyAccess=all
TimeoutStartSec=120
TimeoutStopSec=15

[Install]
WantedBy=multi-user.target
```
Note: `Type=notify` and `NotifyAccess=all` are important

## Container names
Container names are compulsory to make sure that a `systemd` service always relates to/acts upon the same container(s). 
While it may seem as if that could be omitted as long as the `--rm` flag used, that's misleading: the deletion process 
triggered by this flag is actually part of the Docker client logic and if the client detaches for whatever reason from 
the running container, the information is lost (even if another client is re-attached later) and *the container will 
**not** be deleted*.
 
`systemd-docker` adds an additional check and looks for the named container on start and if it exists and is stopped, 
it will be deleted.

# Systemd 
## Automatic container naming
`systemd` populates a range of variables among which %n stands for the name of service (derived from it's filename). This 
allows to write a self-configuring `ExecStart` instructions using the parameters 
`ExecStart=systemd-docker ... run ... --name %n --rm ...`.

## Use of `systemd` environment variables
See the Environment variables section in the systemd-docker options

# Systemd-docker options
## Logging
By default the container's stdout/stderr will be piped to the journal. Add `--logs=false` before the `run` instruction 
to disable, as shown below:

`ExecStart=systemd-docker --logs=false run ...`

## Environment Variables
`systemd` handles environment variables with the instructions `Environment=...` and `EnvironmentFile=...`. To inject 
variables into other instructions, the pattern *${varible_name}* is used, for example:

`ExecStart=systemd-docker ... run -e ABC=${ABC} -e XYZ=${XYZ} ...` 

The systemd environment variables are automatically passed through to the docker container if the `--env` flag is provided.  
This will essentially read all the current environment variables and add the appropriate `-e ...` flags to the docker run 
command.  For example:

```
EnvironmentFile=/etc/environment
ExecStart=systemd-docker --env run ...
```
The contents of `/etc/environment` will be added to the docker run command.

## Cgroups
The main magic of how this works is that the container processes are moved from the Docker cgroups to the system unit cgroups.  
By default all application cgroups will be moved. This means by default you can't use `--cpuset` or `-m` in Docker.  If you 
don't want to use the systemd cgroups, but instead use the Docker cgroups, you can control which cgroups are transfered using 
the `--cgroups` option.  **Minimally you must set `name=systemd`; otherwise, systemd will lose track of the container**.  For 
example

`ExecStart=/opt/bin/systemd-docker --cgroups name=systemd --cgroups=cpu run --rm --name %n nginx`

The above command will use the `name=systemd` and `cpu` cgroups of systemd but then use Docker's cgroups for all the others, like the freezer cgroup.

## Pid File
To create a PID file for the container, just add `--pid-file=<path/to/pid_file>` as shown below

`ExecStart=systemd-docker --pid-file=/var/run/%n.pid run ...`

## systemd-notify support

By default `systemd-docker` will send READY=1 to the `systemd` notification socket.  With the `systemd-docker` `--notify` flag the READY=1 call is 
delegated to the container itself:

`ExecStart=systemd-docker --notify run ...`

If this flag is provided to `systemd-docker` it bind mounts the `systemd` notification socket into the container and sets the NOTIFY_SOCKET 
environment variable. Please be aware that `systemd-notify` comes with its own quirks - more info can be found in this 
[mailing list thread](http://comments.gmane.org/gmane.comp.sysutils.systemd.devel/18649).  In short, `systemd-notify` is not reliable because often 
the child dies before `systemd` has time to determine which cgroup it is a member of.

# Docker restrictions
## `--cpuset` or `-m`
These flags can't be used because they are incompatible with the cgroup migration systemd-docker requires. 

## `-d` flag / detaching the client
The `-d` argument to docker has no effect under `systemd-docker`. To cause the `systemd-docker` client to detach after the container is running, use 
`--logs=false --rm=false`. If either `--logs` or `--rm` is true, the Docker client instance used by `systemd-docker` is kept alive until the service 
is stopped or the container exits.

# Known issues
## Inconsistent cgroup
CentOS 7 is inconsistent in the way it handles some cgroups. 
It has `3:cpuacct,cpu:/user.slice` in `/proc/[pid]/cgroups` which is inconsistent with the cgroup path `/sys/fs/cgroup/cpu,cpuacct/` that systemd-docker is trying to move pids to.

This will cause `systemd-docker` to fail unless run with`systemd-docker --cgroups name=systemd run`

See https://github.com/ibuildthecloud/systemd-docker/issues/15 for details.

# License
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)
