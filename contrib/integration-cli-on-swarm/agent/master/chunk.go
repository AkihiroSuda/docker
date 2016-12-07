package master

import (
	"math/rand"
)

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
		i := r.Intn(hintChunkSize) + 1
		if i > len(rest) {
			i = len(rest)
		}
		chunk := rest[0:i]
		rest = rest[i:]
		result = append(result, chunk)
	}
	return result
}
