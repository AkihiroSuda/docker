package master

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

// Main is the entrypoint for master agent
func Main() error {
	log.Printf("Loading config from the environment")
	funkerName := os.Getenv("WORKER_SERVICE")
	if funkerName == "" {
		return errors.New("WORKER_SERVICE unset")
	}
	batchSizeS := os.Getenv("BATCH_SIZE")
	if batchSizeS == "" {
		return errors.New("BATCH_SIZE unset")
	}
	batchSize, err := strconv.Atoi(batchSizeS)
	if err != nil {
		return fmt.Errorf("bad BATCH_SIZE %s: %v", batchSizeS, err)
	}
	input := os.Getenv("INPUT")
	if input == "" {
		return errors.New("INPUT not set")
	}
	log.Printf("WORKER_SERVICE=%q, BATCH_SIZE=%d, INPUT=%q", funkerName, batchSize, input)
	log.Printf("Loading tests from %s", input)
	tests, err := loadTests(input)
	if err != nil {
		return err
	}
	testChunks := chunkStrings(tests, batchSize)
	log.Printf("Loaded %d tests (%d chunks)", len(tests), len(testChunks))
	return executeTests(funkerName, testChunks)
}

func loadTests(filename string) ([]string, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var tests []string
	for _, line := range strings.Split(string(b), "\n") {
		s := strings.TrimSpace(line)
		if s != "" {
			tests = append(tests, s)
		}
	}
	return tests, nil
}
