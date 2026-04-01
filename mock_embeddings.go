package main

import "openai-mokku/api"

// defaultEmbeddingDimensions is the number of dimensions used when none is specified.
const defaultEmbeddingDimensions = 1536

// normalizeInputStrings normalizes the embedding input union type to a []string.
func normalizeInputStrings(input api.CreateEmbeddingRequestInput) []string {
	if input.IsString() {
		return []string{input.String}
	}
	return input.StringArray
}

// generateVector generates a deterministic pseudo-random unit vector using a linear congruential generator.
// The returned slice has exactly `dimensions` elements with values in [-1, 1].
func generateVector(text string, dimensions int) []float64 {
	vector := make([]float64, dimensions)
	seed := int64(0)
	for i, c := range text {
		seed = seed*31 + int64(c) + int64(i)*41
	}
	for i := 0; i < dimensions; i++ {
		seed = seed*1103515245 + 12345
		// lower 31 bits → [0, 1) → [-1, 1)
		value := float64(seed&0x7FFFFFFF)/float64(0x7FFFFFFF)*2 - 1
		vector[i] = value
	}
	return vector
}
