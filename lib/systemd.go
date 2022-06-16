package lib

import (
	"errors"
	"fmt"
	"net"
	"os"
)

func SetupSystemdNotify(c *Context) {
	c.NotifySocket = os.Getenv("NOTIFY_SOCKET")
	newArgs := []string{}
	if c.Notify {
		if len(c.NotifySocket) > 0 {
			newArgs = append(newArgs, "-e", fmt.Sprintf("NOTIFY_SOCKET=%s", c.NotifySocket))
			newArgs = append(newArgs, "-v", fmt.Sprintf("%s:%s", c.NotifySocket, c.NotifySocket))
		} else {
			c.Log.Warnf("No NOTIFY_SOCKET is found")
		}
	}
	if len(newArgs) > 0 {
		c.Args = append(newArgs, c.Args...)
	}
}

const (
	SdNotifyReady     = "READY=1"
	SdNotifyStopping  = "STOPPING=1"
	SdNotifyReloading = "RELOADING=1"
	SdNotifyWatchdog  = "WATCHDOG=1"
)

func Notify(c *Context) error {

	if !c.Notify {
		return nil
	}

	socketAddr := &net.UnixAddr{
		Name: c.NotifySocket,
		Net:  "unixgram",
	}

	// NOTIFY_SOCKET not set
	if socketAddr.Name == "" {
		return errors.New("NOTIFY_SOCKET not set")
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	// Error connecting to NOTIFY_SOCKET
	if err != nil {
		return err
	}
	defer conn.Close()

	if IsPidRunning(c.Pid) {
		_, err = conn.Write([]byte(fmt.Sprintf("MAINPID=%d", c.Pid)))
		if err != nil {
			return err
		}

		_, err = conn.Write([]byte(SdNotifyReady))
		if err != nil {
			return err
		}
	} else {
		_, err = conn.Write([]byte(fmt.Sprintf("MAINPID=%d", os.Getpid())))
		if err != nil {
			return err
		}
		_, err = conn.Write([]byte(SdNotifyStopping))
		if err != nil {
			return err
		}
		return errors.New("container exited before we could notify systemd")
	}

	return nil
}
