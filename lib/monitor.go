package lib

import (
	"context"
	"os"
	"strconv"
	"time"
)

type Monitor struct {
	timer    *time.Ticker
	exit     chan bool
	interval int
}

func NewMonitor() *Monitor {

	interval := os.Getenv("WATCHDOG_USEC")
	if interval == "" {
		return nil
	}
	intervalMs, err := strconv.Atoi(interval)
	if err != nil {
		return nil
	}

	m := &Monitor{
		timer:    &time.Ticker{},
		exit:     make(chan bool),
		interval: intervalMs / 2,
	}

	return m
}

func (m *Monitor) RunMonitor(c *Context) {
	c.Log.Infof("Start systemd watchdog monitor with interval %d", m.interval)
	Watchdog(c)
	m.timer = time.NewTicker((time.Duration(m.interval) * time.Microsecond))
	for {
		select {
		case <-m.exit:
			return
		case t := <-m.timer.C:
			inspect, err := c.Client.ContainerInspect(context.Background(), c.Id)
			if err != nil {
				c.Log.Infof("Invalid Container", t)
			}
			if inspect.State.Running && c.Pid == inspect.State.Pid {
				err := Watchdog(c)
				if err != nil {
					c.Log.Warnf("Failed to send SD_Watchdog")
				}
			} else {
				c.Log.Infof("Container to monitor is not Ok ", t)
			}
		}
	}
}

func (m *Monitor) StopMonitor() {
	m.exit <- true
}
