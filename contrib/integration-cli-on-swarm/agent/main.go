package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/contrib/integration-cli-on-swarm/agent/master"
	"github.com/docker/docker/contrib/integration-cli-on-swarm/agent/worker"
)

func main() {
	var err error
	binName := filepath.Base(os.Args[0])
	switch binName {
	case "master":
		err = master.Main()
	case "worker":
		err = worker.Main()
	default:
		err = fmt.Errorf("Wrong binary name: %s (needs to be \"master\" or \"worker\")", binName)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
