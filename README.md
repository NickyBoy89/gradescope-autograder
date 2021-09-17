# Gradescope-Autograder

Gradescope autograder is a program that runs the Gradescope testcases against your own code, and instead of diff messages such as "these two lines are different", where there is an uncertain number of invisible characters different between the two lines

## Requirements

1. A version of `golang` installed

## Usage

1. Build the binary with `go build`

2. Copy your code into the same directory that the tests are in
    * For Linux/MacOS, just append a "`cp *.py <directory that tests are in> && `" to the beginning of the next code

3. Run the binary with `./gradescope-autograder --main <your code file> --targetDir <directory that the tests are in>`

4. If you have any errors, make sure that you copied your code files into the directory that the test files are in

## Command-line args

* `--main` specifies the code to test against, defaults to `main.py`
* `--targetDir` specifies the directory that you put the tests in, defaults to `.`, or the current directory
