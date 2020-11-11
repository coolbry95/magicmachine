// Package util provides a basic unique string array by using a map
package spell

// Hashset is the basic interface for a unique string array
type Hashset interface {
	Add(string) bool
	Remove(string) bool
	Exists(string) bool
	Len() int
}

// Hash is a unique array of strings
type Hash map[string]struct{}

// NewHash returns a new Hash
// change name conflicts with sha1
func NewHash() Hash {
	return make(map[string]struct{})
}

// Add adds a new string to the array
func (h Hash) Add(in string) bool {
	if _, ok := h[in]; ok {
		// already in the set
		return false
	}
	// not in the set
	h[in] = struct{}{}
	return true
}

// Remove removes a string from the array
func (h Hash) Remove(in string) bool {
	if _, ok := h[in]; ok {
		// exists
		delete(h, in)
		return true
	}
	return false
}

// Exists checks if a string exists
func (h Hash) Exists(in string) bool {
	if _, ok := h[in]; ok {
		return true
	}
	return false
}

// Len returns the length of the list
func (h Hash) Len() int {
	return len(h)
}
