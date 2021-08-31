module github.com/neilgerring/systemd-docker

go 1.16

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.8.1

require (
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/docker/docker v20.10.3-0.20210804232411-deda3d4933d3+incompatible
	github.com/fsouza/go-dockerclient v1.7.2
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f // indirect
	github.com/weaveworks/common v0.0.0-20210722103813-e649eff5ab4a
)
