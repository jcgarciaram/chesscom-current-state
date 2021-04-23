package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/notnil/chess"
)

const (
	PgnResultWhiteWin   = "1-0"
	PgnResultBlackWin   = "0-1"
	PgnResultDraw       = "1/2-1/2"
	PgnResultInProgress = "*"
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

type chessGamesByEndTimeDesc []chessGame

func (a chessGamesByEndTimeDesc) Len() int      { return len(a) }
func (a chessGamesByEndTimeDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a chessGamesByEndTimeDesc) Less(i, j int) bool {
	return a[i].PgnParsed.ParsedEndtime.After(a[j].PgnParsed.ParsedEndtime)
}

type chessGame struct {
	ChessGame *chess.Game `json:"-"`
	PgnParsed pgnParsed   `json:"-"`
	URL       string      `json:"-"`
}

type pgnParsed struct {
	Event           string
	Site            string
	Date            string
	Round           string
	White           string
	Black           string
	Result          string
	CurrentPosition string
	Timezone        string
	ECO             string
	ECOUrl          string
	UTCDate         string
	UTCTime         string
	WhiteElo        string
	BlackElo        string
	TimeControl     string
	Termination     string
	StartTime       string
	EndDate         string
	EndTime         string
	Link            string

	ParsedEndtime time.Time
}

type archiveResponse struct {
	Archives []string `json:"archives"`
}

// Call chess.com API to get the games for the passed username.
// This function will also go ahead and reag the PGN for the game
// and populate ChessGame field on game struct.
func getUserUnfinishedGames(username string) ([]chessGame, error) {

	// Get games from chess.com API
	resp, err := http.Get(fmt.Sprintf("https://api.chess.com/pub/player/%s/games", username))
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not get games for username %s: %w", username, err)
	}

	// get the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
	}

	// Unmarshal response body to struct above
	games := chessComCurrentUserGames{}
	err = json.Unmarshal(respBody, &games)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
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

	// Get archival url games from chess.com API
	resp, err := http.Get(fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", username))
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not get games for username %s: %w", username, err)
	}

	// get the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
	}

	// Unmarshal response body to struct above
	archives := archiveResponse{}
	err = json.Unmarshal(respBody, &archives)
	if err != nil {
		return []chessGame{}, fmt.Errorf("could not read response body for games for username %s: %w", username, err)
	}

	chessGames := []chessGame{}

	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}

	// Loop through all archive URLs and get all finished games
	for _, archiveURL := range archives.Archives {

		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("could not get games for username %s: %s", username, err)
				return
			}

			// get the response body
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("could not read response body for games for username %s: %s", username, err)
				return
			}

			gamesForUser := chessComFinishedUserGames{}
			// Unmarshal response body to struct above
			err = json.Unmarshal(respBody, &gamesForUser)
			if err != nil {
				log.Printf("could not read response body for games for username %s: %s", username, err)
				return
			}

			// Loop through all games and build a ChessGame from PGN.
			for i := 0; i < len(gamesForUser.Games); i++ {
				pgnChessGame, err := getChessGame(gamesForUser.Games[i].Pgn)
				if err != nil {
					log.Printf("could not read Pgn for game for username %s: %s", username, err)
					return
				}

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

// getChessGame takes a PGN string and returns a chess.Game pointer.
func getChessGame(pgnString string) (chessGame, error) {
	pgnReader := strings.NewReader(pgnString)
	pgn, err := chess.PGN(pgnReader)
	if err != nil {
		return chessGame{}, fmt.Errorf("could not read pgn: %w", err)
	}

	parsedChessGame := chess.NewGame(pgn)

	parsedPgn := pgnParsed{}
	tagPairs := parsedChessGame.TagPairs()
	for _, tagPair := range tagPairs {
		key := tagPair.Key
		val := tagPair.Value

		if key == "Event" {
			parsedPgn.Event = val
		}
		if key == "Site" {
			parsedPgn.Site = val
		}
		if key == "Date" {
			parsedPgn.Date = val
		}
		if key == "Round" {
			parsedPgn.Round = val
		}
		if key == "White" {
			parsedPgn.White = val
		}
		if key == "Black" {
			parsedPgn.Black = val
		}
		if key == "Result" {
			parsedPgn.Result = val
		}
		if key == "CurrentPosition" {
			parsedPgn.CurrentPosition = val
		}
		if key == "Timezone" {
			parsedPgn.Timezone = val
		}
		if key == "ECO" {
			parsedPgn.ECO = val
		}
		if key == "ECOUrl" {
			parsedPgn.ECOUrl = val
		}
		if key == "UTCDate" {
			parsedPgn.UTCDate = val
		}
		if key == "UTCTime" {
			parsedPgn.UTCTime = val
		}
		if key == "WhiteElo" {
			parsedPgn.WhiteElo = val
		}
		if key == "BlackElo" {
			parsedPgn.BlackElo = val
		}
		if key == "TimeControl" {
			parsedPgn.TimeControl = val
		}
		if key == "Termination" {
			parsedPgn.Termination = val
		}
		if key == "StartTime" {
			parsedPgn.StartTime = val
		}
		if key == "EndDate" {
			parsedPgn.EndDate = val
		}
		if key == "EndTime" {
			parsedPgn.EndTime = val
		}
		if key == "Link" {
			parsedPgn.Link = val
		}
	}

	if parsedPgn.EndDate != "" && parsedPgn.EndTime != "" {
		format := "2006.01.02 15:04:05"
		parsedEndTime, err := time.Parse(format, parsedPgn.EndDate+" "+parsedPgn.EndTime)
		if err == nil {
			parsedPgn.ParsedEndtime = parsedEndTime
		}
	}

	game := chessGame{
		ChessGame: parsedChessGame,
		PgnParsed: parsedPgn,
	}

	return game, nil
}

func getUnfinishedGamesForUsers(users []string) ([]chessGame, []userStats) {
	// Loop through all users that are in the chess club
	// and get all their current games.
	// This will include games against players not in the club
	// which will be filtered out later.
	allGames := []chessGame{}
	for _, user := range users {
		games, err := getUserUnfinishedGames(user)
		if err != nil {
			continue
		}
		allGames = append(allGames, games...)
	}

	return filterGamesForUsers(users, allGames)
}

func getFinishedGamesForUsers(users []string) ([]chessGame, []userStats) {
	// Loop through all users that are in the chess club
	// and get all their finished games.
	// This will include games against players not in the club
	// which will be filtered out later.
	allGames := []chessGame{}

	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}
	for _, user := range users {
		wg.Add(1)

		go func(u string) {
			defer wg.Done()
			games, err := getUserFinishedGames(u)
			if err != nil {

				log.Printf("%s\n", err)
				return
			}
			mutex.Lock()
			allGames = append(allGames, games...)
			mutex.Unlock()
		}(user)
	}

	wg.Wait()

	return filterGamesForUsers(users, allGames)
}

func filterGamesForUsers(users []string, allGames []chessGame) ([]chessGame, []userStats) {

	// Build a game ID map to keep track of games we have already seen.
	// We only want to include unique games once.
	gameIDMap := make(map[string]struct{})

	// Loop through all the games and only keep those which are
	// between two members of the club.
	// Store these in selectGames.
	selectGames := []chessGame{}
	for _, game := range allGames {

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
			if strings.EqualFold(game.PgnParsed.Black, user) {
				usernamesFound++
			}

			if strings.EqualFold(game.PgnParsed.White, user) {
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

	sort.Sort(chessGamesByEndTimeDesc(selectGames))

	// Initialize userStats map to be returned
	userStatsMap := make(map[string]userStats)
	for _, user := range users {
		userStatsMap[strings.ToLower(user)] = userStats{
			User: user,
		}
	}

	for _, game := range selectGames {

		// Get usernames
		white := game.PgnParsed.White
		black := game.PgnParsed.Black

		whiteStats := userStatsMap[strings.ToLower(white)]
		blackStats := userStatsMap[strings.ToLower(black)]

		if game.PgnParsed.Result == PgnResultWhiteWin {
			whiteStats.Wins++
			whiteStats.Points += 1
			blackStats.Losses++

			if whiteStats.Losses == 0 {
				whiteStats.WinStreak++
			}

		} else if game.PgnParsed.Result == PgnResultBlackWin {
			whiteStats.Losses++
			blackStats.Wins++
			blackStats.Points += 1

			if blackStats.Losses == 0 {
				blackStats.WinStreak++
			}

		} else if game.PgnParsed.Result == PgnResultDraw {
			whiteStats.Draws++
			whiteStats.Points += 0.5
			blackStats.Draws++
			blackStats.Points += 0.5
		}

		userStatsMap[strings.ToLower(white)] = whiteStats
		userStatsMap[strings.ToLower(black)] = blackStats
	}

	statsSlice := make([]userStats, len(userStatsMap))
	index := 0
	for _, stats := range userStatsMap {
		statsSlice[index] = stats
		index++
	}

	return selectGames, statsSlice
}
