package master

import (
	"math/rand"
)

// chunkStrings chunks the string slice
func chunkStrings(x []string, numChunks int) [][]string {
	var result [][]string
	chunkSize := (len(x) + numChunks - 1) / numChunks
	for i := 0; i < len(x); i += chunkSize {
		ub := i + chunkSize
		if ub > len(x) {
			ub = len(x)
		}
		result = append(result, x[i:ub])
	}
	return result
}

// chunkStringsRandom chunks the string slice.
// The number of chunks is likely to be close to hintNumChunks.
func chunkStringsRandom(x []string, hintNumChunks int, seed int64) [][]string {
	var result [][]string
	r := rand.New(rand.NewSource(seed))
	hintChunkSize := len(x) * 2 / hintNumChunks
	if hintChunkSize == 0 {
		hintChunkSize = len(x)
	}
	rest := x
	for len(rest) > 0 {
		ub := r.Intn(hintChunkSize) + 1
		if ub > len(rest) {
			ub = len(rest)
		}
		chunk := rest[0:ub]
		rest = rest[ub:]
		result = append(result, chunk)
	}
	return result
}

// shuffleStrings shuffles strings
func shuffleStrings(x []string, seed int64) {
	r := rand.New(rand.NewSource(seed))
	for i := range x {
		j := r.Intn(i + 1)
		x[i], x[j] = x[j], x[i]
	}
}
