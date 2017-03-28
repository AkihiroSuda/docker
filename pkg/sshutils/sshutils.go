// Package sshutils provides utilities for SSH
package sshutils

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// ResolveAuthMethods returns the list of available auth methods.
// Currently supported methods:
//  - ssh-agent (on Unixen)
//  - publickey (files under $HOME)
//
// Methods lkely to be supported in future:
//  - pageant (on Windows)
//
// Methods unlikely to be supported in future:
//  - keyboard interactive
//
// NOTE: currently, ~/.ssh/config and /etc/ssh/ssh_config are unused.
func ResolveAuthMethods() ([]ssh.AuthMethod, error) {
	home, err := getHome()
	if err != nil {
		return nil, err
	}
	return resolveAuthMethods(home, os.Getenv, ioutil.ReadFile)
}

func getHome() (string, error) {
	// TODO: use docker/pkg/homedir (mess for static bin, see comments in homedir_linux.go)
	// NOTE: even on Windows, ssh keys are likely to located under %HOME% (with cygwin/msys)
	home := os.Getenv("HOME")
	if home == "" {
		return "", errors.New("HOME unset")
	}
	return home, nil
}

// Dial dials with available auth methods
func Dial(user, n, addr string) (*ssh.Client, error) {
	auths, err := ResolveAuthMethods()
	if err != nil {
		return nil, err
	}
	if len(auths) == 0 {
		return nil, errors.New("no auth method found")
	}
	return ssh.Dial(n, addr, &ssh.ClientConfig{
		User: user,
		Auth: auths,
	})
}
