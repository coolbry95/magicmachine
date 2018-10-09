# MagicMachine
MagicMachine generates rules based on a list of words. It reverses the words to source words then generates rules based on the reversal.

# Installation
Go 1.6 is required due to some CGo fixes. The enchant development library is also needed for a spell checker.  
For Ubuntu
```sudo apt-get install libenchant-dev```  
```go install github.com/coolbry95/magicmachine```  

# Usage/Help
Must supply flags then wordlist or wordlist then flags. This is due to a limitation in the flag library.
```Usage of magicmachine:
  -basename string
        basename for out files (default "analysis")
  -bruterules
        do not apply preanalysis rules such as reversing the password
  -debug
        output debugging information
  -engine string
        engine to use defaults to aspell, this is experimental may not provide good results (default "aspell")
  -maxrulelen int
        max rule length (default 15)
  -maxrules int
        max rules (default 5)
  -maxwordist int
        max word distance (default 10)
  -maxwords int
        max words (default 5)
  -morerules
        more rules
  -morewords
        more words
  -process string
        process a dicitonary to save time later
  -processed string
        processed dictionary to use
  -processout string
        where to save the processed dictionary
  -quiet
        quiet
  -simplerules
        simple rules
  -simplewords
        simple words
  -specialdict string
        special dict to use with special engine
  -threads int
        number of threads to use default max CPUS (default 8)
  -verbose
        verbose
  -word string
        force word to use```



# License
MagicMachine is licensed under the MIT license.

#### Thanks
Steve Hatchett for the optimized levenshtien algorithm  
iphelix for the orignal implementation  
faroo for the spelling correction algorithm  
