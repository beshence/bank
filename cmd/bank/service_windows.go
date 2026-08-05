//go:build windows

package main

import (
	"bank/internal/runner"

	"golang.org/x/sys/windows/svc"
)

func runAsService() bool {
	isService, err := svc.IsWindowsService()
	if err != nil || !isService {
		return false
	}

	err = svc.Run(
		"BeshenceBank",
		&bankService{},
	)

	if err != nil {
		panic(err)
	}

	return true
}

type bankService struct{}

func (s *bankService) Execute(
	args []string,
	requests <-chan svc.ChangeRequest,
	status chan<- svc.Status,
) (
	exitCode bool,
	exitCodeNum uint32,
) {

	status <- svc.Status{
		State: svc.StartPending,
	}

	go func() {
		if err := runner.Run(); err != nil {
			panic(err)
		}
	}()

	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for request := range requests {

		switch request.Cmd {

		case svc.Stop, svc.Shutdown:
			status <- svc.Status{
				State: svc.StopPending,
			}

			return false, 0
		}
	}

	return false, 0
}
