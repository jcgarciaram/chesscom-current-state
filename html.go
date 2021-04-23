package main

import (
	"bytes"
	"fmt"
	"html/template"
	"path"
	"time"
)

var (
	chessTemplate = "chessTemplate.html"
	faviconFile   = "favicon.ico"
	// chessTemplate = os.Getenv("HOME") + "/chessTemplate.html"
	// faviconFile   = os.Getenv("HOME") + "/favicon.ico"
)

// The following structs are a bit funky, but they were built this way
// so that when passed to HTML template file, games can be displayed in groups of three.

// htmlData has all the data needed to build out the html template.
type htmlData struct {
	CurrGameGroups     []gameGroup
	FinishedGameGroups []gameGroup
}

// getIndexHTML takes a slice of games and returns an HTML
// webpage using chessTemplate.html as a template file.
func getIndexHTML(currentGameGroups []gameGroup, finishedGameGroups []gameGroup) ([]byte, error) {

	// Initialize the gameSlices object which will be passed
	// into the html template file
	data := htmlData{
		CurrGameGroups:     currentGameGroups,
		FinishedGameGroups: finishedGameGroups,
	}

	funcs := template.FuncMap{
		"add":         add,
		"subtract":    subtract,
		"getIndexes":  getIndexes,
		"monthString": monthString,
	}

	// Parse the HTML template file
	name := path.Base(chessTemplate)
	tmplt, err := template.New(name).Funcs(funcs).ParseFiles(chessTemplate)
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
