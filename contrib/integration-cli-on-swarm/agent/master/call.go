package master

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bfirsh/funker-go"
)

const (
	// defaultFunkerRetryTimeout is for the issue https://github.com/bfirsh/funker/issues/3
	// When all the funker replicas are busy in their own job, we cannot connect to funker.
	defaultFunkerRetryTimeout = 1 * time.Hour
	defaultFunkerRetryDuration = 1 * time.Second
	verbose                   = false
)

func executeTests(funkerName string, testChunks [][]string) error {
	begin := time.Now()
	log.Printf("Executing %d chunks in parallel, against %q", len(testChunks), funkerName)
	var wg sync.WaitGroup
	var passed, failed uint32
	for i, testChunk := range testChunks {
		wg.Add(1)
		// TODO: limit number of goroutines
		go func(i int, testChunk []string) {
			defer wg.Done()
			log.Printf("Executing %d-th chunk (contains %d tests)", i, len(testChunk))
			testChunkBegin := time.Now()
			code, err := executeTestChunkWithRetry(funkerName, testChunk, defaultFunkerRetryTimeout)
			if err != nil {
				log.Printf("Error while executing %d-th chunk: %v",
					i, err)
				atomic.AddUint32(&failed, 1)
			} else {
				if code == 0 {
					atomic.AddUint32(&passed, 1)
				} else {
					atomic.AddUint32(&failed, 1)
				}
				log.Printf("Finished %d-th chunk [%d/%d] in %s, code=%d.",
					i, passed+failed, len(testChunks), time.Now().Sub(testChunkBegin), code)
			}
		}(i, testChunk)
	}
	wg.Wait()
	// TODO: print tests rather than chunks
	log.Printf("Executed %d test chunks in %s. PASS: %d, FAIL: %d.",
		len(testChunks), time.Now().Sub(begin), passed, failed)
	if failed > 0 {
		return fmt.Errorf("%d test chunks failed", failed)
	}
	return nil
}

func executeTestChunk(funkerName string, testChunk []string) (int, error) {
	if verbose {
		log.Printf("Calling funker.Call (%q, %q)", funkerName, testChunk)
	}
	ret, err := funker.Call(funkerName, testChunk)
	if verbose {
		log.Printf("Called funker.Call (%q, %q)=(%v, %v)", funkerName, testChunk, ret, err)
	}
	if err != nil {
		return 1, err
	}
	code, ok := ret.(float64)
	if !ok {
		return 1, fmt.Errorf("unexpected result from funker.Call (%q, %q): %v",
			funkerName, testChunk, ret)
	}
	return int(code), nil
}

func executeTestChunkWithRetry(funkerName string, testChunk []string, funkerRetryTimeout time.Duration) (int, error) {
	begin := time.Now()
	for i := 0; time.Now().Sub(begin) < funkerRetryTimeout; i++ {
		if i > 0 && i%100 == 0 {
			log.Printf("Calling executeTestChunk(%q, %q), trial %d", funkerName, testChunk, i)
		}
		code, err := executeTestChunk(funkerName, testChunk)
		if err == nil {
			log.Printf("executeTestChunk(%q, %q) returned code %d in trial %d", funkerName, testChunk, code, i)
			return code, nil
		}
		if errorSeemsInteresting(err) || verbose {
			log.Printf("Error while calling executeTestChunk(%q, %q), will retry (trial %d): %v",
				funkerName, testChunk, i, err)
		}
		// TODO: non-constant sleep
		time.Sleep(defaultFunkerRetryDuration)
	}
	return 1, fmt.Errorf("could not call executeTestChunk(%q, %q) in %v", funkerName, testChunk, funkerRetryTimeout)
}

//  errorSeemsInteresting returns true if err does not seem about https://github.com/bfirsh/funker/issues/3
func errorSeemsInteresting(err error) bool {
	boringSubstrs := []string{"connection refused", "connection reset by peer", "no such host"}
	errS := err.Error()
	for _, boringS := range boringSubstrs {
		if strings.Contains(errS, boringS) {
			return false
		}
	}
	return true
}
