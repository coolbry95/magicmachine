# MagicMachine
MagicMachine generates rules based on a list of words. It reverses the words to source words then generates rules based on the reversal.

# License
MagicMachine is licensed under the MIT license.

# Installation
Go 1.6 is required due to some CGo fixes. The enchant development library is also needed for a spell checker.  
```go get github.com/coolbry95/edit```

# Usage/Help
must supply flags then wordlist.

#### Thanks
Steve Hatchett for the optimized levenshtien algorithm  
iphelix for the orignal implementation  
faroo for the spelling correction algorithm  
