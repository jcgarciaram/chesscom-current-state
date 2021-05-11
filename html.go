package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"time"
)

var (
	//go:embed website/index.html
	indexHTMLTemplate string

	//go:embed website/gamesForMonth.html
	gamesForMonthHTMLTemplate string

	//go:embed website/img/favicon.ico
	faviconFile []byte
)

// The following structs are a bit funky, but they were built this way
// so that when passed to HTML template file, games can be displayed in groups of three.

// htmlData has all the data needed to build out the html template.
type htmlData struct {
	CurrGameGroups     []gameGroup
	FinishedGameGroups []gameGroup
}

// getIndexHTMLBytes takes a slice of games and returns an HTML
// webpage using index.html as a template file.
func getIndexHTMLBytes(currentGameGroups []gameGroup) ([]byte, error) {

	// Initialize the gameSlices object which will be passed
	// into the html template file
	data := htmlData{
		CurrGameGroups: currentGameGroups,
	}

	funcs := template.FuncMap{
		"add":         add,
		"subtract":    subtract,
		"getIndexes":  getIndexes,
		"monthString": monthString,
	}

	// Parse the HTML template file
	tmplt, err := template.New("index").Funcs(funcs).Parse(indexHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("could not parse file template: %w", err)
	}

	// Pass in the data
	outputParsed := bytes.Buffer{}
	err = tmplt.Execute(&outputParsed, data)
	if err != nil {
		return nil, fmt.Errorf("could not execute file template: %w", err)
	}

	// Return the bytes of the webpage
	return outputParsed.Bytes(), nil
}

// getGamesForMonthHTMLBytes takes a slice of games and returns an HTML
// webpage using gamesForMonth.html as a template file.
func getGamesForMonthHTMLBytes(finishedGameGroups []gameGroup) ([]byte, error) {

	// Initialize the gameSlices object which will be passed
	// into the html template file
	data := htmlData{
		FinishedGameGroups: finishedGameGroups,
	}

	funcs := template.FuncMap{
		"add":         add,
		"subtract":    subtract,
		"getIndexes":  getIndexes,
		"monthString": monthString,
	}

	// Parse the HTML template file
	tmplt, err := template.New("gamesForMonth").Funcs(funcs).Parse(gamesForMonthHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("could not parse file template: %w", err)
	}

	// Pass in the data
	outputParsed := bytes.Buffer{}
	err = tmplt.Execute(&outputParsed, data)
	if err != nil {
		return nil, fmt.Errorf("could not execute file template: %w", err)
	}

	// Return the bytes of the webpage
	return outputParsed.Bytes(), nil
}

func add(x, y int) int {
	return x + y
}

func subtract(x, y int) int {
	return x - y
}

func getIndexes(interval int, max int) []int {
	indexes := []int{}
	for i := 0; i < max; i += interval {
		indexes = append(indexes, i)
	}
	return indexes
}

func monthString(month time.Month) string {
	return month.String()
}
