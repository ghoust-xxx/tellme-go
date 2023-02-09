# About

`tellme-go` allow you to download and listen pronunciation audio records.
For now only forvo.com supported. Support for different sources possible
in future.

Program supports cache and CLI/TUI interface for comfortable user interaction.
Batch processing is also possible.

You can enter words from stdin, as CLI arguments or from a text file.

# Installation

You can build program by cloning repository and running in source directory:
```
cd tellme-go/
go build
./tellme-go
```

After building you can put executable in any directory in your `$PATH` if you
wish.

You also will need installed [mpg123](https://www.mpg123.de). Make sure it
could be found in your `$PATH`.

You also can found compiled version for `x86_64-linux` in `Releases` tab.


# Usage

```
Usage: tellme-go [options] [words for pronunciation]

  -f [filename]
        file with words for pronunciation
  -c [yes | no]
        cache files [yes | no]. Default yes
  -cache-dir [any valid path]
        cache directory [any valid path]. Default /home/ghoust/.cache/tellme
  -check [yes | no]
        check existence of pronunciation [yes | no]. Default yes
  -d [yes | no]
        download audio files in current directory [yes | no]. Default yes
  -f filename
        read input from filename
  -i [yes | no]
        interactive mode [yes | no]. Default no
  -l [en | es | de | etc]
        language [en | es | de | etc]. Default en
  -t [mp3 | ogg ]
        audio files type [mp3 | ogg ]. Default mp3
  -verbose [yes | no]
        verbose mode [yes | no]. Default no
  -version
        print program version
```

# Example

You want to listen how to pronounce word `cat` in english. Just type:
```
tellme-go -i yes -l en cat
```
and you will be presented with CLI interface where you can choose one of
pronunciations (use `j` and `k` keys or enter a number), repeat the same
audio again (`r` key) or enter a new word (`e` key).

If you have entered more then one word you can go back and forward between
them using `n` (next) and `p` (previous) keys.

Without `-i yes` program will just downloads and saves file `cat.mp3` in
your current directory.

![Program in action](/doc/in_action.gif)


- Copyright (c) 2022 Alex Ghoust.
