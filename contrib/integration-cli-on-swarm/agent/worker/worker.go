package worker

import (
	"flag"
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
func Main() error {
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
	return funker.Handle(func(args *types.Args) types.Result {
		log.Printf("Executing chunk %d, contains %d test filters",
			args.ChunkID, len(args.Tests))
		begin := time.Now()
		code, err := executeTestChunk(workerImage, args.Tests, dryRun)
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
		}
	})
}

// executeTests executes a chunk of tests and returns the single error code
// TODO: it should return []int, rather than int.
func executeTestChunk(workerImage string, tests []string, dryRun bool) (int, error) {
	testFlags := "-check.f " + strings.Join(tests, "|")
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
