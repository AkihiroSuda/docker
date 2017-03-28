package client

import (
	"net"
	"net/url"

	"github.com/pkg/errors"
)

func sshDial(u *url.URL) (net.Conn, error){
	return nil, errors.New("sshDial unimplemented yet")
}
