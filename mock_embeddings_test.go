package main

import (
	"testing"

	"openai-mokku/api"
)

// --- generateVector ---

func TestGenerateVector_ReturnsDefaultDimensions(t *testing.T) {
	// Given: default dimension count
	// When
	got := generateVector("hello world", defaultEmbeddingDimensions)
	// Then
	if len(got) != defaultEmbeddingDimensions {
		t.Errorf("expected %d dimensions, got %d", defaultEmbeddingDimensions, len(got))
	}
}

func TestGenerateVector_ReturnsExactDimensions(t *testing.T) {
	// Given: explicit small dimension
	// When
	got := generateVector("hello", 256)
	// Then
	if len(got) != 256 {
		t.Errorf("expected 256 dimensions, got %d", len(got))
	}
}

func TestGenerateVector_IsDeterministic(t *testing.T) {
	// Given: same text and dimensions
	text := "deterministic test string"
	dims := 64
	// When: called twice
	first := generateVector(text, dims)
	second := generateVector(text, dims)
	// Then: results are identical
	if len(first) != len(second) {
		t.Fatalf("length mismatch: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("vector not deterministic at index %d: %v vs %v", i, first[i], second[i])
			return
		}
	}
}

func TestGenerateVector_DifferentInputsDifferentVectors(t *testing.T) {
	// Given: two different input texts
	v1 := generateVector("hello", 64)
	v2 := generateVector("world", 64)
	// Then: at least one element differs
	different := false
	for i := range v1 {
		if v1[i] != v2[i] {
			different = true
			break
		}
	}
	if !different {
		t.Error("expected different vectors for different inputs")
	}
}

func TestGenerateVector_AllValuesInNormalizedRange(t *testing.T) {
	// Given: any text
	got := generateVector("range check", 128)
	// Then: all values are in [-1, 1]
	for i, v := range got {
		if v < -1.0 || v > 1.0 {
			t.Errorf("value at index %d out of range [-1, 1]: %v", i, v)
		}
	}
}

func TestGenerateVector_SmallDimensions_ReturnsExactCount(t *testing.T) {
	// Given: very small dimension (edge case)
	got := generateVector("edge case", 1)
	// Then
	if len(got) != 1 {
		t.Errorf("expected 1 dimension, got %d", len(got))
	}
}

// --- normalizeInputStrings ---

func TestNormalizeInputStrings_SingleString_ReturnsSingleElement(t *testing.T) {
	// Given: a single string input
	// When
	got := normalizeInputStrings(api.NewStringCreateEmbeddingRequestInput("hello"))
	// Then
	if len(got) != 1 {
		t.Fatalf("expected 1 element, got %d", len(got))
	}
	if got[0] != "hello" {
		t.Errorf("expected 'hello', got %q", got[0])
	}
}

func TestNormalizeInputStrings_StringSlice_ReturnsSameElements(t *testing.T) {
	// Given: a slice of strings
	input := []string{"alpha", "beta", "gamma"}
	// When
	got := normalizeInputStrings(api.NewStringArrayCreateEmbeddingRequestInput(input))
	// Then
	if len(got) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(got))
	}
	for i, want := range input {
		if got[i] != want {
			t.Errorf("index %d: expected %q, got %q", i, want, got[i])
		}
	}
}

func TestNormalizeInputStrings_EmptyStringSlice_ReturnsEmpty(t *testing.T) {
	// Given: empty string slice
	// When
	got := normalizeInputStrings(api.NewStringArrayCreateEmbeddingRequestInput([]string{}))
	// Then
	if len(got) != 0 {
		t.Errorf("expected empty result, got %d elements", len(got))
	}
}
