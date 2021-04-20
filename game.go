package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/notnil/chess"
)

// Chess.com response when getting games for
// an individual user.
type userGames struct {
	Games []game `json:"games"`
}

// Chess.com game.
type game struct {
	URL          string `json:"url"`
	MoveBy       int    `json:"move_by"`
	Pgn          string `json:"pgn"`
	TimeControl  string `json:"time_control"`
	LastActivity int    `json:"last_activity"`
	Rated        bool   `json:"rated"`
	Turn         string `json:"turn"`
	Fen          string `json:"fen"`
	StartTime    int    `json:"start_time"`
	TimeClass    string `json:"time_class"`
	Rules        string `json:"rules"`
	White        string `json:"white"`
	Black        string `json:"black"`

	// Calculated fields
	ChessGame *chess.Game `json:"-"`
	ID        string      `json:"-"`
}

// Call chess.com API to get the games for the passed username.
// This function will also go ahead and reag the PGN for the game
// and populate ChessGame field on game struct.
func getUserGames(username string) ([]game, error) {

	// Get games from chess.com API
	games := userGames{}
	resp, err := http.Get(fmt.Sprintf("https://api.chess.com/pub/player/%s/games", username))
	if err != nil {
		return games.Games, fmt.Errorf("could not get games for username %s: %w", username, err)
	}

	// get the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return games.Games, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
	}

	// Unmarshal response body to struct above
	err = json.Unmarshal(respBody, &games)
	if err != nil {
		return games.Games, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
	}

	// Loop through all games and build a ChessGame from PGN.
	for i := 0; i < len(games.Games); i++ {
		chessGame, err := readPgn(games.Games[i].Pgn)
		if err != nil {
			return games.Games, fmt.Errorf("could not read Pgn for game for username %s: %w", username, err)
		}

		games.Games[i].ChessGame = chessGame
	}

	return games.Games, nil
}

// readPgn takes a PGN string and returns a chess.Game pointer.
func readPgn(pgnString string) (*chess.Game, error) {
	pgnReader := strings.NewReader(pgnString)
	pgn, err := chess.PGN(pgnReader)
	if err != nil {
		return &chess.Game{}, fmt.Errorf("could not read pgn: %w", err)
	}

	return chess.NewGame(pgn), nil
}
