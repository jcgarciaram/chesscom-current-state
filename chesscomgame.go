package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Chess.com response when getting games for
// an individual user.
type chessComCurrentUserGames struct {
	Games []chessComCurrentGame `json:"games"`
}

// Chess.com chessComCurrentGame.
type chessComCurrentGame struct {
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
}

// Chess.com response when getting games for
// an individual user.
type chessComFinishedUserGames struct {
	Games []chessComFinishedGame `json:"games"`
}

type chessComFinishedGame struct {
	URL         string `json:"url"`
	Pgn         string `json:"pgn"`
	TimeControl string `json:"time_control"`
	EndTime     int    `json:"end_time"`
	Rated       bool   `json:"rated"`
	Fen         string `json:"fen"`
	StartTime   int    `json:"start_time"`
	TimeClass   string `json:"time_class"`
	Rules       string `json:"rules"`
	White       struct {
		Rating   int    `json:"rating"`
		Result   string `json:"result"`
		ID       string `json:"@id"`
		Username string `json:"username"`
	} `json:"white"`
	Black struct {
		Rating   int    `json:"rating"`
		Result   string `json:"result"`
		ID       string `json:"@id"`
		Username string `json:"username"`
	} `json:"black"`
}

// Call chess.com API to get the games for the passed username.
// This function will also go ahead and reag the PGN for the game
// and populate ChessGame field on game struct.
func getUserUnfinishedGames(username string) ([]chessGame, error) {

	c := &http.Client{
		Timeout: time.Second * 5,
	}

	// Get games from chess.com API
	resp, err := c.Get(fmt.Sprintf("https://api.chess.com/pub/player/%s/games", username))
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not get current games for username %s: %w", username, err)
	}

	defer resp.Body.Close()

	// get the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not read response body for current games for username %s: %w", username, err)
	}

	// Unmarshal response body to struct above
	games := chessComCurrentUserGames{}
	err = json.Unmarshal(respBody, &games)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not unmarshal response body for current games for username %s: %w", username, err)
	}

	chessGames := make([]chessGame, len(games.Games))
	// Loop through all games and build a ChessGame from PGN.
	for i := 0; i < len(games.Games); i++ {
		pgnChessGame, err := getChessGame(games.Games[i].Pgn)
		if err != nil {
			return []chessGame{}, fmt.Errorf("could not read Pgn for game for username %s: %w", username, err)
		}

		pgnChessGame.URL = games.Games[i].URL

		chessGames[i] = pgnChessGame
	}

	return chessGames, nil
}

// Call chess.com API to get the finished games for the passed username.
// This function will also go ahead and reag the PGN for the game
// and populate ChessGame field on game struct.
func getUserFinishedGames(username string) ([]chessGame, error) {

	c := &http.Client{
		Timeout: time.Second * 5,
	}

	// Get archival url games from chess.com API
	res, err := c.Get(fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", username))
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not get archives games for username %s: %w", username, err)
	}

	defer res.Body.Close()

	// get the response body
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not read response body for archives for username %s: %w", username, err)
	}

	// Unmarshal response body to struct above
	archives := archiveResponse{}
	err = json.Unmarshal(resBody, &archives)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not unmarshal response body for archives for username %s: %w", username, err)
	}

	chessGames := []chessGame{}

	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}

	// Loop through all archive URLs and get all finished games
	for _, archiveURL := range archives.Archives {

		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			c2 := &http.Client{
				Timeout: time.Second * 5,
			}

			resp, err := c2.Get(url)
			if err != nil {
				log.Printf("could not get finished games. url (%s) for username %s: %s", url, username, err)
				return
			}

			defer resp.Body.Close()

			// get the response body
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("could not read response body for finished games. url (%s) for username %s: %s", url, username, err)
				return
			}

			gamesForUser := chessComFinishedUserGames{}
			// Unmarshal response body to struct above
			err = json.Unmarshal(respBody, &gamesForUser)
			if err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"url":      url,
					"username": username,
					"respBody": string(respBody),
				}).Info("could not unmarshal response body for finished games")
				return
			}

			// Loop through all games and build a ChessGame from PGN.
			for i := 0; i < len(gamesForUser.Games); i++ {
				pgnChessGame, err := getChessGame(gamesForUser.Games[i].Pgn)
				if err != nil {
					log.Printf("could not read Pgn for game for username %s: %s", username, err)
					return
				}

				// Set boolean fields for HTML rendering for black
				if gamesForUser.Games[i].Black.Result == ChessComResultWin {
					pgnChessGame.PgnParsed.BlackWon = true
				} else if gamesForUser.Games[i].Black.Result == ChessComResultCheckmated {
					pgnChessGame.PgnParsed.BlackWasCheckmated = true
				} else if gamesForUser.Games[i].Black.Result == ChessComResultResigned {
					pgnChessGame.PgnParsed.BlackResigned = true
				} else if gamesForUser.Games[i].Black.Result == ChessComResultTimeout {
					pgnChessGame.PgnParsed.BlackTimedOut = true
				}

				// Set boolean fields for HTML rendering for white
				if gamesForUser.Games[i].White.Result == ChessComResultWin {
					pgnChessGame.PgnParsed.WhiteWon = true
				} else if gamesForUser.Games[i].White.Result == ChessComResultCheckmated {
					pgnChessGame.PgnParsed.WhiteWasCheckmated = true
				} else if gamesForUser.Games[i].White.Result == ChessComResultResigned {
					pgnChessGame.PgnParsed.WhiteResigned = true
				} else if gamesForUser.Games[i].White.Result == ChessComResultTimeout {
					pgnChessGame.PgnParsed.WhiteTimedOut = true
				}

				if pgnChessGame.PgnParsed.Result == PgnResultDraw {
					pgnChessGame.PgnParsed.Draw = true
				}

				pgnChessGame.ChessComFinishedGame = &gamesForUser.Games[i]
				pgnChessGame.URL = gamesForUser.Games[i].URL

				mutex.Lock()
				chessGames = append(chessGames, pgnChessGame)
				mutex.Unlock()
			}
		}(archiveURL)
	}

	wg.Wait()

	return chessGames, nil
}
