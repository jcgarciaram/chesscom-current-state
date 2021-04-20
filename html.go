package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image/color"
	"log"
	"os"
	"path"
	"strings"

	"github.com/notnil/chess/image"
)

var (
	chessTemplate = os.Getenv("HOME") + "/chessTemplate.html"
)

type gamesHtml struct {
	Games []*gameHtml
}

type gameHtml struct {
	Black string
	White string
	Image string
	ID    string
}

func getIndexHTML(games []game) ([]byte, error) {

	htmlGames := make([]*gameHtml, len(games))
	for i, game := range games {

		whiteSplit := strings.Split(game.White, "/")
		white := whiteSplit[len(whiteSplit)-1]

		blackSplit := strings.Split(game.Black, "/")
		black := blackSplit[len(blackSplit)-1]

		// default mark to not have any markings
		yellow := color.RGBA{255, 255, 0, 1}
		mark := image.MarkSquares(yellow)

		// if at least one move has been made, mark the last move
		moves := game.ChessGame.Moves()
		if len(moves) > 0 {
			lastMove := moves[len(moves)-1]
			mark = image.MarkSquares(yellow, lastMove.S1(), lastMove.S2())
		}

		svgBuffer := bytes.Buffer{}

		// write board SVG to file
		board := game.ChessGame.Position().Board()
		if err := image.SVG(&svgBuffer, board, mark); err != nil {
			log.Fatal(err)
		}

		svgBytes := svgBuffer.Bytes()

		svgBase64 := base64.StdEncoding.EncodeToString(svgBytes)

		htmlGames[i] = &gameHtml{
			Black: black,
			White: white,
			Image: svgBase64,
			ID:    game.ID,
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

// coverName := path.Base(coverFilePath)
// coverTmpl, err := template.New(coverName).Funcs(funcMap).ParseFiles(coverFilePath)
// if err != nil {
// 	return nil, fmt.Errorf("could not parse conver file template: %w", err)
// }

// coverOutputParsed := bytes.Buffer{}
// err = coverTmpl.Execute(&coverOutputParsed, data)
// if err != nil {
// 	return nil, fmt.Errorf("could not execute cover file template: %w", err)
// }
