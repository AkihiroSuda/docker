package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/bfirsh/funker-go"
)

func seemsValidImageID(s string) bool {
	return !strings.Contains(s, "/")
}

func main() {
	// workerImage needs to be this image itself
	workerImage := os.Getenv("WORKER_IMAGE")
	if workerImage == "" {
		fmt.Fprintf(os.Stderr, "WORKER_IMAGE unset\n")
		os.Exit(1)
	}
	if !seemsValidImageID(workerImage) {
		// Because of issue #29582.
		// `docker service create localregistry.example.com/blahblah:latest` pulls the image data to local, but not a tag.
		// So, `docker run localregistry.example.com/blahblah:latest` fails: `Unable to find image 'localregistry.example.com/blahblah:latest' locally`
		fmt.Fprintf(os.Stderr, "Currently, WORKER_IMAGE must be an ID, not a name (even with a tag). "+
			"%q does not seem a valid ID\n", workerImage)
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
		begin := time.Now()
		code, err := executeTest(workerImage, test)
		if err != nil {
			log.Printf("Error while executing %q: %v", test, err)
			return 1
		}
		elapsed := time.Now().Sub(begin)
		log.Printf("Finished %q, code=%d, elapsed=%v", test, code, elapsed)
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
		"-e", "DOCKER_INTEGRATION_TESTS_VERIFIED=1",
		"-e", "BINDDIR=", // for avoiding bind-mounting "bundles" dir
		"--entrypoint", "/bin/bash",
		workerImage,
		"hack/make.sh", "test-integration-cli",
		// "-c", "sleep 62",
		//
		// MEMO: the hanging issue seems related to some 60s stuff,
		//       it is even reproducible with "sleep" instead of hack/make.sh
		// - sleep  56: PASS
		// - sleep  58: Partially PASS, partially hangs
		// - sleep 62: HANG
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
