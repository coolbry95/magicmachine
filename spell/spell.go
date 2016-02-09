// Package spell provides spell checking based off of the spell checker found at http://blog.faroo.com/2012/06/07/improved-edit-distance-based-spelling-correction/
package spell

import (
	"bufio"
	"fmt"
	"io"
	"sync"
	"encoding/gob"

	"github.com/coolbry95/magicmachine/util"
)

// Speller provides a basic interface for spell checking
type Speller interface {
	Suggest(string) []string
	// dont really think it needs Replace
	Replace(string, string)
}

// Term contains the count of how many times a word was seen and the suggestions for the word
type Term struct {
	Count       int      `json:"count"`
	Suggestions []string `json:"suggestions"`
}

// Model holds the data for the spell checker
type Model struct {
	// really should orcrestrate with channels
	Data map[string]*Term `json:"data"`
	// amount of times to see term before adding it
	Threshold int `json:"threshold"`
	// edit distance depth
	Depth int `json:"depth"`
	// maximum dictionary term length
	// dont really know what this is used for
	// 224 symspell.cs ??
	Max int `json:"max"`
	m   sync.Mutex
}

// NewModel returns a Model with default parameters
func NewModel() *Model {
	return &Model{
		Data:      make(map[string]*Term),
		Threshold: 1,
		Depth:     100,
		//depth:     3,
		// this is something like max int size i think
		Max: 10000,
	}
}

// LoadWordList loads a wordlist and puts the data into the Model
func (m *Model) LoadWordList(file io.Reader) {
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		m.CreateEntry(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("err scanning words")
	}
}

// SaveWordList saves a wordlist to disc
func (m *Model) SaveWordList(file io.Writer) {
	m.m.Lock()
	err := gob.NewEncoder(file).Encode(m)
	if err != nil {
		fmt.Println(err)
	}
	m.m.Unlock()
}

// LoadSavedWordList loads a wordlist that is saved on disc
func LoadSavedWordList(file io.Reader) *Model {
	m := NewModel()
	err := gob.NewDecoder(file).Decode(m)
	if err != nil {
		fmt.Println(err)
	}
	return m
}

// Replace is a wrapper to satisfy the Speller interface
func (m *Model) Replace(mispelled, correct string) {
	// just a wrapper to satisfy the Speller interface
	// this method is based off of the aspell package replace
	// just trying to adhere to that
	m.CreateEntry(correct)
}

// CreateEntry adds an entry to the model
func (m *Model) CreateEntry(word string) {
	// make this non exported?
	m.m.Lock()
	if v, ok := m.Data[word]; ok {
		// this works becuase v is a pointer
		// eg it points to that point in memory its not a value
		v.Count++
	} else {
		// can probably guess the size of suggestions based on length
		// eg len(word)
		m.Data[word] = &Term{1, make([]string, 0)}
	}

	// set the max
	if m.Data[word].Count > m.Max {
		m.Max = m.Data[word].Count
	}

	// how many times how we seen this term
	if m.Data[word].Count == m.Threshold {
		m.m.Unlock()
		// create suggestions here
		m.createSuggestions(word)
	} else {
		m.m.Unlock()
	}
}

// this needs to uniq the values returned
func (m *Model) createSuggestions(word string) {
	m.m.Lock()
	// get all of the edits
	edits := Edits([]rune(word), 0, m.Depth)

	// for each delete add the word it derived from to the suggestions for it
	for _, val := range edits {
		// make this 0 instead of one??
		// because it is an edit not a word
		m.Data[val] = &Term{1, make([]string, 0)}
		m.Data[val].Suggestions = append(m.Data[val].Suggestions, word)
	}
	m.m.Unlock()

	//m.Data[word].Suggestions = append(m.Data[word].Suggestions, edits...)
}

// with pointer make a method on type so
// func (deletes *type) Edits(..) {}
/*
func Edit(word []rune, editDistance int, depth int, deletes *[]string) *[]string {
	// increase how far we have gone
	editDistance++
	if len(word) > 1 {
		// create an edit for the length of the word
		for i := 0; i < len(word); i++ {
			// append the one letter removed
			*deletes = append(*deletes, string(word[:i])+string(word[i+1:]))
			// if we havent hit how many edits we want to do
			if editDistance < depth {
				Edits(word[i+1:], editDistance, depth, deletes)
			}
		}
	}
	return deletes
}
*/

// Edits creates a list of all deletes from a word up to editDistance
func Edits(word []rune, editDistance int, depth int) []string {

	// this needs refactoring to be more efficient
	// also is there a way to get rid of stuff?
	// symspell uses a hashset- a list of unique values

	// this conatains our edits
	deletes := make([]string, 0, len(word))
	// increase how far we have gone
	editDistance++
	if len(word) > 1 {
		// create an edit for the length of the word
		for i := 0; i < len(word); i++ {
			// append the one letter removed
			deletes = append(deletes, string(word[:i])+string(word[i+1:]))
			// if we havent hit how many edits we want to do
			if editDistance < depth {
				// stuff is a temp variable that holds the edits
				// what if we do &slice??
				// https://play.golang.org/p/WAEIzRgSNB
				// mine
				// https://play.golang.org/p/AuEXks0ZyD
				//https://play.golang.org/p/MvieVStQst
				stuff := Edits(word[i+1:], editDistance, depth)
				// this appends the edits to delete
				// this is necessary to have the delete outside the scope of this function
				for _, v := range stuff {
					deletes = append(deletes, v)
				}
			}
		}
	}
	return deletes
}

// Suggest is a wrapper for the Speller interface
func (m *Model) Suggest(word string) []string {
	return m.Suggestion(word, 10)
}

// Suggestion checks the spelling of a word
func (m *Model) Suggestion(word string, editDistanceMax int) []string {
// and sort based on likelyness
// change suggestions to a *Term will have to change slicing operations
	m.m.Lock()
	wordRune := []rune(word)
	if len(wordRune)-editDistanceMax > m.Max {
		return []string{}
	}

	hashset1 := util.NewHash()
	hashset2 := util.NewHash()

	candidates := []string{}
	candidates = append(candidates, word)
	//hashset1.Add(word)
	suggestions := []string{}

	//fmt.Println(len(candidates))

	for len(candidates) > 0 {
		// this does not change the slice the undlerying array is still the same
		//candidate, candidates := candidates[0], candidates[1:]
		// can do this https://play.golang.org/p/pyum8Afrin
		candidate := candidates[0]
		candidates = append(candidates[:0], candidates[1:]...)
		candidateRune := []rune(candidate)

		if temp, ok := m.Data[candidate]; ok {
			if temp.Count > 0 && hashset2.Add(candidate) {
				//fmt.Println("ok added to suggestions")
				suggestions = append(suggestions, temp.Suggestions[:]...)
			}

			for _, suggestion := range suggestions {
				suggestionRune := []rune(suggestion)

				if hashset2.Add(suggestion) {

					// check the beginning and back of the suggestion
					prefix := 0
					suffix := 0
					for (prefix < len(suggestionRune)) && (prefix < len(wordRune)) &&
						(suggestionRune[prefix] == wordRune[prefix]) {

						prefix++
					}
					for (suffix < len(suggestionRune)-prefix) && (suffix < len(wordRune)-prefix) &&
						(suggestionRune[len(suggestionRune)-suffix-prefix-1] == wordRune[len(wordRune)-suffix-prefix-1]) {

						suffix++
					}

					distance := 0
					if suggestion != word {
						if len(suggestionRune) == len(wordRune) {
							if (prefix > 0) || (suffix > 0) {
								distance = len(wordRune) - len(candidateRune)
							}
						} else if len(word) == len(candidate) {
							if (prefix > 0) || (suffix > 0) {
								distance = len(suggestionRune) - len(candidateRune)
							}
						} else {
							if (prefix > 0) || (suffix > 0) {

								// can used optimized one here from edit.go
								distance = DamerauLevenshteinDistance(
									suggestionRune[prefix:len(suggestionRune)-suffix],
									wordRune[prefix:len(wordRune)-suffix])
							} else {
								if (prefix > 0) || (suffix > 0) {
									distance = DamerauLevenshteinDistance(suggestionRune, wordRune)
								}
							}
						}
					}

					// remove for verbose missing here

					if distance <= editDistanceMax {
						if val, ok := m.Data[suggestion]; ok {
							suggestions = append(suggestions, val.Suggestions[:]...)
						}
					}
				}
			}
		}

		if len(wordRune)-len(candidateRune) < editDistanceMax {
			// verbose stuff here missing

			// this does the same thing as Edits()
			for i := 0; i < len(candidateRune); i++ {
				delete := string(wordRune[:i]) + string(wordRune[i+1:])
				if hashset1.Add(delete) {
					candidates = append(candidates, delete)
				}
			}
		}
	}
	m.m.Unlock()

	temp := []string{}
	a := make(map[string]struct{})
	for _, val := range suggestions {
		a[val] = struct{}{}
	}

	for k := range a {
		temp = append(temp, k)
	}

	return temp

}

// DamerauLevenshteinDistance returns the DamerauLevenshteinDistance
func DamerauLevenshteinDistance(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	} else if len(b) == 0 {
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

		// faster than b[start+j] in inner loop below
		// this is if there is only one char different
		//if bLen == 1 {
		//  b = string(b[start])
		//} else {
		b = b[start : bLen+start]
		//}
	}

	v0 := make([]int, bLen)
	v2 := make([]int, bLen)

	for j := 0; j < bLen; j++ {
		v0[j] = j + 1
	}

	aChar := a[0]
	current := 0
	for i := 0; i < aLen; i++ {
		prevaChar := aChar
		aChar = a[start+i]
		bChar := b[0]
		left := i
		current = i + 1
		nextTransCost := 0
		for j := 0; j < bLen; j++ {
			above := current
			thisTransCost := nextTransCost
			nextTransCost = v2[j]
			current = left
			v2[j] = current
			left = v0[j]
			prevbChar := bChar
			bChar = b[j]
			if aChar != bChar {
				if left < current {
					current = left
				}
				if above < current {
					current = above
				}
				current++
				if i != 0 && j != 0 && aChar == prevbChar && prevaChar == bChar {
					thisTransCost++
					if thisTransCost < current {
						current = thisTransCost
					}
				}
			}
			v0[j] = current
		}
	}
	return current
}

/*
// doesnt work
func DamerauLevenshteinDistance(a, b []rune) int {
	matrix := make([][]int, len(a)+1)

	for i := 0; i < len(a)+1; i++ {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := 0; j < len(b)+1; j++ {
		matrix[0][j] = j
	}

	for i := 1; i < len(a)+1; i++ {
		for j := 1; j < len(b)+1; j++ {
			if a[i-1] == b[j-1] {
				matrix[i][j] = matrix[i-1][j-1]
			} else {
				insertion := matrix[i-1][j] + 1
				deletion := matrix[i][j-1] + 1
				substitution := matrix[i-1][j-1] + 1

				if i > 0 && j > 0 {
					// index error here
					transposition := matrix[i-2][j-2] + 1
					matrix[i][j] =
						mini(insertion, deletion, substitution, transposition)
				} else {
					matrix[i][j] = mini(insertion, deletion, substitution)
				}
			}
		}
	}
	return matrix[0][len(matrix[0])-1]
}
*/

// mini returns the lowest value of ints
func mini(a ...int) int {
	lowest := a[0]
	for _, val := range a[1:] {
		if val < lowest {
			lowest = val
		}
	}
	return lowest
}

// this trains from and io.Reader
/*
func trainword(Data io.Reader) map[string]int {
	NWORDS := make(map[string]int)

	// https://github.com/wolfgarbe/symspell/blob/master/symspell.cs
	// regex takex from there
	// note different from Norvigs
	// that guy is a jack ass that regex makes no sense only matches
	// ]3 or something like that
	// \w is only for ASCII
	// this one is closer to what \w but without '_'
	//`[\p{L}\p{N}]+`
	// bufio.scanwords?
	pattern := regexp.MustCompile(`[\w-[\d_]]+`)
	scanner := bufio.NewScanner(Data)
	for scanner.Scan() {
		w := pattern.FindAllString(strings.ToLower(scanner.Text()), -1)
		NWORDS[w]++
	}

	return NWORDS
}
*/
