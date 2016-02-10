# MagicMachine
MagicMachine generates rules based on a list of words. It reverses the words to source words then generates rules based on the reversal.

# License
MagicMachine is licensed under the MIT license.

# Installation
Go 1.6 is required due to some CGo fixes. The enchant development library is also needed for a spell checker.  
For Ubuntu
```sudo apt-get install libenchant-dev
go install github.com/coolbry95/magicmachine```

# Usage/Help
Must supply flags then wordlist or wordlist then flags. This is due to a limitation in the flag library.

#### Thanks
Steve Hatchett for the optimized levenshtien algorithm  
iphelix for the orignal implementation  
faroo for the spelling correction algorithm  
