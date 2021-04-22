package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image/color"
	"os"
	"path"
	"sort"

	"github.com/notnil/chess/image"
)

var (
	// chessTemplate = "chessTemplate.html"
	chessTemplate = os.Getenv("HOME") + "/chessTemplate.html"
)

// The following structs are a bit funky, but they were built this way
// so that when passed to HTML template file, games can be displayed in groups of three.

// htmlData has all the data needed to build out the html template.
type htmlData struct {
	CurrGameSlices     []gameHTMLSlice
	FinishedGameSlices []gameHTMLSlice
	Stats              []userStats
}

// gameHTMLSlice contains the actual games.
type gameHTMLSlice struct {
	Games []gameHTML
}

// gameHTML represents a single game
type gameHTML struct {
	Black string
	White string
	Image string
}

// getGameHTML takes a game and converts into a gameHtml to be passed to
// html template file.
func getGameHTML(g chessGame) (gameHTML, error) {

	// Get white player
	white := g.PgnParsed.White

	// Get black player
	black := g.PgnParsed.Black

	// Mark will be used to represent the last move made.
	// By default, there will be no markings.
	yellow := color.RGBA{255, 255, 0, 1}
	mark := image.MarkSquares(yellow)

	// If at least one move has been made, mark the last move
	moves := g.ChessGame.Moves()
	if len(moves) > 0 {
		lastMove := moves[len(moves)-1]
		mark = image.MarkSquares(yellow, lastMove.S1(), lastMove.S2())
	}

	// Initialize buffer to write the SVG image of the chess board
	svgBuffer := bytes.Buffer{}

	// Write board SVG to buffer
	board := g.ChessGame.Position().Board()
	if err := image.SVG(&svgBuffer, board, mark); err != nil {
		return gameHTML{}, fmt.Errorf("could not get svg file: %w", err)
	}

	// Base64 encode the SVG image to be able to embed in HTML file
	svgBytes := svgBuffer.Bytes()
	svgBase64 := base64.StdEncoding.EncodeToString(svgBytes)

	return gameHTML{
		Black: black,
		White: white,
		Image: svgBase64,
	}, nil
}

func getGameHTMLSlice(games []chessGame) ([]gameHTMLSlice, error) {
	// Slice which will contain the games slices
	htmlGames := make([]gameHTMLSlice, 0)

	// Individual slice of games.
	currGameSlice := make([]gameHTML, 0)

	// Loop though games passed, get the gameHtml represenation
	// which will be passed into the HTML template file,
	// and build slices of 3 gaames at a time to be displayed
	// in rows of 3 in the webpage.
	for i, game := range games {

		// Get the gameHtml represenation
		// which will be passed into the HTML template file
		gHtml, err := getGameHTML(game)
		if err != nil {
			return nil, fmt.Errorf("could not get gameHtml: %w", err)
		}

		// Append to the individual slice of games
		currGameSlice = append(currGameSlice, gHtml)

		// If the slice already has 3 games OR this is the last
		// game being looked at, append it to
		// the slice that contains all the slices, and reset
		isLastGame := i == len(games)-1
		if len(currGameSlice) == 3 || isLastGame {

			tempGameSlice := gameHTMLSlice{
				Games: currGameSlice,
			}

			htmlGames = append(htmlGames, tempGameSlice)
			currGameSlice = make([]gameHTML, 0)
		}
	}

	return htmlGames, nil
}

// getIndexHTML takes a slice of games and returns an HTML
// webpage using chessTemplate.html as a template file.
func getIndexHTML(currentGames []chessGame, finishedGames []chessGame, stats []userStats) ([]byte, error) {

	currentGamesHTMLSlices, err := getGameHTMLSlice(currentGames)
	if err != nil {
		return nil, err
	}

	finishedGamesHTMLSlices, err := getGameHTMLSlice(finishedGames)
	if err != nil {
		return nil, err
	}

	sort.Sort(userStatsByPointsDesc(stats))

	// Initialize the gameSlices object which will be passed
	// into the html template file
	data := htmlData{
		CurrGameSlices:     currentGamesHTMLSlices,
		FinishedGameSlices: finishedGamesHTMLSlices,
		Stats:              stats,
	}

	// Parse the HTML template file
	name := path.Base(chessTemplate)
	tmplt, err := template.New(name).ParseFiles(chessTemplate)
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
