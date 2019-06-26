package main

import (
	"os"

	"golang.org/x/sys/windows/svc"
)

const serviceName = "Dynatrace OneAgent"

type service struct{}

func (m *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	ok := true
	for ok {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				ok = false
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}

	return false, 0
}

func main() {
	// If argument in command line, run as service.
	if len(os.Args) > 1 && os.Args[1] == "service" {
		svc.Run(serviceName, &service{})
		return
	}

	// Otherwise, run the installer.
	os.Exit(installService())
}
