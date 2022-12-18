package utils

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Testsuites contains list of testsuites
type Testsuites struct {
	XMLName    xml.Name    `xml:"testsuites"`
	Testsuites []Testsuite `xml:"testsuite"`
}

// Testsuite contain all the information about the tests
type Testsuite struct {
	XMLName xml.Name `xml:"testsuite"`
	//Properties []Property `xml:"properties"`
	Name      string     `xml:"name,attr"`
	Tests     string     `xml:"tests,attr"`
	Skipped   string     `xml:"skipped,attr"`
	Errors    string     `xml:"errors,attr"`
	Failures  string     `xml:"failures,attr"`
	Testcases []Testcase `xml:"testcase"`
}

// Testcase is an individual testcase in the end-to-end report
type Testcase struct {
	XMLName xml.Name `xml:"testcase"`
	Name    string   `xml:"name,attr"`
	Status  string   `xml:"status,attr"`
}

// E2eReportParser will parse the kubernetes end-to-end report.
func E2eReportParser(filename string) {
	xmlFile, err := os.Open(filepath.Clean(filename))
	if err != nil {
		fmt.Println(err)
	}
	defer func(xmlFile *os.File) {
		err := xmlFile.Close()
		if err != nil {
			fmt.Printf("Unable to close %s", filename)
		}
	}(xmlFile)
	byteValue, _ := ioutil.ReadAll(xmlFile)
	var testsuites Testsuites
	err = xml.Unmarshal(byteValue, &testsuites)
	if err != nil {
		return
	}
	var totalTestsCount int
	var skippedTestsCount int
	var failedTestsCount int
	var skippedTests []string
	var passedTests []string
	var failedTests []string

	for i := 0; i < len(testsuites.Testsuites); i++ {
		fmt.Println("TestSuite Name: " + testsuites.Testsuites[i].Name)
		totalTestsCount, _ = strconv.Atoi(testsuites.Testsuites[i].Tests)
		skippedTestsCount, _ = strconv.Atoi(testsuites.Testsuites[i].Skipped)
		failedTestsCount, _ = strconv.Atoi(testsuites.Testsuites[i].Failures)
		testsRun := totalTestsCount - skippedTestsCount
		TestsPassed := testsRun - failedTestsCount
		fmt.Printf("Total Tests Executed: %d\n", testsRun)
		fmt.Printf("Total Tests Passed: %d\n", TestsPassed)
		fmt.Printf("Total Tests Failed: %d\n", failedTestsCount)
		for j := 0; j < len(testsuites.Testsuites[i].Testcases); j++ {
			if strings.HasPrefix(testsuites.Testsuites[i].Testcases[j].Name, "[It] External Storage") {
				if testsuites.Testsuites[i].Testcases[j].Status == "skipped" {
					skippedTests = append(skippedTests, testsuites.Testsuites[i].Testcases[j].Name)
				} else if testsuites.Testsuites[i].Testcases[j].Status == "failed" {
					failedTests = append(failedTests, testsuites.Testsuites[i].Testcases[j].Name)
				} else {
					passedTests = append(passedTests, testsuites.Testsuites[i].Testcases[j].Name)
				}

			}

		}
		if len(failedTests) > 0 {
			fmt.Println("Failed Tests Are:")
		}
		for k := 0; k < len(failedTests); k++ {
			fmt.Println(failedTests[k] + "\n")

		}

	}

}
