package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image/color"
	"os"
	"path"
	"strings"

	"github.com/notnil/chess/image"
)

var (
	// chessTemplate = "chessTemplate.html"
	chessTemplate = os.Getenv("HOME") + "/chessTemplate.html"
)

type gamesHtml struct {
	GameSlices []gameSlice
}

type gameSlice struct {
	Games []gameHtml
}

type gameHtml struct {
	Black string
	White string
	Image string
	ID    string
}

func getGameHTML(g game) (gameHtml, error) {
	whiteSplit := strings.Split(g.White, "/")
	white := whiteSplit[len(whiteSplit)-1]

	blackSplit := strings.Split(g.Black, "/")
	black := blackSplit[len(blackSplit)-1]

	// default mark to not have any markings
	yellow := color.RGBA{255, 255, 0, 1}
	mark := image.MarkSquares(yellow)

	// if at least one move has been made, mark the last move
	moves := g.ChessGame.Moves()
	if len(moves) > 0 {
		lastMove := moves[len(moves)-1]
		mark = image.MarkSquares(yellow, lastMove.S1(), lastMove.S2())
	}

	svgBuffer := bytes.Buffer{}

	// write board SVG to file
	board := g.ChessGame.Position().Board()
	if err := image.SVG(&svgBuffer, board, mark); err != nil {
		return gameHtml{}, fmt.Errorf("could not get svg file: %w", err)
	}

	svgBytes := svgBuffer.Bytes()

	svgBase64 := base64.StdEncoding.EncodeToString(svgBytes)

	return gameHtml{
		Black: black,
		White: white,
		Image: svgBase64,
		ID:    g.ID,
	}, nil
}

func getIndexHTML(games []game) ([]byte, error) {

	htmlGames := make([]gameSlice, 0)
	gamesInCurrentSlice := 0
	currGameSlice := make([]gameHtml, 0)
	for _, game := range games {

		gHtml, err := getGameHTML(game)
		if err != nil {
			return nil, fmt.Errorf("could not get gameHtml: %w", err)
		}

		currGameSlice = append(currGameSlice, gHtml)
		gamesInCurrentSlice++

		if gamesInCurrentSlice == 3 {

			tempGameSlice := gameSlice{
				Games: currGameSlice,
			}

			htmlGames = append(htmlGames, tempGameSlice)
			gamesInCurrentSlice = 0
			currGameSlice = make([]gameHtml, 0)
		}
	}

	data := gamesHtml{htmlGames}

	name := path.Base(chessTemplate)
	tmplt, err := template.New(name).ParseFiles(chessTemplate)
	if err != nil {
		return nil, fmt.Errorf("could not parse file template: %w", err)
	}

	outputParsed := bytes.Buffer{}
	err = tmplt.Execute(&outputParsed, data)
	if err != nil {
		return nil, fmt.Errorf("could not execute file template: %w", err)
	}

	return outputParsed.Bytes(), nil
}
