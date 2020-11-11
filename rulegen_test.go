package main

import (
	"github.com/coolbry95/edit/spell"
	"testing"
)

// test in order that function is called

//func (h hashcatRulePrint) String() string {
//}

var model *spell.Model

var testWords = []string{
	"password",
	"test",
	"tesing",
}

func init() {
	model = spell.NewModel()

	for _, word := range testWords {
		model.CreateEntry(word)
	}
}

func TestprintRules(t *Testing.t) {
}

func TestanalyzePassword(t *Testing.t) {
}

func TestgenerateHashcatRules(t *Testing.t) {
}

func TestgenerateWords(t *Testing.t) []Word {
}

func TestgenerateSimpleWords(t *Testing.t) {
}

func TestgenerateAdvancedWords(t *Testing.t) {
}

func TestRuleWorks(t *Testing.t) {
}

func TestSimpleHashcatRules(t *Testing.t) {
}

func TestAdvancedHashcatRules(t *Testing.t) {
}

func TestcheckReversiblePassword(t *Testing.t) {
}

func TestprintRules(t *Testing.t) {
}
