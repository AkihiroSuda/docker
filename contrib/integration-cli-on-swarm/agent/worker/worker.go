package worker

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/bfirsh/funker-go"
	"github.com/docker/docker/contrib/integration-cli-on-swarm/agent/types"
)

func seemsValidImageID(s string) bool {
	return !strings.Contains(s, "/")
}

// Main is the entrypoint for worker agent.
// TODO: should we use flags rather than os.Getenv?
func Main() error {
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
		return fmt.Errorf("Currently, WORKER_IMAGE must be an ID, not a name (even with a tag). "+
			"%q does not seem a valid ID", workerImage)
	}
	dryRun := os.Getenv("DRY_RUN") != ""
	return handle(workerImage, dryRun)
}

func handle(workerImage string, dryRun bool) error {
	return funker.Handle(func(args *types.Args) types.Result {
		log.Printf("Executing chunk %d, contains %d tests",
			args.ChunkID, len(args.Tests))
		begin := time.Now()
		code, err := executeTestChunk(workerImage, args.Tests, dryRun)
		if err != nil {
			log.Printf("Error while executing chunk %d: %v", args.ChunkID, err)
			return types.Result{
				ChunkID: args.ChunkID,
				Code: 1,
			}
		}
		elapsed := time.Now().Sub(begin)
		log.Printf("Finished chunk %d, code=%d, elapsed=%v", args.ChunkID, code, elapsed)
		return types.Result{
			ChunkID: args.ChunkID,
			Code:    code,
		}
	})
}

// executeTests executes a chunk of tests and returns the single error code
// TODO: it should return []int, rather than int.
func executeTestChunk(workerImage string, tests []string, dryRun bool) (int, error) {
	testFlags := ""
	for _, test := range tests {
		// our local fork of go-check supports multiple -check.f. (OR-match)
		testFlags += "-check.f " + test + " "
	}

	// NOTE: docker.sock needs to be bind-mounted
	// TODO: support other TESTFLAGS as well (e.g. -race)
	cmd := exec.Command("docker",
		"run",
		"--rm",
		"-i",
		"--privileged",
		"-e", "TESTFLAGS="+strings.TrimSpace(testFlags),
		"-e", "KEEPBUNDLE=1",
		"-e", "DOCKER_INTEGRATION_TESTS_VERIFIED=1",
		"-e", "BINDDIR=", // for avoiding bind-mounting "bundles" dir
		"--entrypoint", "/bin/bash",
		workerImage,
		"hack/make.sh", "test-integration-cli",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if dryRun {
		return dryRunCommand(cmd)
	}
	return runCommand(cmd)
}

func dryRunCommand(cmd *exec.Cmd) (int, error) {
	log.Printf("DRYRUN %v", cmd.Args)
	return 0, nil
}

func runCommand(cmd *exec.Cmd) (int, error) {
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
