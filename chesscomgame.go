package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
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

func getUserFinishedGamesForYearMonth(username string, year, month int) ([]chessGame, string, error) {
	archives, err := getUserArchivalURLs(username)
	if err != nil {
		return []chessGame{}, "", fmt.Errorf("could not get user archival urls %s: %w", username, err)
	}

	yearMonthStringToFind := fmt.Sprintf("%04d%02d", year, month)

	urlForYearMonthToFind := ""
	maxYearMonthPriorToFind := "190001"
	for _, archiveURL := range archives.Archives {
		urlSplit := strings.Split(archiveURL, "/")
		if len(urlSplit) < 2 {
			continue
		}

		yearString := urlSplit[len(urlSplit)-2]
		monthString := urlSplit[len(urlSplit)-1]

		currYearMonthString := fmt.Sprintf("%04s%02s", yearString, monthString)

		if yearMonthStringToFind == currYearMonthString {
			urlForYearMonthToFind = archiveURL
		}

		if currYearMonthString < yearMonthStringToFind && currYearMonthString > maxYearMonthPriorToFind {
			maxYearMonthPriorToFind = currYearMonthString
		}
	}

	games := []chessGame{}
	if urlForYearMonthToFind != "" {
		games, err = getFinishedGamesWithURL(username, urlForYearMonthToFind)
		if err != nil {
			return []chessGame{}, "", fmt.Errorf("could not get finished games with url %s: %w", username, err)
		}
	}

	return games, maxYearMonthPriorToFind, nil
}

// Call chess.com API to get the finished games for the passed username.
// This function will also go ahead and reag the PGN for the game
// and populate ChessGame field on game struct.
func getUserFinishedGames(username string) ([]chessGame, error) {

	archives, err := getUserArchivalURLs(username)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not get user archival urls %s: %w", username, err)
	}

	chessGames := []chessGame{}

	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}

	// Loop through all archive URLs and get all finished games
	for _, archiveURL := range archives.Archives {

		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			attempts := 5
			for i := 0; i < attempts; i++ {

				pgnChessGames, err := getFinishedGamesWithURL(username, url)
				if err != nil {
					logrus.WithError(err).WithField("attempt loop", i).Warn("could not get finished games with url")
					continue
				}

				mutex.Lock()
				chessGames = append(chessGames, pgnChessGames...)
				mutex.Unlock()

				break
			}
		}(archiveURL)
	}

	wg.Wait()

	return chessGames, nil
}

func getUserArchivalURLs(username string) (archiveResponse, error) {
	c := &http.Client{
		Timeout: time.Second * 5,
	}

	// Get archival url games from chess.com API
	res, err := c.Get(fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", username))
	if err != nil {
		return archiveResponse{}, fmt.Errorf("could not get archives games for username %s: %w", username, err)
	}

	defer res.Body.Close()

	// get the response body
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return archiveResponse{}, fmt.Errorf("could not read response body for archives for username %s: %w", username, err)
	}

	// Unmarshal response body to struct above
	archives := archiveResponse{}
	err = json.Unmarshal(resBody, &archives)
	if err != nil {
		return archiveResponse{}, fmt.Errorf("could not unmarshal response body for archives for username %s: %w", username, err)
	}

	return archives, nil
}

func getFinishedGamesWithURL(username, url string) ([]chessGame, error) {
	c2 := &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := c2.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not get finished games. url (%s) for username %s: %s", url, username, err)
	}

	defer resp.Body.Close()

	// get the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body for finished games. url (%s) for username %s: %s", url, username, err)
	}

	gamesForUser := chessComFinishedUserGames{}
	// Unmarshal response body to struct above
	err = json.Unmarshal(respBody, &gamesForUser)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal response body for finished games. url (%s) for username %s: %s", url, username, err)
	}

	pngChessGames := make([]chessGame, len(gamesForUser.Games))
	// Loop through all games and build a ChessGame from PGN.
	for i := 0; i < len(gamesForUser.Games); i++ {
		pgnChessGame, err := getChessGame(gamesForUser.Games[i].Pgn)
		if err != nil {
			return nil, fmt.Errorf("could not read Pgn for game for username %s: %s", username, err)
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
		} else if gamesForUser.Games[i].Black.Result == ChessComResultAgreed {
			pgnChessGame.PgnParsed.BlackAgreed = true
		} else if gamesForUser.Games[i].Black.Result == ChessComResultInsufficient {
			pgnChessGame.PgnParsed.BlackInsufficient = true
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
		} else if gamesForUser.Games[i].White.Result == ChessComResultAgreed {
			pgnChessGame.PgnParsed.WhiteAgreed = true
		} else if gamesForUser.Games[i].White.Result == ChessComResultInsufficient {
			pgnChessGame.PgnParsed.WhiteInsufficient = true
		}

		if pgnChessGame.PgnParsed.Result == PgnResultDraw {
			pgnChessGame.PgnParsed.Draw = true
		}

		pgnChessGame.ChessComFinishedGame = &gamesForUser.Games[i]
		pgnChessGame.URL = gamesForUser.Games[i].URL

		pngChessGames[i] = pgnChessGame
	}

	return pngChessGames, nil
}
