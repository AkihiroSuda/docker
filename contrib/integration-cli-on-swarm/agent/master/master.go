package master

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Main is the entrypoint for master agent.
// TODO: should we use flags rather than os.Getenv?
func Main() error {
	log.Printf("Loading config from the environment")
	funkerName := os.Getenv("WORKER_SERVICE")
	if funkerName == "" {
		return errors.New("WORKER_SERVICE unset")
	}
	chunksS := os.Getenv("CHUNKS")
	if chunksS == "" {
		return errors.New("CHUNKS unset")
	}
	chunks, err := strconv.Atoi(chunksS)
	if err != nil {
		return fmt.Errorf("bad CHUNKS %s: %v", chunksS, err)
	}
	input := os.Getenv("INPUT")
	if input == "" {
		return errors.New("INPUT not set")
	}
	randSeed := time.Now().UnixNano()
	randSeedS := os.Getenv("RAND_SEED")
	if randSeedS != "" {
		randSeed, err = strconv.ParseInt(randSeedS, 10, 64)
		if err != nil {
			return fmt.Errorf("bad RAND_SEED %s: %v", randSeedS, err)
		}
	}
	log.Printf("WORKER_SERVICE=%q, CHUNKS=%d, INPUT=%q, RAND_SEED=%s",
		funkerName, chunks, input, randSeedS)
	log.Printf("Loading tests from %s", input)
	tests, err := loadTests(input)
	if err != nil {
		return err
	}
	var testChunks [][]string
	testChunks = chunkStringsRandom(tests, chunks, randSeed)
	// len(testChunks) is designed to be close to chunks, but not always equal
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
