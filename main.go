package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/notnil/chess/image"
)

func main() {
	pipoGambitGames, err := getUserGames("pipogambit")
	if err != nil {
		log.Fatal(err)
	}

	for _, game := range pipoGambitGames {

		whiteSplit := strings.Split(game.White, "/")
		white := whiteSplit[len(whiteSplit)-1]

		blackSplit := strings.Split(game.Black, "/")
		black := blackSplit[len(blackSplit)-1]

		// create file
		f, err := os.Create(fmt.Sprintf("/Users/jgarcia/Flourish/go/src/github.com/jcgarciaram/chess-curr-state/%s.svg", white+"_"+black+"_"+strconv.Itoa(game.StartTime)))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		// default mark to not have any markings
		yellow := color.RGBA{255, 255, 0, 1}
		mark := image.MarkSquares(yellow)

		// if at least one move has been made, mark the last move
		moves := game.ChessGame.Moves()
		if len(moves) > 0 {
			lastMove := moves[len(moves)-1]
			mark = image.MarkSquares(yellow, lastMove.S1(), lastMove.S2())
		}

		// write board SVG to file
		board := game.ChessGame.Position().Board()
		if err := image.SVG(f, board, mark); err != nil {
			log.Fatal(err)
		}
	}

}
