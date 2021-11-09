# Gradescope-Autograder

Gradescope autograder is a program that runs python Gradescope testcases against your own code

It provides configurable diff messages, and shows invisible characters that would otherwise cause errors without a clear cause

## Requirements

1. A version of `golang` installed

## Usage

1. Build the binary with `go build`

2. Run the binary with `./gradescope-autograder --main <your code file> --targetDir <directory that the tests are in>`

## Command-line args

* `--main` specifies the code to test against, required for testcases that run a python program specifically
* `--targetDir` specifies the directory that you put the tests in, defaults to `.`, or the current directory
