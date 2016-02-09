// rulegen reverses passwords to source words
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/coolbry95/magicmachine/enchant"
	"github.com/coolbry95/magicmachine/spell"
	"github.com/coolbry95/magicmachine/util"
	"github.com/coolbry95/passutils/ruleprocessor/rules"
)

const usage = `
./magicmachine
`

var (
	// word generation tuning
	maxWordDist *int
	maxWords    *int
	moreWords   *bool
	simpleWords *bool

	// rule generation finetuning
	maxRuleLen  *int
	maxRules    *int
	moreRules   *bool
	simpleRules *bool
	bruteRules  *bool

	// threads
	threads *int

	// out file basename
	basename *string

	// Debugging options
	verbose   *bool
	debug     *bool
	wordDebug *string
	quiet     *bool

	// engine to use
	engine *string

	// process dictionary with special engine
	process    *string
	processOut *string

	// use already processed dictionary with special engine
	processed *string

	// diction
	specialDict *string
)

func main() {
	// we do this so we can skip os.Args[1] which is the list of passwords
	flags := flag.NewFlagSet("magicmachine", flag.ExitOnError)

	// rule generation finetuning
	maxWordDist = flags.Int("maxwordist", 10, "max word distance")
	maxWords = flags.Int("maxwords", 5, "max words")
	moreWords = flags.Bool("morewords", false, "more words")
	simpleWords = flags.Bool("simplewords", false, "simple words")

	// threads
	threads = flags.Int("threads", runtime.NumCPU(), "number of threads to use default max CPUS")

	// out file basename
	basename = flags.String("basename", "analysis", "basename for out files")

	// word generation finetuning
	maxRuleLen = flags.Int("maxrulelen", 15, "max rule length")
	maxRules = flags.Int("maxrules", 5, "max rules")
	moreRules = flags.Bool("morerules", false, "more rules")
	simpleRules = flags.Bool("simplerules", false, "simple rules")
	bruteRules = flags.Bool("bruterules", false, "brute rules")

	// debugging
	verbose = flags.Bool("verbose", false, "verbose")
	debug = flags.Bool("debug", false, "debug")
	wordDebug = flags.String("word", "", "force word to use")
	quiet = flags.Bool("quiet", false, "quiet")

	// engine to use
	engine = flags.String("engine", "aspell", "engine to use defaults to aspell, this is experimental may not provide good results")

	// process dictionary with special engine
	process = flags.String("process", "", "process a dicitonary to save time later")
	processOut = flags.String("processout", "", "where to save the processed dictionary")

	// use already processed dictionary with special engine
	processed = flags.String("processed", "", "processed dictionary to use")

	// dictionary to use with special engine
	specialDict = flags.String("specialdict", "", "special dict to use with special engine")

	// when -h or -help is supplied
	flag.ErrHelp = errors.New("help requested")
	flag.Usage = func() {
		log.Println(usage)
		flags.PrintDefaults()
	}

	flags.Parse(os.Args[1:])

	if len(*process) > 0 {
		if len(*processOut) > 0 {
			log.Println("Please specify out file")
			os.Exit(-1)
		}
		wordlist, err := os.Open(*process)
		if err != nil {
			log.Println(err)
		}
		defer wordlist.Close()

		saved, err := os.Create(*processOut)
		if err != nil {
			log.Println(err)
		}

		m := spell.NewModel()
		m.LoadWordList(wordlist)
		m.SaveWordList(saved)
		saved.Close()
		return
	} else if len(flags.Args()) >= 1 {
		if info, err := os.Stat(flags.Args()[0]); info.IsDir() {
			log.Println("Cannot use directory")
			os.Exit(-1)
		} else if err != nil {
			log.Println("could not open file for reading")
			log.Println(err.Error())
			os.Exit(-1)
		}
	} else {
		fmt.Println("no password file specified")
		os.Exit(-1)
	}

	passwords, err := os.Open(flags.Args()[0])
	if err != nil {
		log.Println(err)
		return
	}
	defer passwords.Close()

	scanner := bufio.NewScanner(passwords)

	// p is the channel to send the passwords down to get processed
	p := make(chan string, *threads)
	// words is the channel to send completed passwords down
	words := make(chan []Word, *threads)

	var wg sync.WaitGroup

	if *engine == "special" {
		m := spell.NewModel()
		if len(*processed) > 0 {
			dict, err := os.Open(*processed)
			if err != nil {
				log.Println(err)
				os.Exit(-1)
			}
			m = spell.LoadSavedWordList(dict)
		} else if len(*specialDict) > 0 {
			wordlist, err := os.Open(*specialDict)
			if err != nil {
				log.Println(err)
				os.Exit(-1)
			}

			m = spell.NewModel()
			m.LoadWordList(wordlist)
			wordlist.Close()
		} else {
			log.Println("no dictionary provided")
			os.Exit(-1)
		}

		for i := 0; i < *threads; i++ {
			go func() {
				for pass := range p {
					analyzePassword(pass, m, words)
				}
				wg.Done()
			}()
			wg.Add(1)
		}
	} else if *engine == "enchant" || *engine != "enchant" {
		for i := 0; i < *threads; i++ {
			go func() {

				m, err := enchant.NewEnchant()
				if err != nil {
					log.Println("enchant err", err)
				}
				defer m.Delete()

				m.BrokerOrdering("*", "aspell,mysell")
				m.LoadDict("en")

				for pass := range p {
					analyzePassword(pass, m, words)
				}
				wg.Done()
			}()
			wg.Add(1)
		}
	}
	go printRules(*basename+".word", *basename+".rule", words)

	quit := make(chan struct{})
	var counter uint
	//ticker := time.Tick(time.Second * 5)
	ticker := time.Tick(time.Second)
	start := time.Now()

	go func() {
		for {
			select {
			case <-ticker:
				elapsed := uint(time.Since(start).Seconds())
				if elapsed > 0 {
					fmt.Printf("\033[2K passwords processed %d; duration: %v; %d pass/s\r", counter, time.Since(start), counter/elapsed)
				}
			case <-quit:
				break
			}
		}
	}()

	for scanner.Scan() {
		counter++
		temp := scanner.Text()
		if checkReversiblePassword([]rune(temp)) {
			p <- temp
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("problem with scanner")
		log.Println(err.Error())
	}

	close(quit)
	close(p)
	wg.Wait()
	close(words)

	// this makes the terminal line go back to normal
	fmt.Println()

}

// TODO
// make this faster right now it is slow due to using fmt.Printf and
// concatenating strings in the String() method

func printRules(wordFileName, ruleFileName string, words chan []Word) {
	var wordFile *os.File
	var ruleFile *os.File

	// may not even need to stat it.
	// os.create will just overwrite the file

	// we remove the old files so we can write the new contents
	if _, err := os.Stat(wordFileName); err != nil {
		if os.IsExist(err) {
			err = os.Remove(wordFileName)
			if err != nil {
				log.Println(err.Error())
			}

		}
	}
	wordFile, err := os.Create(wordFileName)
	if err != nil {
		log.Println("cannot open file to write to:" + err.Error())
	}

	if _, err := os.Stat(ruleFileName); err != nil {
		if os.IsExist(err) {
			err = os.Remove(ruleFileName)
			if err != nil {
				log.Println(err.Error())
			}

		}
	}
	ruleFile, err = os.Create(ruleFileName)
	if err != nil {
		log.Println("cannot open file to write to:" + err.Error())
	}

	defer wordFile.Close()
	defer ruleFile.Close()

	wordbuf := bufio.NewWriter(wordFile)
	rulebuf := bufio.NewWriter(ruleFile)

	for word := range words {
		for _, a := range word {
			fmt.Fprintln(wordbuf, a.suggestion)
			fmt.Fprintf(rulebuf, "%v", a.hashcatRules)
			// try to flush right before the buffer gets filled
			if wordbuf.Buffered() >= 4000 {
				wordbuf.Flush()
			}
			if rulebuf.Buffered() >= 4000 {
				rulebuf.Flush()
			}
		}
	}
	// make sure that everything is flushed
	rulebuf.Flush()
	wordbuf.Flush()
}

// analyzePassword analyzing a single password
func analyzePassword(password string, m spell.Speller, out chan []Word) {

	var words []Word

	// if we are debugging then don't generateWords
	if len(*wordDebug) > 0 && len(m.Suggest(password)) > 0 {
		var temp Word
		temp.password = password
		temp.suggestion = *wordDebug
		temp.hashcatRules = make([][]string, 1)
		temp.bestRuleLength = 999

		temp.hashcatRules = generateHashcatRules(temp.suggestion, temp.password)
	} else {

		// generate words based on the password
		words = generateWords(password, m)

		for i, word := range words {
			// generate a list of hashcat rules for each suggestion
			words[i].hashcatRules = generateHashcatRules(word.suggestion, word.password)
		}
	}

	out <- words
}

type rule [][]string

func (r rule) String() string {
	var rule string
	for _, h := range r {
		for _, in := range h {
			rule += in + " "
		}
		rule += "\n"
	}
	return rule
}

func (r rule) Len() int           { return len(r) }
func (r rule) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r rule) Less(i, j int) bool { return len(r[i]) < len(r[j]) }

func generateHashcatRules(suggestion, password string) [][]string {
	levRules := GenerateLevenshteinRules([]rune(suggestion), []rune(password))

	var hashcatRules rule
	var hashcatRulesCollection rule

	var hashcatRule []string
	// generate a hashcat rule for each word
	for _, levRule := range levRules {

		if *simpleRules {
			hashcatRule = SimpleHashcatRules([]rune(suggestion), []rune(password), levRule)
		} else {
			hashcatRule = AdvancedHashcatRules(password, suggestion, levRule)
		}

		if hashcatRule == nil {
			log.Printf("processing failed")
		} else {
			hashcatRules = append(hashcatRules, hashcatRule)
		}
	}

	bestFoundRuleLength := 9999

	// perform some optimization
	sort.Sort(rule(hashcatRules))
	for _, hashcatRule := range hashcatRules {

		ruleLength := len(hashcatRule)

		if !*moreRules {
			if ruleLength < bestFoundRuleLength {
				bestFoundRuleLength = ruleLength

			} else if ruleLength > bestFoundRuleLength {
				if *debug {
					log.Printf("best rule length exceeded")
				}
				break
			}

			if ruleLength <= *maxRuleLen {
				hashcatRulesCollection = append(hashcatRulesCollection, hashcatRule)
			}
		}
	}

	return hashcatRulesCollection
}

// Word holds information for generating a hashcat rule
type Word struct {
	distance       int
	suggestion     string
	password       string
	preRule        string
	hashcatRules   rule
	bestRuleLength int
}

// Words is an alias to []Word
type Words []Word

func (w Words) Len() int           { return len(w) }
func (w Words) Swap(i, j int)      { w[i], w[j] = w[j], w[i] }
func (w Words) Less(i, j int) bool { return w[i].distance < w[j].distance }

func generateWords(password string, m spell.Speller) []Word {

	var words []Word
	var wordsCollection []Word

	// collect best edit distance
	bestFoundDistance := 9999

	preanalysisRules := []string{":", "r"}
	//preanalysisRules := []string{":", "r", "}", "{"}

	if !*bruteRules {
		preanalysisRules = preanalysisRules[:1]
	}

	var prePassword string
	for _, preRule := range preanalysisRules {
		prePassword = rules.ApplyRules([]string{preRule}, password)

		var suggestions []string
		if len(*wordDebug) > 0 {
			suggestions = []string{*wordDebug}
		} else if *simpleWords {
			suggestions = generateSimpleWords(prePassword, m)
		} else {
			suggestions = generateAdvancedWords(prePassword, m)
		}

		hashset1 := util.NewHash()

		for _, val := range suggestions {
			hashset1.Add(val)
		}

		// rules for each of the suggestions
		for _, suggestion := range suggestions {
			suggestion = strings.Replace(suggestion, " ", "", -1)
			suggestion = strings.Replace(suggestion, "-", "", -1)

			if !hashset1.Exists(suggestion) {
				suggestions = append(suggestions, suggestion)
				hashset1.Add(suggestion)
			}
		}

		/*
			// TODO what is the point of this??
			// debugging??
			if len(suggestions) != hashset1.Len() {
				// make these sorted
				//fmt.Println(suggestions)
				//fmt.Println(hashset1)
			}
		*/

		for _, suggestion := range suggestions {
			distance := Levenshtein(suggestion, prePassword)

			temp := Word{
				suggestion:     suggestion,
				distance:       distance,
				password:       prePassword,
				preRule:        preRule,
				bestRuleLength: 9999,
			}

			words = append(words, temp)
		}
	}

	sort.Sort(Words(words))

	for _, word := range words {
		if !*moreWords {
			if word.distance < bestFoundDistance {
				bestFoundDistance = word.distance
			}
		} else if word.distance > bestFoundDistance {
			if *debug {
				log.Println("best found distance suboptimal")
			}
			break
		}

		// filter words with large edit distance
		if word.distance <= *maxWordDist {
			wordsCollection = append(wordsCollection, word)
		} else {
			if *debug {
				log.Println("max distance exceeded")
			}
		}
	}

	if *maxWords > 0 {
		if *maxWords > len(wordsCollection) {
			wordsCollection = wordsCollection[:]
		} else {
			wordsCollection = wordsCollection[:*maxWords]
		}
	}

	return wordsCollection
}

func generateSimpleWords(password string, m spell.Speller) []string {
	return m.Suggest(password)
}

// leet speek translation map
var leet = map[rune]rune{
	'1': 'i',
	'2': 'z',
	'3': 'e',
	'4': 'a',
	'5': 's',
	'6': 'b',
	'7': 't',
	'8': 'b',
	'9': 'g',
	'0': 'o',
	'!': 'i',
	'|': 'i',
	'@': 'a',
	'$': 's',
	'+': 't',
}

// this is really expensive
// so we make them stay so not to run them for every word
var insertRegex = regexp.MustCompile(`(?i)^[^a-z]*(?P<password>.+?)[^a-z]*$`)
var emailRegex = regexp.MustCompile(`(?i)^(?P<password>.+?)@[A-Z0-9.-]+\.[A-Z]{2,4}`)

func generateAdvancedWords(password string, m spell.Speller) []string {
	// remove non alpha prefix and/or suffix
	// (?i) is ignore case
	insertionMatches := insertRegex.FindStringSubmatch(password)
	if insertionMatches != nil {
		// only the last one
		password = insertionMatches[len(insertionMatches)-1]

	}

	// match emails
	emailMatches := emailRegex.FindStringSubmatch(password)
	if emailMatches != nil {
		// only do the last one
		password = emailMatches[len(emailMatches)-1]
	}

	// common character matches to leet speak
	var preanalysisPassword string
	for _, c := range password {
		if val, ok := leet[c]; ok {
			preanalysisPassword += string(val)
		} else {
			preanalysisPassword += string(c)
		}
	}

	password = preanalysisPassword

	return generateSimpleWords(password, m)
}

// RuleWorks tests if a rule results in the correct managled word
func RuleWorks(word []rune, password []rune, operations []EditOp) bool {
	temp := make([]rune, len(word))
	copy(temp, word)

	for _, op := range operations {
		if op.Op == "insert" {
			rules.InsertAtN(temp, op.P, password[op.P])
		} else if op.Op == "delete" {
			rules.DeleteN(temp, op.P)
		} else if op.Op == "replace" {
			rules.OverwriteAtN(temp, op.P, password[op.P])
		}
	}

	if string(temp) == string(password) {
		return true
	}

	return false
}

// SimpleHashcatRules applies the basic hashcat rules based on delete, insert, replace
func SimpleHashcatRules(word []rune, password []rune, operations []EditOp) []string {
	if string(word) == string(password) {
		return []string{":"}
	}

	temp := make([]rune, len(word))
	copy(temp, word)
	r := []string{}

	for _, op := range operations {
		if op.Op == "insert" {
			r = append(r, fmt.Sprintf("i%d%c", op.P, password[op.P]))
			temp = rules.InsertAtN(temp, op.P, password[op.P])
		} else if op.Op == "delete" {
			r = append(r, fmt.Sprintf("D%d", op.P))
			temp = rules.DeleteN(temp, op.P)
		} else if op.Op == "replace" {
			r = append(r, fmt.Sprintf("o%d%c", op.P, password[op.P]))
			temp = rules.OverwriteAtN(temp, op.P, password[op.P])
		}
	}

	if string(temp) == string(password) {
		return r
	}

	return nil
}

// **** TODO need to fix for new rule and change of rule
// OMN and xMN
// not all rules are here add more??

// AdvancedHashcatRules applies all hashcat rules to a word
func AdvancedHashcatRules(passwordString, wordString string, perations []EditOp) []string {

	// TODO
	// can we do this earlier not in this function to save a fucntion call
	if passwordString == wordString {
		return []string{":"}
	}

	password := []rune(passwordString)
	word := []rune(wordString)

	needNewName := []string{}
	// this holds the current mangled as rules are applied
	wordRules := make([]rune, 0, len(word))
	wordRules = append(wordRules, []rune(word)[:]...)

	var passwordLower int
	var passwordUpper int
	for _, r := range password {
		if unicode.IsLower(r) {
			passwordLower++
		} else if unicode.IsUpper(r) {
			passwordUpper++
		}
	}

	for i, op := range perations {

		if op.Op == "insert" {
			needNewName = append(needNewName, fmt.Sprintf("i%c%c", rules.ToAlpha(op.P), password[op.P]))
			wordRules = rules.InsertAtN(wordRules, op.P, password[op.P])
		} else if op.Op == "delete" {
			needNewName = append(needNewName, fmt.Sprintf("D%c", rules.ToAlpha(op.P)))
			wordRules = rules.DeleteN(wordRules, op.P)
		} else if op.Op == "replace" {

			// rule was made obsolete by prior global replacement
			// test to see if word is greater than password to avoid index error
			if len(wordRules) >= len(password) && wordRules[op.P] == password[op.P] {
				if *debug {
					fmt.Println("obsolete rule")
				}

				// Swapping rules
			} else if op.P < len(password)-1 && op.P < len(word)-1 &&
				word[op.P] == password[op.P+1] &&
				word[op.P+1] == password[op.P] {

				if op.P == 0 && RuleWorks(word, password, perations[i+1:]) {
					needNewName = append(needNewName, "k")
					wordRules = rules.SwapFront(wordRules)
				} else if op.P == len(wordRules)-2 && RuleWorks(rules.SwapBack(wordRules), password, perations[i+1:]) {
					needNewName = append(needNewName, "K")
					wordRules = rules.SwapBack(wordRules)
				} else if RuleWorks(rules.SwapAtN(wordRules, op.P, op.P+1), password, perations[i+1:]) {
					// Swap any two characters (only adjacent swapping is supported)
					needNewName = append(needNewName, fmt.Sprintf("*%c%c", op.P, rules.ToAlpha(op.P+1)))
					wordRules = rules.SwapAtN(wordRules, op.P, op.P+1)
				} else {
					needNewName = append(needNewName, fmt.Sprintf("o%c%c", rules.ToAlpha(op.P), password[op.P]))
					wordRules = rules.OverwriteAtN(wordRules, op.P, password[op.P])
				}

				// Case Toggle: Uppercased a letter
			} else if unicode.IsLower(wordRules[op.P]) && unicode.ToUpper(wordRules[op.P]) == password[op.P] {
				// Toggle the case of all characters in word (mixed cases)
				if passwordUpper > 0 && passwordLower > 0 && RuleWorks(rules.ToggleCase(wordRules), password, perations[i+1:]) {
					needNewName = append(needNewName, "t")
					wordRules = rules.ToggleCase(wordRules)
					// Capitalize all letters
				} else if RuleWorks(rules.Uppercase(wordRules), password, perations[i+1:]) {
					needNewName = append(needNewName, "u")
					wordRules = rules.Uppercase(wordRules)
					// Capitalize the first letter
				} else if op.P == 0 && RuleWorks(rules.Capitalize(wordRules), password, perations[i+1:]) {
					needNewName = append(needNewName, "c")
					wordRules = rules.Capitalize(wordRules)
					// Toggle the case of characters at position N
				} else {
					needNewName = append(needNewName, fmt.Sprintf("T%c", rules.ToAlpha(op.P)))
					wordRules = rules.ToggleAt(wordRules, op.P)
				}

				// Case Toggle Lowercased a letter
			} else if unicode.IsUpper(wordRules[op.P]) && unicode.ToLower(wordRules[op.P]) == password[op.P] {
				// Toggle the case of all characters in word (mixed cases)
				if passwordUpper > 0 && passwordLower > 0 && RuleWorks(rules.ToggleCase(wordRules), password, perations[i+1:]) {
					needNewName = append(needNewName, "t")
					wordRules = rules.ToggleCase(wordRules)
					// Lowercase all letters
				} else if RuleWorks(rules.Lowercase(wordRules), password, perations[i+1:]) {
					needNewName = append(needNewName, "l")
					wordRules = rules.Lowercase(wordRules)
					// Lowercase the first found character, uppercase the rest
				} else if op.P == 0 && RuleWorks(rules.InvertCapitalize(wordRules), password, perations[i+1:]) {
					needNewName = append(needNewName, "C")
					wordRules = rules.InvertCapitalize(wordRules)
					// Toggle the case of characters at position N
				} else {
					needNewName = append(needNewName, fmt.Sprintf("T%c", rules.ToAlpha(op.P)))
					wordRules = rules.ToggleAt(wordRules, op.P)
				}

				// Special case substitution of 'all' instances (1337 $p34k)
			} else if unicode.IsLetter(wordRules[op.P]) && !unicode.IsLetter(password[op.P]) &&
				RuleWorks(rules.Replace(wordRules[0:], wordRules[op.P], password[op.P]), password, perations[i+1:]) {

				needNewName = append(needNewName, fmt.Sprintf("s%c%c", wordRules[op.P], password[op.P]))
				wordRules = rules.Replace(wordRules, wordRules[op.P], password[op.P])

				// Replace next character with current
			} else if op.P < len(password)-1 && op.P < len(wordRules)-1 &&
				password[op.P] == password[op.P+1] && password[op.P] == wordRules[op.P+1] {
				needNewName = append(needNewName, fmt.Sprintf(".%c", rules.ToAlpha(op.P)))
				wordRules = rules.ReplaceNPlus(wordRules, op.P)

				// Replace previous character with current
			} else if op.P > 0 && op.Word > 0 && password[op.P] == password[op.P-1] && password[op.P] == wordRules[op.P-1] {
				needNewName = append(needNewName, fmt.Sprintf(",%c", rules.ToAlpha(op.P)))
				wordRules = rules.ReplaceNMinus(wordRules, op.P)

				// ASCII increment
			} else if wordRules[op.P]+1 == password[op.P] {
				needNewName = append(needNewName, fmt.Sprintf("+%c", rules.ToAlpha(op.P)))
				wordRules = rules.ASCIIIncrementPlus(wordRules, op.P)

				// ASCII decrement
			} else if wordRules[op.P]-1 == password[op.P] {
				needNewName = append(needNewName, fmt.Sprintf("-%c", rules.ToAlpha(op.P)))
				wordRules = rules.ASCIIIncrementMinus(wordRules, op.P)

				// SHIFT left
			} else if wordRules[op.P]<<1 == password[op.P] {
				needNewName = append(needNewName, fmt.Sprintf("L%c", rules.ToAlpha(op.P)))
				wordRules = rules.BitwiseShiftLeft(wordRules, op.P)

				// SHIFT right
			} else if wordRules[op.P]>>1 == password[op.P] {
				needNewName = append(needNewName, fmt.Sprintf("R%c", rules.ToAlpha(op.P)))
				wordRules = rules.BitwiseShiftRight(wordRules, op.P)

				// Position based replacements.
			} else {
				needNewName = append(needNewName, fmt.Sprintf("o%c%c", rules.ToAlpha(op.P), password[op.P]))
				wordRules = rules.OverwriteAtN(wordRules, op.P, password[op.P])
			}

		}
	}
	// out of for loop

	// these next things convert rules to append $ and prepend rules

	// TODO
	// possibility to have either what the rule is now or
	// the rule swapped with these replacements

	// Prefix rules
	lastPrefix := 0
	var prefixRules []string
	for i, hashcatRule := range needNewName {
		if hashcatRule[0] == 'i' && rules.ToNumByte(hashcatRule[1]) == lastPrefix {
			prefixRules = append(prefixRules, fmt.Sprintf("^%c", hashcatRule[2]))
			lastPrefix++
			needNewName[i] = fmt.Sprintf("^%c", hashcatRule[2])
		} else {
			// TODO
			// dont know about breaking early here
			break
		}
	}

	// Appendix rules
	lastAppendix := len(password) - 1
	var appendixRules []string
	for i, hashcatRule := range needNewName {
		if hashcatRule[0] == 'i' && rules.ToNumByte(hashcatRule[1]) == lastAppendix {
			appendixRules = append(appendixRules, fmt.Sprintf("$%c", hashcatRule[2]))
			lastAppendix--
			needNewName[i] = fmt.Sprintf("$%c", hashcatRule[2])
		} else {
			break
		}
	}

	// Truncate left rules
	lastPrecut := 0
	for i, hashcatRule := range needNewName {
		if hashcatRule[0] == 'D' && rules.ToNumByte(hashcatRule[1]) == lastPrecut {
			needNewName[i] = "["
		} else {
			break
		}
	}

	// Truncate right rules
	lastPostcut := len(password)
	for i, hashcatRule := range needNewName {
		if hashcatRule[0] == 'D' && rules.ToNumByte(hashcatRule[1]) >= lastPostcut {
			needNewName[i] = "]"
		} else {
			break
		}
	}

	/*
		// naive implementation of OMN
		// will only work if the first rule is a delete
		overwrite := 0
		for i, hashcatRule := range needNewName {
			if hashcatRule[0] == 'D' && i < len(password)-1 && needNewName[i+1] == 'D' {
				overwrite++
				needNewName[i] = ""
			} else {
				break
			}
		}
		if overwrite > 0 {
			var temp []string
			temp = append(temp, fmt.Sprintf("O%c%c", rules.ToAlpha(0), rules.ToAlpha(overwrite)))
			temp = append(temp, needNewName[:]...)
			needNewName = temp
		}
	*/

	// Check if rules result in the correct password
	if string(wordRules) == passwordString {
		return needNewName
	}

	log.Printf("advanced processing failed: P: %s, M: %s, O: %s, %v\n", passwordString, string(wordRules), wordString, needNewName)
	return nil
}

func checkReversiblePassword(password []rune) bool {
	// check if a password is likely to be reversed
	// skip numeric passwords
	d := 0
	for _, r := range password {
		if unicode.IsDigit(r) {
			d++
		}
	}
	if d == len(password) {
		return false
	}

	// skip with less than 25% alpha
	// TODO based on entropy?
	d = 0
	for _, r := range password {
		if unicode.IsLetter(r) {
			d++
		}
	}
	if d < len(password)/4 {
		return false
	}

	return true
}
