package sockets

import (
	"net"
	"net/http"
	"time"

	"github.com/Microsoft/go-winio"
)

func configureUnixTransport(tr *http.Transport, proto, addr string) error {
	return ErrProtocolNotAvailable
}

func configureNpipeTransport(tr *http.Transport, proto, addr string) error {
	// No need for compression in local communications.
	tr.DisableCompression = true
	tr.Dial = func(_, _ string) (net.Conn, error) {
		return DialPipe(addr, defaultTimeout)
	}
	return nil
}

func configureSSHTransport(tr *http.Transport, proto, addr string) error {
	return ErrProtocolNotAvailable
}

// DialPipe connects to a Windows named pipe.
func DialPipe(addr string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(addr, &timeout)
}

// DialSSH connects to a Unix socket over SSH.
// This is not supported on other OSes.
func DialSSH(addr string) (net.Conn, error) {
	return nil, syscall.EAFNOSUPPORT
}
