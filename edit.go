package main

import (
	"fmt"
)

// Levenshtein computes the levenshtein edit distance between two strings
// this is an optimized version from http://blog.softwx.net/2015/01/optimizing-damerau-levenshtein_15.html
func Levenshtein(a, b string) int {
	if a == "" {
		return len(b)
	} else if b == "" {
		return len(a)
	}

	// make the shorter string a can speed up by just running the inner loop
	// swap the two
	if len(a) > len(b) {
		a, b = b, a
	}

	aLen := len(a)
	bLen := len(b)

	// ignore suffix that are the same
	for aLen > 0 && a[aLen-1] == b[bLen-1] {
		aLen--
		bLen--
	}

	start := 0
	if a[0] == b[0] || aLen == 0 {
		// prefix common to both strings can be ignored
		for start < aLen && a[start] == b[start] {
			start++
		}
		// length of the part excluding common prefix and suffix
		aLen -= start
		bLen -= start
		// if all of shorter string matches prefix and/or suffix of longer string, then
		// edit distance is just the delete of additional characters present in longer string
		if aLen == 0 {
			return bLen
		}

		b = b[start : bLen+start]
	}

	v0 := make([]int, bLen)
	v2 := make([]int, bLen)

	for j := 0; j < bLen; j++ {
		v0[j] = j + 1
	}

	// the few commented out things here are for the transposition
	aChar := a[0]
	current := 0
	for i := 0; i < aLen; i++ {
		//prevaChar := aChar
		aChar = a[start+i]
		bChar := b[0]
		left := i
		current = i + 1
		//nextTransCost := 0
		for j := 0; j < bLen; j++ {
			above := current
			//thisTransCost := nextTransCost
			//nextTransCost = v2[j]
			current = left
			v2[j] = current
			left = v0[j]
			//prevbChar := bChar
			bChar = b[j]
			if aChar != bChar {
				if left < current {
					current = left
				}
				if above < current {
					current = above
				}
				current++
				/*
					if i != 0 && j != 0 && aChar == prevbChar && prevaChar == bChar {
						thisTransCost++
						if thisTransCost < current {
							current = thisTransCost
						}
					}
				*/
			}
			v0[j] = current
		}
	}
	return current
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Edit computes the levenshtein edit distance but fills an array as it goes
func Edit(a, b []rune) [][]int {

	// this maks a slice like [[][][]]
	// that can hold len(a) []
	// +1 so we can skip the top corner which is 0
	matrix := make([][]int, len(b)+1)

	// going across
	// this inserts the new slices
	// it also makes the first value of each of these 0-len(a)
	// [[0,0,0,0][1,0,0,0][2,0,0,0]]
	for i := 0; i < len(b)+1; i++ {
		matrix[i] = make([]int, len(a)+1)
		matrix[i][0] = i
	}

	// going down
	// this makes the first sub array filled to len(b)
	// [[0,1,2,3][1,0,0,0][2,0,0,0]]
	for j := 0; j < len(a)+1; j++ {
		matrix[0][j] = j
	}

	// fill in the rest of the matrix
	for i := 1; i < len(b)+1; i++ {
		for j := 1; j < len(a)+1; j++ {
			// if they match carry down the value from above to the left
			if b[i-1] == a[j-1] {
				matrix[i][j] = matrix[i-1][j-1]
			} else {
				// the adding of 1 here are the costs
				// the purpose of this is if you want to change the weight of insert delete etc..
				insertion := matrix[i-1][j] + 1
				deletion := matrix[i][j-1] + 1
				substitution := matrix[i-1][j-1] + 1
				matrix[i][j] = min(insertion, min(deletion, substitution))
			}
		}
	}
	return matrix
}

// PrettyPrint prints the matrix in a pretty format
func PrettyPrint(matrix [][]int, a, b string) {

	fmt.Printf("      ")
	for _, v := range a {
		fmt.Printf("%c  ", v)
	}
	fmt.Println()
	for i := range matrix {
		if i == 0 {
			fmt.Printf("   ")
		} else {
			fmt.Printf("%c  ", b[i-1])
		}
		for j := 0; j < len(matrix[0]); j++ {
			fmt.Printf("%d  ", matrix[i][j])
		}
		fmt.Println()
	}
}

// GenerateLevenshteinRules generates a list of paths to take based on the matrix
func GenerateLevenshteinRules(word, password []rune) [][]EditOp {
	matrix := Edit(word, password)

	paths := ReverseRecurse(matrix, len(matrix)-1, len(matrix[0])-1, 0)

	var finalPath [][]EditOp
	for _, path := range paths {
		if len(path) <= matrix[len(matrix)-1][len(matrix[0])-1] {
			finalPath = append(finalPath, path)
		}
	}

	return finalPath
}

// EditOp holds the edit operation and the position on the word and password it
// occured
type EditOp struct {
	Op   string // short for operation
	P    int    // short for password holds where the change is in the password
	Word int    // holds where the change is in the word
}

// these define max in sizes
const maxuint = ^uint(0)
const maxint = int(maxuint >> 1)

// ReverseRecurse walks a matrix backwards
func ReverseRecurse(matrix [][]int, i, j, pathLen int) [][]EditOp {
	// how this works
	// keeps calling itself until it reaches the edit of the matrix
	// then [][]EditOp with length one is returned
	// then it falls back through the calls to itself appending to
	// the list each time

	// called with matrix, len(matrix)-1, len(matrix[0])-1, 0
	//https://play.golang.org/p/b5N3Q1j1DZ

	// test if we are at the end
	// or if our path_len is longer than the minimum edit distance
	// path_len is how far back we have traversed so far
	if i == 0 && j == 0 || pathLen > matrix[len(matrix)-1][len(matrix[0])-1] {
		return make([][]EditOp, pathLen, pathLen)
	} else {

		cost := matrix[i][j]
		var edit [][]EditOp

		costMin := 0
		costInsert := maxint
		costDelete := maxint
		costEqualReplace := maxint
		// this is insert
		if i > 0 {
			costInsert = matrix[i-1][j]
		}
		// this is deleting on from the word
		if j > 0 {
			costDelete = matrix[i][j-1]
		}
		// replace or equal
		if i > 0 && j > 0 {
			costEqualReplace = matrix[i-1][j-1]
		}
		// choose the path of least resistence
		costMin = min(costInsert, min(costDelete, costEqualReplace))

		if costInsert == costMin {
			insertEdits := ReverseRecurse(matrix, i-1, j, pathLen+1)
			var temp []EditOp
			for _, path := range insertEdits {
				temp = append(path, EditOp{"insert", i - 1, j})
				edit = append(edit, temp)
			}
		}

		if costDelete == costMin {
			deleteEdits := ReverseRecurse(matrix, i, j-1, pathLen+1)
			var temp []EditOp
			for _, path := range deleteEdits {
				temp = append(path, EditOp{"delete", i, j - 1})
				edit = append(edit, temp)
			}
		}
		if costEqualReplace == costMin {
			if costEqualReplace == cost {
				equalPaths := ReverseRecurse(matrix, i-1, j-1, pathLen)
				for _, path := range equalPaths {
					edit = append(edit, path)
				}
			} else {
				replacePaths := ReverseRecurse(matrix, i-1, j-1, pathLen+1)
				for _, path := range replacePaths {
					path = append(path, EditOp{"replace", i - 1, j - 1})
					edit = append(edit, path)
				}
			}
		}

		return edit
	}
}
