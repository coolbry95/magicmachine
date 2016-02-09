// Package enchant provides a binding to the enchant spell checking library.
package enchant

/*
#cgo CFLAGS: -O2
#cgo LDFLAGS: -lenchant
#include <stdlib.h>
#include <sys/types.h>
#include "enchant/enchant.h"

static char* getString(char **c, int i) {
	return c[i];
}
*/
import "C"

import (
	"unsafe"
)

// Version provides the version of the enchant library
func Version() string {
	return C.GoString(C.enchant_get_version())
}

/*
// this is complicated
// says *[0]byte is the second argument type in enchant_broker_describe
func (e *Enchant) Describe() {
	C.enchant_broker_describe(e.broker, go_callback, nil)
}

//export go_callback
func go_callback(cName *C.char, cDesc *C.char, cFile *C.char, cUserData unsafe.Pointer) {

	fmt.Println(C.GoString(cName))
}
*/

// Enchant wraps the basic C structures from the enchant library
type Enchant struct {
	broker *C.EnchantBroker
	dict   *C.EnchantDict
}

// NewEnchant makes a new Enchant struct
func NewEnchant() (*Enchant, error) {
	broker := C.enchant_broker_init()
	// check for a broker error here
	return &Enchant{broker, nil}, nil
}

// Delete deletes the Enchant struct
func (e *Enchant) Delete() {
	if e.dict != nil {
		C.enchant_broker_free_dict(e.broker, e.dict)
	}
	C.enchant_broker_free(e.broker)
}

// LoadDict loads a dictionary for use
func (e *Enchant) LoadDict(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	if e.dict != nil {
		C.enchant_broker_free_dict(e.broker, e.dict)
	}

	e.dict = C.enchant_broker_request_dict(e.broker, cName)
}

// LoadPersonalDict loads a dictionary that is not a default provided one
func (e *Enchant) LoadPersonalDict(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	if e.dict != nil {
		C.enchant_broker_free_dict(e.broker, e.dict)
	}

	e.dict = C.enchant_broker_request_pwl_dict(e.broker, cName)
}

// DictExists test if a dictionary exists
// checks to see if you have en_GB or en_US
func (e *Enchant) DictExists(name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	return C.enchant_broker_dict_exists(e.broker, cName) > 0
}

// BrokerOrdering sets the ordering for the brokers used.
// format is in a comma seperated list eg. "aspell,myspell"
func (e *Enchant) BrokerOrdering(lang string, name string) {
	if lang == "" || name == "" {
		return
	}

	cLang := C.CString(lang)
	defer C.free(unsafe.Pointer(cLang))
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	C.enchant_broker_set_ordering(e.broker, cLang, cName)
}

// Error returns the error from the broker
func (e *Enchant) Error() string {
	// redo this
	// so it calls dict and broker or something
	// do we need to do something with cstr??
	// like free it
	cstr := C.enchant_broker_get_error(e.broker)
	err := C.GoString(cstr)

	return err
}

func (e *Enchant) GetParam() {
	// enchant_broker_get_param
}

func (e *Enchant) SetParam() {
	// enchant_broker_set_param
}

// DictCheck checks if a word is in the dictionary
func (e *Enchant) DictCheck(word string) bool {
	// check if the word is in the dictionary
	if len(word) == 0 {
		// maybe make it false
		return true
	}

	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))

	size := uintptr(len(word))
	s := (*C.ssize_t)(unsafe.Pointer(&size))

	return C.enchant_dict_check(e.dict, cWord, *s) == 0
}

// Suggest performs the spell checking on a word
func (e *Enchant) Suggest(word string) []string {
	if len(word) == 0 {
		return nil
	}

	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))

	size := uintptr(len(word))
	s := (*C.ssize_t)(unsafe.Pointer(&size))

	var n int
	nSugg := uintptr(n)
	ns := (*C.size_t)(unsafe.Pointer(&nSugg))

	// ns will be modified for how many suggestions there are
	response := C.enchant_dict_suggest(e.dict, cWord, *s, ns)

	var suggestions []string
	for i := 0; i < int(*ns); i++ {
		ci := C.int(i)
		suggestions = append(suggestions, C.GoString(C.getString(response, ci)))
	}

	return suggestions
}

func (e *Enchant) Replace(word, wordNew string) {
	// does the dictionary need to be loaded here?
	cWord := C.CString(word)
	defer C.free(unsafe.Pointer(cWord))

	cWordNew := C.CString(wordNew)
	defer C.free(unsafe.Pointer(cWordNew))

	cWordLen := uintptr(len(word))
	s := C.ssize_t(cWordLen)

	cWordLenNew := uintptr(len(wordNew))
	sNew := C.ssize_t(cWordLenNew)

	C.enchant_dict_store_replacement(e.dict, cWord, s, cWordNew, sNew)
}
