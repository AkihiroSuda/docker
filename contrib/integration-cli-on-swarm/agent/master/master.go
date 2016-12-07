package master

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

// Main is the entrypoint for master agent.
func Main() error {
	workerService := flag.String("worker-service", "", "Name of worker service")
	chunks := flag.Int("chunks", 0, "Number of chunks")
	input := flag.String("input", "", "Path to input file")
	randSeed := flag.Int64("rand-seed", int64(0), "Random seed (0 is treated as the current time)")
	shuffle := flag.Bool("shuffle", false, "Shuffle the input so as to mitigate makespan nonuniformity")
	randomChunking := flag.Bool("random-chunking", false, "Randomize the chunking size so as to mitigate makespan nonuniformity")
	flag.Parse()
	if *workerService == "" {
		return errors.New("worker-service unset")
	}
	if *chunks == 0 {
		return errors.New("chunks unset")
	}
	if *input == "" {
		return errors.New("input unset")
	}
	if *randSeed == int64(0) {
		*randSeed = time.Now().UnixNano()
	}
	tests, err := loadTests(*input)
	if err != nil {
		return err
	}
	testChunks := chunkTests(tests, *chunks, *shuffle, *randomChunking, *randSeed)
	log.Printf("Loaded %d tests (%d chunks)", len(tests), len(testChunks))
	return executeTests(*workerService, testChunks)
}

func chunkTests(tests []string, numChunks int, shuffle, randomChunking bool, randSeed int64) [][]string {
	// shuffling (experimental) mitigates makespan nonuniformity
	// Not sure this can cause some locality problem..
	if shuffle {
		shuffleStrings(tests, randSeed)
	}
	var testChunks [][]string
	if randomChunking {
		// random chunking (experimental) mitigates makespan nonuniformity
		// len(testChunks) is designed to be close to chunks, but not always equal
		testChunks = chunkStringsRandom(tests, numChunks, randSeed)
	} else {
		testChunks = chunkStrings(tests, numChunks)
	}
	return testChunks
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
