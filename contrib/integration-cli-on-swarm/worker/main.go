package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/bfirsh/funker-go"
)

func main() {
	// workerImage needs to be this image itself
	workerImage := os.Getenv("WORKER_IMAGE")
	if workerImage == "" {
		fmt.Fprintf(os.Stderr, "WORKER_IMAGE unset\n")
		os.Exit(1)
	}
	if err := xmain(workerImage); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func xmain(workerImage string) error {
	log.Printf("Worker started, image=%s", workerImage)
	return funker.Handle(func(pTest *string) int {
		test := *pTest
		log.Printf("Executing %q", test)
		code, err := executeTest(workerImage, test)
		if err != nil {
			log.Printf("Error while executing %q: %v", test, err)
			return 1
		}
		log.Printf("Finished %q, code=%d", test, code)
		return code
	})
}

func executeTest(workerImage, test string) (int, error) {
	// NOTE: docker.sock needs to be bind-mounted
	// TODO: support other TESTFLAGS as well (e.g. -race)
	cmd := exec.Command("docker",
		"run",
		"--rm",
		"-i",
		"--privileged",
		"-e", "TESTFLAGS=-check.f "+test,
		"-e", "KEEPBUNDLE=1",
		"-e", "BINDDIR=",
		"--entrypoint", "/bin/bash",
		workerImage,
		"hack/make.sh",
		"test-integration-cli",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return executeCommand(cmd)
}

func executeCommand(cmd *exec.Cmd) (int, error) {
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
		} else {
			return 1, err
		}
	}
	return 0, nil
}
