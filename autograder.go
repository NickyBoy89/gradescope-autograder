package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/NickyBoy89/gradescope-autograder/diffmatchpatch"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
)

// Autograder

// Reads and runs python files starting with "test-", which are tests

// Also takes in files beginning with ".stdin" which test the execution of the entire program

func main() {
	mainFile := flag.String("main", "main.py", "The python file that contains the main function, defaults to \"main.py\"")
	targetDirectory := flag.String("targetDir", ".", "The directory to search for test files in, defaults to \".\" (the current directory)")
	stopFail := flag.Bool("stopFail", false, "Stops the test after the first test failure. Useful with large test diffs")
	verbose := flag.Bool("v", false, "Always prints the stdout of commands, instead of only on error")
	raw := flag.Bool("raw", false, "Displays the raw output of both commands, instead of a diff")

	flag.Parse()

	if *mainFile == "main.py" {
		log.Warn("No main python file specified, defaulting to \"main.py\". Specify a main file with --main <filename>")
	}

	if *targetDirectory == "." {
		log.Warn("No target directory specified, the program will look for tests in the current directory")
		log.Warn("If this is not what you intended, set a target directory with the --targetDir flag")
	}

	err := os.Chdir(*targetDirectory)
	if err != nil {
		log.Fatal("Error changing directory: %v", err)
	}

	// Get all the matching python files to run as tests
	testFiles, err := filepath.Glob("test-*.py")
	if err != nil {
		log.Fatalf("Error searching for test files: %v", err)
	}

	// Run all the tests that test parts of the program
	color.Blue("Running tests now")
	for _, test := range testFiles {
		fmt.Printf("Running test [%v] ... ", color.GreenString(test))
		output, err := TestFile(test, "")
		if err != nil {
			color.Red("FAILED")
			color.Red(err.Error())
		}

		testOutput, err := os.ReadFile(ChangeFileExtensionTo(test, ".out"))
		if err != nil {
			log.Fatalf("Error opening output file %v: %v", ChangeFileExtensionTo(test, ".out"), err)
		}

		if string(output) != string(testOutput) {
			color.Red("FAILED")

			diff := diffmatchpatch.New()
			outputDifference := diff.DiffMain(string(output), string(testOutput), true)
			fmt.Println(diff.DiffPrettyText(outputDifference))
			continue
		}
		color.Green("PASSED")
	}

	// Get all the .stdin files
	inFiles, err := filepath.Glob("*.stdin")
	if err != nil {
		log.Fatalf("Error searching for .stdin files: %v", err)
	}

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
			if *verbose {
				fmt.Println(commandOutput)
			}
			color.Red(err.Error())
			if *stopFail {
				break
			}
			continue
		}

		outputData, err := os.ReadFile(testOutput)
		if err != nil {
			log.Fatalf("Error reading output file %v: %v", testOutput, err)
		}

		if string(commandOutput) != string(outputData) {
			color.Red("FAILED")
			if !*raw {
				// Find the diff between the outputs of the two strings
				diff := diffmatchpatch.New()
				outputDifference := diff.DiffMain(string(outputData), string(commandOutput), false)
				//err := os.WriteFile("index.html", []byte("<html><body style=\"white-space:pre;\">"+diff.DiffPrettyHtml(outputDifference)+"</body></html>"), 0755)
				//if err != nil {
				//	panic(err)
				//}
				fmt.Println(diff.DiffPrettyText(outputDifference))
			} else {
				color.Blue("EXPECTED")
				fmt.Println(string(outputData))
				color.Blue("ACTUAL")
				fmt.Println(string(commandOutput))
			}
			if *stopFail {
				break
			}
		}
		color.Green("PASSED")
	}
}

func ChangeFileExtensionTo(path, ext string) string {
	return path[:len(path)-len(filepath.Ext(path))] + ext
}

func TestFile(testFile, inputFile string) ([]byte, error) {
	testCommand := exec.Command("python3", testFile)

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

	if len(stderr) != 0 {
		return nil, errors.New(string(stderr))
	}

	return stdout, nil
}
