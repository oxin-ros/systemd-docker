#go mod init github.com/oxin-ros/systemd-docker


vendor:
	rm -r vendor 2> /dev/null || true
	go mod vendor

build:
	rm Build/systemd-docker 2> /dev/null || true
	go build -buildvcs=false -mod vendor -o Build/systemd-docker

.PHONY: vendor build