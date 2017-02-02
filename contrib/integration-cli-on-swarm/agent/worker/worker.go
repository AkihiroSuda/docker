package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/bfirsh/funker-go"
	"github.com/docker/docker/contrib/integration-cli-on-swarm/agent/types"
)

func main() {
	if err := xmain(); err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}

func seemsValidImageID(s string) bool {
	return !strings.Contains(s, "/")
}

func xmain() error {
	workerImage := flag.String("worker-image", "", "Needs to be this image itself")
	dryRun := flag.Bool("dry-run", false, "Dry run")
	flag.Parse()
	if !seemsValidImageID(*workerImage) {
		// Because of issue #29582.
		// `docker service create localregistry.example.com/blahblah:latest` pulls the image data to local, but not a tag.
		// So, `docker run localregistry.example.com/blahblah:latest` fails: `Unable to find image 'localregistry.example.com/blahblah:latest' locally`
		return fmt.Errorf("Currently, worker-image must be an ID, not a name (even with a tag). "+
			"%q does not seem a valid ID", *workerImage)
	}
	return handle(*workerImage, *dryRun)
}

func handle(workerImage string, dryRun bool) error {
	log.Printf("Waiting for a funker request")
	return funker.Handle(func(args *types.Args) types.Result {
		log.Printf("Executing chunk %d, contains %d test filters",
			args.ChunkID, len(args.Tests))
		begin := time.Now()
		rawLog, code, err := executeTestChunk(workerImage, args.Tests, dryRun)
		if err != nil {
			log.Printf("Error while executing chunk %d: %v", args.ChunkID, err)
			return types.Result{
				ChunkID: args.ChunkID,
				Code:    1,
			}
		}
		elapsed := time.Now().Sub(begin)
		log.Printf("Finished chunk %d, code=%d, elapsed=%v", args.ChunkID, code, elapsed)
		return types.Result{
			ChunkID: args.ChunkID,
			Code:    code,
			RawLog:  string(rawLog),
		}
	})
}

// executeTests executes a chunk of tests and returns the single error code
// FIXME: it should return []int, rather than int.
func executeTestChunk(workerImage string, tests []string, dryRun bool) ([]byte, int, error) {
	testFlags := "-check.f " + strings.Join(tests, "|")
	// NOTE: docker.sock needs to be bind-mounted
	// TODO: support other TESTFLAGS as well
	// TODO: use docker/client instead of os/exec.
	cmd := exec.Command("docker",
		"run",
		"--rm",
		"-i",
		"--privileged",
		"-e", "TESTFLAGS="+strings.TrimSpace(testFlags),
		"-e", "KEEPBUNDLE=1",
		"-e", "DOCKER_INTEGRATION_TESTS_VERIFIED=1",
		"-e", "BINDDIR=", // for avoiding bind-mounting "bundles" dir
		"--entrypoint", "hack/dind",
		workerImage,
		"hack/make.sh", "test-integration-cli",
	)
	if dryRun {
		return dryRunCommand(cmd)
	}
	return runCommand(cmd)
}

func dryRunCommand(cmd *exec.Cmd) ([]byte, int, error) {
	log.Printf("DRYRUN %v", cmd.Args)
	return nil, 0, nil
}

func runCommand(cmd *exec.Cmd) ([]byte, int, error) {
	var rawLog bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &rawLog)
	cmd.Stderr = io.MultiWriter(os.Stderr, &rawLog)
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return rawLog.Bytes(), status.ExitStatus(), nil
			}
		} else {
			return rawLog.Bytes(), 1, err
		}
	}
	return rawLog.Bytes(), 0, nil
}
