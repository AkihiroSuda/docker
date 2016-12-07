package master

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func testChunkStringsRandom(t *testing.T, inputLen, hintNumChunks int, seed int64) {
	t.Logf("inputLen=%d, hintNumChunks=%d, seed=%d", inputLen, hintNumChunks, seed)
	input := []string{}
	for i := 0; i < inputLen; i++ {
		input = append(input, fmt.Sprintf("s%d", i))
	}
	result := chunkStringsRandom(input, hintNumChunks, seed)
	t.Logf("result has %d chunks", len(result))
	inputReconstructedFromResult := []string{}
	for i, chunk := range result {
		t.Logf("chunk %d has %d elements", i, len(chunk))
		inputReconstructedFromResult = append(inputReconstructedFromResult, chunk...)
	}
	if !reflect.DeepEqual(input, inputReconstructedFromResult) {
		t.Fatal("input != inputReconstructedFromResult")
	}
}

func TestChunkStringsRandom_4_4(t *testing.T) {
	testChunkStringsRandom(t, 4, 4, time.Now().UnixNano())
}

func TestChunkStringsRandom_4_1(t *testing.T) {
	testChunkStringsRandom(t, 4, 1, time.Now().UnixNano())
}

func TestChunkStringsRandom_1000_8(t *testing.T) {
	testChunkStringsRandom(t, 1000, 8, time.Now().UnixNano())
}
