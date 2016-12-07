package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bfirsh/funker-go"
	"gopkg.in/yaml.v2"
)

const (
	// defaultFunkerRetryTimeout is for the issue https://github.com/bfirsh/funker/issues/3
	// When all the funker replicas are busy in their own job, we cannot connect to funker.
	defaultFunkerRetryTimeout = 1 * time.Hour
)

type config struct {
	Tests []string `yaml:"tests,omitempty"`
}

func main() {
	funkerName := os.Getenv("WORKER_SERVICE")
	if funkerName == "" {
		fmt.Fprintf(os.Stderr, "WORKER_SERVICE unset\n")
		os.Exit(1)
	}
	// TODO: support alternative config? (how do we inject that to this container? env var?)
	defaultConfigFile := "config.yaml"
	if err := xmain(funkerName, defaultConfigFile); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func xmain(funkerName, configFile string) error {
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	var c config
	if err = yaml.Unmarshal(bytes, &c); err != nil {
		return err
	}
	return executeTests(funkerName, c.Tests)
}

func executeTests(funkerName string, tests []string) error {
	begin := time.Now()
	log.Printf("Executing %d tests in parallel, using %s", len(tests), funkerName)
	var wg sync.WaitGroup
	var passed, failed uint32
	for _, test := range tests {
		wg.Add(1)
		go func(test string) {
			defer wg.Done()
			log.Printf("Executing %q", test)
			testBegin := time.Now()
			code, err := executeTestWithRetry(funkerName, test, defaultFunkerRetryTimeout)
			if err != nil {
				log.Printf("Error while executing %q: %v",
					test, err)
				atomic.AddUint32(&failed, 1)
			} else {
				if code == 0 {
					atomic.AddUint32(&passed, 1)
				} else {
					atomic.AddUint32(&failed, 1)
				}
				log.Printf("Finished %q in %s, code=%d.",
					test, time.Now().Sub(testBegin), code)
			}
		}(test)
	}
	wg.Wait()
	log.Printf("Executed %d tests in %s. PASS: %d, FAIL: %d.",
		len(tests), time.Now().Sub(begin), passed, failed)
	if failed > 0 {
		return fmt.Errorf("%d tests failed", failed)
	}
	return nil
}

func executeTest(funkerName string, test string) (int, error) {
	log.Printf("[FUNKER] Calling funker %s(%q)", funkerName, test)
	ret, err := funker.Call(funkerName, test)
	log.Printf("[FUNKER] Called funker %s(%q)=(%v, %v)", funkerName, test, ret, err)
	if err != nil {
		return 1, err
	}
	code, ok := ret.(float64)
	if !ok {
		return 1, fmt.Errorf("unexpected result from %s(%q): %v",
			funkerName, test, ret)
	}
	return int(code), nil
}

func executeTestWithRetry(funkerName string, test string, funkerRetryTimeout time.Duration) (int, error) {
	begin := time.Now()
	for i := 0; time.Now().Sub(begin) < funkerRetryTimeout; i++ {
		code, err := executeTest(funkerName, test)
		if err == nil {
			return code, nil
		}
		log.Printf("Error while calling %s(%q), will retry (%d): %v",
			funkerName, test, i, err)
		// TODO: if err is not about https://github.com/bfirsh/funker/issues/3 ,
		// we should return err immediately
		// TODO: non-constant sleep
		time.Sleep(3 * time.Second)
	}
	return 1, fmt.Errorf("could not call %s(%q) in %v", funkerName, test, funkerRetryTimeout)
}
