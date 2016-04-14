package logger

import (
	"bufio"
	"io"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

// Copier can copy logs from specified sources to Logger and attach
// ContainerID and Timestamp.
// Writes are concurrent, so you need implement some sync in your logger
type Copier struct {
	// cid is the container id for which we are copying logs
	cid string
	// srcs is map of name -> reader pairs, for example "stdout", "stderr"
	srcs     map[string]io.Reader
	dst      Logger
	copyJobs sync.WaitGroup
	closed   chan struct{}
}

// NewCopier creates a new Copier
func NewCopier(cid string, srcs map[string]io.Reader, dst Logger) *Copier {
	return &Copier{
		cid:    cid,
		srcs:   srcs,
		dst:    dst,
		closed: make(chan struct{}),
	}
}

// Run starts logs copying
func (c *Copier) Run() {
	for src, w := range c.srcs {
		c.copyJobs.Add(1)
		go c.copySrc(src, w)
	}
}

func (c *Copier) copySrc(name string, src io.Reader) {
	defer c.copyJobs.Done()
	reader := bufio.NewReader(src)

	scanner := bufio.NewScanner(reader)

	for {
		select {
		case <-c.closed:
			return
		default:
			scanned, err := scanner.Scan(), scanner.Err()
			if err != nil {
				logrus.Errorf("Error scanning log stream: %s", err)
				if err == bufio.ErrTooLong {
					return
				}
			}

			line := scanner.Bytes()
			if len(line) > 0 {
				if !scanned {
					line = append([]byte("<incomplete>"), line...)
				}
				if logErr := c.dst.Log(&Message{ContainerID: c.cid, Line: line, Source: name, Timestamp: time.Now().UTC()}); logErr != nil {
					logrus.Errorf("Failed to log msg %q for logger %s: %s", line, c.dst.Name(), logErr)
				}
			}
			if !scanned && err == nil {
				// scanner doesn't return io.EOF
				return
			}
		}
	}
}

// Wait waits until all copying is done
func (c *Copier) Wait() {
	c.copyJobs.Wait()
}

// Close closes the copier
func (c *Copier) Close() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}
