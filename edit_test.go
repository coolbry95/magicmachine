package main

import (
	"testing"
)

func TestLevenshtein(t *testing.T) {

	var words = []struct{
		first string
		second string
		dist int
	}{
		{"california", "California", 1},
		{"rules", "ralse", 3},
	}

	for _, word := range words {
		out := Levenshtein(word.first, word.second)
		if out != word.dist {
			t.Errorf("should be %d, got %d", word.dist, out)
		}
	}

}

var result int

func BenchmarkLevenshtein(b *testing.B) {

	var r int

	for n:= 0; n < b.N; n++ {
		r = Levenshtein("asdfadsf", "lkjlkjhjhlkjl")
	}

	result = r
}

func TestLevenshteinNew(t *testing.T) {
	var words = []struct{
		first string
		second string
		dist int
	}{
		{"california", "California", 1},
		{"rules", "ralse", 3},
	}

	for _, word := range words {
		out := LevenshteinNew(word.first, word.second)
		if out != word.dist {
			t.Errorf("should be %d, got %d", word.dist, out)
		}
	}

}

func BenchmarkLevenshteinNew(b *testing.B) {

	var r int

	for n:= 0; n < b.N; n++ {
		r = LevenshteinNew("asdfadsf", "lkjlkjhjhlkjl")
	}

	result = r
}

func Testmin(t *testing.T) {

}

func TestEdit(t *testing.T) {

}

func TestPrettyPrint(t *testing.T) {
}

func TestGenerateLevenshteinRules(t *testing.T) {

}
func TestReverseRecurse(t *testing.T) {
}

