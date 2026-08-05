package main

import (
	"bank/internal/runner"
)

func main() {
	if runAsService() {
		return
	}

	if err := runner.Run(); err != nil {
		panic(err)
	}
}
