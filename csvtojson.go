package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	filePath   string
	separator  string
	pretty     bool
	outputPath string
}

func main() {
	// Shows useful information when user enters --help option
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <csvFile>\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
	// Getting file data entered by user
	opts, err := getOpts()

	// If there is an error, gracefully exit
	if err != nil {
		exitGracefully(err)
	}

	// Validating file input
	if _, err := checkIfValidFile(opts.filePath); err != nil {
		exitGracefully(err)
	}

	if err != nil {
		exitGracefully(err)
	}

	// Create a channel to handle writing JSON to file between goroutines
	writerChannel := make(chan map[string]string)
	// A channel to be written to to signify the file is done being written
	done := make(chan bool)

	// Parsing the CSV
	go processCsvFile(opts, writerChannel)
	// Writing JSON to new file
	go writeJSONFile(opts.outputPath, writerChannel, done, opts.pretty)

	// Wait for done channel to receive a value so that the function can finish
	<-done
}

// Responsible for getting the terminal input data, validating, and returning the
// struct (or error) that our program will use
func getOpts() (options, error) {

	// Ensuring we're getting the correct # of arguments
	if len(os.Args) < 2 {
		return options{}, errors.New("A filepath argument must be given!")
	}

	// These are out options flags.
	// Using the flag pkg from stdlib, we provide the flag's name, a default value, and
	// a short description that can be displayed with --help to the user
	separator := flag.String("separator", "comma", "Column Separator")
	pretty := flag.Bool("pretty", false, "Generate pretty JSON")
	outputPath := flag.String("outputPath", "", "Path to save JSON output file")

	// Parsing our command line arguments
	flag.Parse()

	// The only non-flag arg is the file location
	fileLocation := flag.Arg(0)

	// Validating the separator flags
	if !(*separator == "comma" || *separator == "semicolon") {
		return options{}, errors.New("Only comma or semicolon separators allowed")
	}

	// If a path to save the output json is provided, check that the path exists and is
	// a directory
	if *outputPath != "" {
		fileInfo, err := os.Stat(*outputPath)
		if os.IsNotExist(err) {
			return options{}, errors.New("Path provided to save output JSON does not exist")
		} else if !fileInfo.IsDir() {
			return options{}, errors.New("Path provided to save output JSON is not a directory")
		}
		// Otherwise, if not provided, default it to the input file's directory path
	} else {
		*outputPath = filepath.Dir(fileLocation)
	}

	// Complete the output path for saving the json data by joining the path to the output
	// directory and the csv filename without the csv prefix
	*outputPath = filepath.Join(*outputPath, fmt.Sprintf("%s.json", strings.TrimSuffix(filepath.Base(fileLocation), ".csv")))

	// If all validations have been passed, we return the struct that gives our program
	// all it needs to run
	return options{fileLocation, *separator, *pretty, *outputPath}, nil
}

// Responsible for ensuring the file is a csv file and/or exists
func checkIfValidFile(filename string) (bool, error) {
	// Checking if the filename has a .csv extension
	if fileExtension := filepath.Ext(filename); fileExtension != ".csv" {
		return false, fmt.Errorf("File %s is not a CSV", filename)
	}

	if _, err := os.Stat(filename); err != nil && os.IsNotExist(err) {
		return false, fmt.Errorf("File %s does not exist", filename)
	}

	return true, nil
}

func processCsvFile(opts options, writerChannel chan<- map[string]string) {
	// Open the file based on the filepath
	file, err := os.Open(opts.filePath)
	// Make sure there's no error, and if there is error gracefully
	check(err)
	// Close file when all is said and done
	defer file.Close()

	// Defining headers and line slices
	var headers, line []string

	// Init CSV reader
	reader := csv.NewReader(file)

	// Change the default separator if the semicolon option is set
	if opts.separator == "semicolon" {
		reader.Comma = ';'
	}

	// Read the first line to get the headers
	headers, err = reader.Read()

	// Check for error
	check(err)

	// While loop iterating until broken
	for {
		// Read the next line, returns a slice of string with each elem being a csv column
		line, err = reader.Read()

		// If we get an End Of File error, close the channel and break the loop
		if err == io.EOF {
			close(writerChannel)
			break
		} else if err != nil {
			// Gracefully handle unexpected errors
			exitGracefully(err)
		}

		// Process the CSV line
		record, err := processLine(headers, line)

		// If we get an error here, it means we got a wrong number of columns, so we skip this line
		if err != nil {
			fmt.Printf("Line : %sError: %s\n", line, err)
			continue
		}

		// Otherwise send the processed record thru the channel
		writerChannel <- record
	}
}

// Responsible for returning a map of header to column data per csv line
func processLine(headers []string, dataList []string) (map[string]string, error) {
	// Make sure there is the same num of headers as columns, otherwise throw error
	if len(dataList) != len(headers) {
		return nil, errors.New("line does not match headers format, skipping line.")
	}

	// Create the map we're going to populate
	recordMap := make(map[string]string)

	// For each header we are going to set a map key with the corresponding column val
	for i, name := range headers {
		recordMap[name] = dataList[i]
	}

	// Returning the generated map
	return recordMap, nil
}

// Responsible for writing the JSON file
func writeJSONFile(jsonOutputPath string, writeChannel <-chan map[string]string, done chan<- bool, pretty bool) {
	// Init a JSON writer func
	writeString := createStringWriter(jsonOutputPath)
	// Init the JSON parse func and the breakline char
	jsonFunc, breakLine := getJSONFunc(pretty)

	//Info log...
	fmt.Println("Writing JSON file...")

	// Write the first character of JSON file, starting with "[" since it will always generate
	// and array of records
	writeString("["+breakLine, false)

	first := true

	for {
		// Waiting for records pushed into writerChannel
		record, more := <-writeChannel

		// If the channel is "open" for more transmission
		if more {
			// If it is NOT the first record, break the line
			if !first {
				writeString(","+breakLine, false)
				// otherwise don't break the line
			} else {
				first = false
			}
			// Parse the record into JSON
			jsonData := jsonFunc(record)
			// Writing the JSON string with the writer function
			writeString(jsonData, false)
			// If here, then no more records to parse and need to close the file
		} else {
			// Writing the last char to the file and close it
			writeString(breakLine+"]", true)
			// Print that we are done to terminal
			fmt.Println("Done!")
			// Send "done" signal to main func to let it know it can start exiting
			done <- true
			// Break out of the loop
			break
		}
	}
}

// Responsible for returning a function that writes to a JSON file
// Uses encapsulation to init a new file and returns a function scoped to the context
// of the file initialized in the outer context
func createStringWriter(jsonOutputPath string) func(string, bool) {
	// Open the JSON file we will start writing to
	f, err := os.Create(jsonOutputPath)
	// Check for err, gracefully error
	check(err)

	// Return the function that will be used to write to the JSON file we decalred above
	return func(data string, close bool) {
		// Write to the JSON file
		_, err := f.WriteString(data)
		// Check for error, gracefully handle
		check(err)
		// If close == true, then there's no more data left to write to close the file
		if close {
			f.Close()
		}
	}
}

// Responsible for defining how the JSON will be written
// Returns a function that is used to write a JSON string based on how
// we configure the function to write the JSON
func getJSONFunc(pretty bool) (func(map[string]string) string, string) {
	// The function that marshals the records into json
	var jsonFunc func(map[string]string) string
	// The linebreak character to use
	var breakLine string

	// If pretty is enabled, we should format the JSON with line breaks and indentation
	if pretty {
		// The linebreak char will be a newline
		breakLine = "\n"
		jsonFunc = func(record map[string]string) string {
			jsonData, _ := json.MarshalIndent(record, "   ", "   ")
			return "   " + string(jsonData)
		}
	} else {
		breakLine = ""
		jsonFunc = func(record map[string]string) string {
			jsonData, _ := json.Marshal(record)
			return string(jsonData)
		}
	}
	return jsonFunc, breakLine
}

func check(e error) {
	if e != nil {
		exitGracefully(e)
	}
}
func exitGracefully(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
