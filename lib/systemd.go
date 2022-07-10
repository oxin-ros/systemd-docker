package lib

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

const (
	SdNotifyReady     = "READY=1"
	SdNotifyStopping  = "STOPPING=1"
	SdNotifyReloading = "RELOADING=1"
	SdNotifyWatchdog  = "WATCHDOG=1"
)

type Notifier struct {
	wc io.WriteCloser
}

func AddNotify(c *Context) (*Notifier, error) {
	NotifySocket := os.Getenv("NOTIFY_SOCKET")

	newArgs := []string{}
	if c.Notify {
		if len(NotifySocket) > 0 {
			newArgs = append(newArgs, "-e", fmt.Sprintf("NOTIFY_SOCKET=%s", NotifySocket))
			newArgs = append(newArgs, "-v", fmt.Sprintf("%s:%s", NotifySocket, NotifySocket))
		} else {
			c.Log.Warnf("No NOTIFY_SOCKET is found")
		}
	}
	if len(newArgs) > 0 {
		c.Args = append(newArgs, c.Args...)
	}

	return Open(NotifySocket)
}

func Notify(c *Context) error {
	if !c.Notify {
		return nil
	}

	if IsPidRunning(c.Pid) {
		err := c.Notifier.Send(fmt.Sprintf("MAINPID=%d", c.Pid))
		if err != nil {
			return err
		}

		err = c.Notifier.Send(SdNotifyReady)
		if err != nil {
			return err
		}

		c.Notifier.Send(SdNotifyWatchdog)

	} else {
		err := c.Notifier.Send(fmt.Sprintf("MAINPID=%d", os.Getpid()))
		if err != nil {
			return err
		}
		err = c.Notifier.Send(SdNotifyStopping)
		if err != nil {
			return err
		}
		return errors.New("container exited before we could notify systemd")
	}

	return nil

}

func Watchdog(c *Context) error {

	err := c.Notifier.Send(SdNotifyWatchdog)
	if err != nil {
		return err
	}
	return nil
}

func Open(sock string) (*Notifier, error) {
	c, err := net.Dial("unixgram", sock)
	if err != nil {
		return nil, err
	}

	return &Notifier{wc: c}, nil
}

func (n *Notifier) Send(s ...string) error {
	if n == nil || len(s) == 0 {
		return nil
	}

	_, err := io.WriteString(n.wc, strings.Join(s, "\n"))
	return err
}

func (n *Notifier) Close() error {
	if n == nil {
		return nil
	}

	return n.wc.Close()
}
