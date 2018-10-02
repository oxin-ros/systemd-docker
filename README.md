# Introduction
This repository provides a wrapper that allows for a better integration of Docker containers as `systemd` services. 

Usually, a Docker container is launched with `docker run ...`. It's important to note that `docker` stands for the 
*Docker client* - a command line utility connected to the *Docker engine running in another process*, which executes 
the image builds, *running containers, etc. in yet other processes*. This **"process confusion"** leads to problems 
if containers are meant to be run as `systemd` services - the equivalent instruction for service launch 
`ExecStart=docker run ...` would imply that `systemd` monitors the client instead of the actual container. That leads 
to a bunch of odd situations:
- the client can detach or crash while the container is fine, yet systemd would trigger failure handling 
- worse, the container crashes and should be restarted, but the client stalled and the problem goes unnoticed
- when a container is stopped using `docker stop`, attached client processes exit with an error code instead of 
  0/success. This triggers `systemd`'s failure handling whereas in fact the container/service was properly shut down

The **key thing that this wrapper does is** that it moves the container process from the *cgroups* set up by Docker 
to the service unit's cgroup **to make systemd supervise the Docker container process**. It's written in Golang and 
allows to *leverage all the cgroup functionality of `systemd` and `systemd-notify`*.

# Repository history and credits
- the code was written by [@ibuildthecloud](https://github.com/ibuildthecloud) and his co-contributors in this [repository](https://github.com/ibuildthecloud/systemd-docker). 
The motivation is explained in this [Docker Issue #6791](https://github.com/docker/docker/issues/6791) and this [mailing list thread](https://groups.google.com/d/topic/coreos-dev/wf7G6rA7Bf4/discussion).
- [@agend07](https://github.com/agend07) and co-contributors fixed outdated dependancies and clean-up a bit
- I removed all outdated and broken elements and created a new compilation docker container which can be found [here]()

# Installation
Supposing that a Go environment is available, the build instruction is `go get github.com/dontsetse/systemd-docker`. The 
executable can then be found in the Go binary directory, f.ex. `/go/bin`. 

It can also be build using a stand-alone docker image, see [here]()

# Usage
Both
- `systemctl` to manage systemd services, and
- the `docker` CLI

can be used and everything should stay in sync.

Basically, the command is `systemd-docker run` instead of `docker run`.  Here's an unit file example to run a nginx container:
```ini
[Unit]
Description=Nginx
After=docker.service
Requires=docker.service

[Service]
ExecStart=/opt/bin/systemd-docker run --rm --name %n nginx
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

## Container naming
Container names are compulsory to make sure that a `systemd` service always relates to/acts upon the same container(s). 
While it may seem as if that could be omitted as long as the `--rm` flag is passed to `docker run` or rather 
`systemd-docker run`, that's misleading: the deletion process triggered by this flag is actually part of the Docker client 
logic and if the client detaches for whatever reason from the running container, the information is lost (even if another 
client is re-attached later) and the container will not be deleted. 
`systemd-docker` looks for the named container on start and if it exists and is stopped, it will be deleted.
The variable %n is populated by systemd with the name of the service which allows to write a `ExecStart` instruction 
with the parameters `... --name %n --rm ...`.

# Options
## Logging
By default the container's stdout/stderr will be piped to the journal. Add `--logs=false` before the `run` instruction, 
as shown below:

`ExecStart=systemd-docker --logs=false run ...`

## Environment Variables
`systemd` handles environment variables with the instructions `Environment=...` and `EnvironmentFile=...`. To inject 
variables into other instructions, the pattern ${varible_name} is used, for example:

`ExecStart=systemd-docker run -e ABC=${ABC} -e XYZ=${XYZ}` 

The systemd environment variables are automatically transfered to the docker container if the `--env` flag is provided.  
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
If for whatever reason a pid file for the container PID is required, it's easy to create one. Just add `--pid-file` as below

`ExecStart=/opt/bin/systemd-docker --pid-file=/var/run/%n.pid --env run --rm --name %n nginx`

## systemd-notify support

By default `systemd-docker` will send READY=1 to the systemd notification socket.  You can instead delegate the READY=1 call to the container itself.  This is done by adding `--notify`.  For example

`ExecStart=/opt/bin/systemd-docker --notify run --rm --name %n nginx`

What this will do is set up a bind mount for the notification socket and then set the NOTIFY_SOCKET environment variable.  If you are going to use this feature of systemd, take some time to understand the quirks of it.  More info in this [mailing list thread](http://comments.gmane.org/gmane.comp.sysutils.systemd.devel/18649).  In short, systemd-notify is not reliable because often the child dies before systemd has time to determine which cgroup it is a member of

## Detaching the client
The `-d` argument to docker has no effect under `systemd-docker`. To cause the `systemd-docker` client to detach after the container is running, use `--logs=false --rm=false`. If either `--logs` or `--rm` is true, the `systemd-docker` client will stay alive until it is killed or the container exits.

# Known issues
## Inconsistent cgroup
CentOS 7 is inconsistent in the way it handles some cgroups. 
It has `3:cpuacct,cpu:/user.slice` in `/proc/[pid]/cgroups` which is inconsistent with the cgroup path `/sys/fs/cgroup/cpu,cpuacct/` that systemd-docker is trying to move pids to.

This will cause `systemd-docker` to fail unless run with`systemd-docker --cgroups name=systemd run`

See https://github.com/ibuildthecloud/systemd-docker/issues/15 for details.

# License
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)
