package main

import (
	"flag"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func Test_getFileData(t *testing.T) {
	// Defining test slice -- each unit test should have the follow properties:
	tests := []struct {
		name    string    // name of the test
		want    inputFile // what inputFile instance our function should return
		wantErr bool      // whether we want an error
		osArgs  []string  // The command args used for this test
	}{
		// Here we're declaring each unit test input and output data as defined above
		{"Default parameters", inputFile{"test.csv", "comma", false}, false, []string{"cmd", "test.csv"}},
		{"No parameters", inputFile{}, true, []string{"cmd"}},
		{"Semicolon enabled", inputFile{"test.csv", "semicolon", false}, false, []string{"cmd", "--separator=semicolon", "test.csv"}},
		{"Pretty enabled", inputFile{"test.csv", "comma", true}, false, []string{"cmd", "--pretty", "test.csv"}},
		{"Pretty and semicolon enabled", inputFile{"test.csv", "semicolon", true}, false, []string{"cmd", "--pretty", "--separator=semicolon", "test.csv"}},
		{"Separator not identified", inputFile{}, true, []string{"cmd", "--separator=pipe", "test.csv"}},
	}

	// Iterate over our test slice
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save reference to original os.Args
			actualOsArgs := os.Args
			// Function to run once test is done
			defer func() {
				// Resetting the original os.Args
				os.Args = actualOsArgs
				// Resetting the Flag command ling so we can parse flags again
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			}()

			// Setting the command line args for this specifc test
			os.Args = tt.osArgs

			// Running the function we wish to test
			got, err := getFileData()

			// An assertion of whether or not we want to get an error value
			if (err != nil) != tt.wantErr {
				t.Errorf("getFileData() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Asserting we are getting the corrent "want" value
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFileData() = %v, want %v", got, tt.want)
			}

		})
	}
}

func Test_checkIfValidFile(t *testing.T) {

	// Creating a temporay empty csv file
	tmpfile, err := ioutil.TempFile("", "test*.csv")

	// Just in case creating the temporary file returns an error, shouldn't ever happen.
	if err != nil {
		panic(err)
	}

	// Remove the file once the function is done
	defer os.Remove(tmpfile.Name())

	type args struct {
		filename string
	}
	tests := []struct {
		name     string
		filename string
		want     bool
		wantErr  bool
	}{
		{"File does exist", tmpfile.Name(), true, false},
		{"File does not exist", "nowhere/test.csv", false, true},
		{"File is not csv", "test.txt", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkIfValidFile(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkIfValidFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkIfValidFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
