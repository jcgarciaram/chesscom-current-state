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

type archiveResponse struct {
	Archives []string `json:"archives"`
}

// Call chess.com API to get the games for the passed username.
// This function will also go ahead and reag the PGN for the game
// and populate ChessGame field on game struct.
func getUserUnfinishedGames(username string) ([]game, error) {

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

// Call chess.com API to get the finished games for the passed username.
// This function will also go ahead and reag the PGN for the game
// and populate ChessGame field on game struct.
func getUserFinishedGames(username string) ([]game, error) {

	// Get archival url games from chess.com API
	resp, err := http.Get(fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", username))
	if err != nil {
		return []game{}, fmt.Errorf("could not get games for username %s: %w", username, err)
	}

	// get the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []game{}, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
	}

	// Unmarshal response body to struct above
	archives := archiveResponse{}
	err = json.Unmarshal(respBody, &archives)
	if err != nil {
		return []game{}, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
	}

	games := []game{}
	// Loop through all archive URLs and get all finished games
	for _, archiveURL := range archives.Archives {

		resp, err := http.Get(archiveURL)
		if err != nil {
			return []game{}, fmt.Errorf("could not get games for username %s: %w", username, err)
		}

		// get the response body
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []game{}, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
		}

		gamesForUser := userGames{}
		// Unmarshal response body to struct above
		err = json.Unmarshal(respBody, &gamesForUser)
		if err != nil {
			return []game{}, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
		}

		// Loop through all games and build a ChessGame from PGN.
		for i := 0; i < len(gamesForUser.Games); i++ {
			chessGame, err := readPgn(gamesForUser.Games[i].Pgn)
			if err != nil {
				return []game{}, fmt.Errorf("could not read Pgn for game for username %s: %w", username, err)
			}

			gamesForUser.Games[i].ChessGame = chessGame
		}

		games = append(games, gamesForUser.Games...)
	}

	return games, nil
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

func getUnfinishedGamesForUsers(users []string) []game {
	// Loop through all users that are in the chess club
	// and get all their current games.
	// This will include games against players not in the club
	// which will be filtered out later.
	allGames := []game{}
	for _, user := range users {
		games, err := getUserUnfinishedGames(user)
		if err != nil {
			continue
		}
		allGames = append(allGames, games...)
	}

	return filterGamesForUsers(users, allGames)
}

func getFinishedGamesForUsers(users []string) []game {
	// Loop through all users that are in the chess club
	// and get all their finished games.
	// This will include games against players not in the club
	// which will be filtered out later.
	allGames := []game{}
	for _, user := range users {
		games, err := getUserFinishedGames(user)
		if err != nil {
			continue
		}
		allGames = append(allGames, games...)
	}

	return filterGamesForUsers(users, allGames)
}

func filterGamesForUsers(users []string, allGames []game) []game {

	// Build a game ID map to keep track of games we have already seen.
	// We only want to include unique games once.
	gameIDMap := make(map[string]struct{})

	// Loop through all the games and only keep those which are
	// between two members of the club.
	// Store these in selectGames.
	selectGames := []game{}
	for _, game := range allGames {

		// While we're looping, go ahead and split the
		// game URL to get the ID
		gameURLSplit := strings.Split(game.URL, "/")
		if len(gameURLSplit) == 0 {
			continue
		}

		game.ID = gameURLSplit[len(gameURLSplit)-1]

		// Verify if we have seen this game before.
		// If we have, continue with the next gaae
		_, ok := gameIDMap[game.URL]
		if ok {
			continue
		}
		gameIDMap[game.URL] = struct{}{}

		// OK, so we have not seen this game before.
		// Now let's check if it's a game between 2 users of the club.
		// Loop through all the users in the club and match
		// see if Black AND White is a user in the club.
		usernamesFound := 0
		for _, user := range users {
			if strings.Contains(game.Black, user) {
				usernamesFound++
			}

			if strings.Contains(game.White, user) {
				usernamesFound++
			}

			if usernamesFound == 2 {
				break
			}
		}

		// If both Black and white are users in the club,
		// then keep this game
		if usernamesFound == 2 {
			selectGames = append(selectGames, game)
		}

	}

	return selectGames
}
