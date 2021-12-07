package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/NickyBoy89/gradescope-autograder/diffmatchpatch"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
)

// The timeout after which a test will be considered "canceled"
// defaults to one second
const commandTimeout = time.Second * 1

// Autograder

// Reads and runs python files starting with "test-", which are tests

// Also takes in files beginning with ".stdin" which test the execution of the entire program

func main() {
	mainFile := flag.String("main", "", "The python file to run the tests against, required for programs that test that file specifically")
	targetDirectory := flag.String("targetDir", ".", "The directory to search for test files in, defaults to \".\" (the current directory)")
	stopFail := flag.Bool("stopFail", false, "Stops the test after the first test failure. Useful with large test diffs")
	raw := flag.Bool("raw", false, "Displays the raw output of both commands, instead of a diff")

	flag.Parse()

	if *targetDirectory == "." {
		log.Warn("No target directory specified, the program will look for tests in the current directory")
		log.Warn("If this is not what you intended, set a target directory with the --targetDir flag")
	}

	// Collect all the python files in the original directory
	originalFiles, err := filepath.Glob("*.py")
	if err != nil {
		absPath, err := filepath.Abs(".")
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Could not get absolute path of current directory")
		}
		log.WithFields(log.Fields{
			"directory": absPath,
			"error":     err,
		}).Fatal("Error searching for original files")
	}

	// Copy all the original python files to the target directory
	for _, orig := range originalFiles {
		originalFileData, err := os.ReadFile(orig)
		if err != nil {
			log.WithFields(log.Fields{
				"file":  fmt.Sprintf("%s/%s", *targetDirectory, orig),
				"error": err,
			}).Fatal("Error reading file")
		}
		err = os.WriteFile(fmt.Sprintf("%s/%s", *targetDirectory, orig), originalFileData, 0755)
		if err != nil {
			log.WithFields(log.Fields{
				"file":  fmt.Sprintf("%s/%s", *targetDirectory, orig),
				"error": err,
			}).Fatal("Error writing to file")
		}
	}

	// Gather all the test files
	testFiles, err := filepath.Glob(*targetDirectory + "/test-*.py")
	if err != nil {
		log.Fatalf("Error searching for test files: %v", err)
	}

	// Run the testcases that just compare their outputs
	// but do not run a program directly
	color.Blue("Running tests now")
	for _, test := range testFiles {
		fmt.Printf("Running test [%v] ... ", color.GreenString(test))
		output, err := TestFile(test, "")
		if err != nil {
			color.Red("FAILED")
			color.Red(err.Error())
			fmt.Println(string(output))
		}

		testOutput, err := os.ReadFile(ChangeFileExtensionTo(test, ".out"))
		if err != nil {
			log.Fatalf("Error opening output file %v: %v", ChangeFileExtensionTo(test, ".out"), err)
		}

		if !compareOutputs(string(testOutput), string(output), *raw) && *stopFail {
			break
		}
	}

	// Get all the .stdin files
	inFiles, err := filepath.Glob(*targetDirectory + "/*.stdin")
	if err != nil {
		log.Fatalf("Error searching for .stdin files: %v", err)
	}

	// Make sure that the user didn't leave the program name blank
	// and there were testcases to run against them
	if *mainFile == "" && len(inFiles) > 0 {
		log.Error("A main python file was not specified, but was required for some testcases")
		log.Error("Please specify one with the --main flag")
		return
	}

	// Run all the testcases that call a python program directly
	color.Blue("Comparing output of program to testcases")
	for _, testInput := range inFiles {
		testOutput := ChangeFileExtensionTo(testInput, ".out")
		_, err := os.Stat(testOutput)
		if err != nil {
			if os.IsNotExist(err) {
				color.Red("ERROR")
				color.Red("Missing complimentary .out for %v: should be named %v", testInput, testOutput)
				continue
			}
			log.Fatalf("Error getting info of %b: %v", testInput, err)
		}
		fmt.Printf("Running program with input from [%v] ... ", color.GreenString(testInput))

		commandOutput, err := TestFile(*mainFile, testInput)
		if err != nil {
			color.Red("FAILED")
			// Print the error out
			color.Red(err.Error())
			// Print the stdout anyways
			fmt.Println(string(commandOutput))
			if *stopFail {
				break
			}
			continue
		}

		outputData, err := os.ReadFile(testOutput)
		if err != nil {
			log.Fatalf("Error reading output file %v: %v", testOutput, err)
		}

		if !compareOutputs(string(outputData), string(commandOutput), *raw) && *stopFail {
			break
		}
	}
}

// Compares the outputs of two commands, and prints the diff of them
// if the raw arg is passed, it instead prints out the raw outputs of the two commands
// The function also returns the status of the test executed within it
func compareOutputs(expected, actual string, raw bool) bool {
	if actual != expected {
		color.Red("FAILED")
		if !raw {
			// Find the diff between the outputs of the two strings
			diff := diffmatchpatch.New()
			outputDifference := diff.DiffMain(expected, actual, false)
			fmt.Println(diff.DiffPrettyText(outputDifference))
		} else {
			color.Blue("EXPECTED")
			fmt.Println(expected)
			color.Blue("ACTUAL")
			fmt.Println(actual)
		}
		return false
	}
	color.Green("PASSED")
	return true
}

func ChangeFileExtensionTo(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}

// TestFile takes in a filename and its stdin file and runs the code
// returning the stdout and/or any errors that happen along the way
// if the command errors out, the error will return the stderr
func TestFile(testFile, inputFile string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	testCommand := exec.CommandContext(ctx, "python3", testFile)

	if inputFile != "" {
		in, err := os.Open(inputFile)
		if err != nil {
			return nil, err
		}
		testCommand.Stdin = in
	}

	testStdout, err := testCommand.StdoutPipe()
	if err != nil {
		return nil, err
	}

	testStderr, err := testCommand.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := testCommand.Start(); err != nil {
		return nil, err
	}

	stdout, _ := io.ReadAll(testStdout)
	stderr, _ := io.ReadAll(testStderr)

	// Return the stdout anyways, even if there is stderr
	if string(stderr) != "" {
		return stdout, errors.New(string(stderr))
	}
	return stdout, nil
}
