package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// testChunkExecutor executes integration-cli binary.
// image needs to be the worker image itself. testFlags are OR-set of regexp for filtering tests.
type testChunkExecutor func(image string, tests []string) (int64, string, error)

func dryTestChunkExecutor(image string, tests []string) (int64, string, error) {
	return 0, fmt.Sprintf("DRY RUN (image=%q, tests=%v)", image, tests), nil
}

// privilegedTestChunkExecutor invokes a privileged container from the worker
// service via bind-mounted API socket so as to execute the test chunk
func privilegedTestChunkExecutor(image string, tests []string) (int64, string, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return 0, "", err
	}
	config := container.Config{
		Image: image,
		Env: []string{
			"TESTFLAGS=-check.f " + strings.Join(tests, "|"),
			"KEEPBUNDLE=1",
			"DOCKER_INTEGRATION_TESTS_VERIFIED=1", // for avoiding rebuilding integration-cli
			"BINDDIR=",                            // for avoiding bind-mounting "bundles" dir
		},
		// TODO: set label?
		Entrypoint: []string{"hack/dind"},
		Cmd:        []string{"hack/make.sh", "test-integration-cli"},
	}
	hostConfig := container.HostConfig{
		AutoRemove: true,
		Privileged: true,
	}
	id, stream, err := runContainer(context.Background(), cli, config, hostConfig)
	if err != nil {
		return 0, "", err
	}
	var b bytes.Buffer
	teeContainerStream(&b, os.Stdout, os.Stderr, stream)
	rc, err := cli.ContainerWait(context.Background(), id)
	if err != nil {
		return 0, "", err
	}
	return rc, b.String(), nil
}

func runContainer(ctx context.Context, cli *client.Client, config container.Config, hostConfig container.HostConfig) (string, io.ReadCloser, error) {
	created, err := cli.ContainerCreate(context.Background(),
		&config, &hostConfig, nil, "")
	if err != nil {
		return "", nil, err
	}
	if err = cli.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}); err != nil {
		return "", nil, err
	}
	stream, err := cli.ContainerLogs(ctx,
		created.ID,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
	return created.ID, stream, err
}

func teeContainerStream(w, stdout, stderr io.Writer, stream io.ReadCloser) {
	stdcopy.StdCopy(io.MultiWriter(w, stdout), io.MultiWriter(w, stderr), stream)
	stream.Close()
}
