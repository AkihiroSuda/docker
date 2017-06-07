package main

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/client"
)

func main() {
	if err := xmain(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func xmain() error {
	fmt.Println("Connecting to the daemon using NewEnvClient().")
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	fmt.Printf("Connected to the daemon %q\n", cli.DaemonHost())

	fmt.Println("Daemon Info:")
	inf, err := cli.Info(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", inf)
	return cli.Close()
}
