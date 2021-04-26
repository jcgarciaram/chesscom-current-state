package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/color"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/notnil/chess"
	"github.com/notnil/chess/image"
)

const (
	PgnResultWhiteWin   = "1-0"
	PgnResultBlackWin   = "0-1"
	PgnResultDraw       = "1/2-1/2"
	PgnResultInProgress = "*"

	ChessComResultWin        = "win"
	ChessComResultCheckmated = "checkmated"
	ChessComResultResigned   = "resigned"
	ChessComResultTimeout    = "timeout"
)

type chessGamesByEndTimeDesc []chessGame

func (a chessGamesByEndTimeDesc) Len() int      { return len(a) }
func (a chessGamesByEndTimeDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a chessGamesByEndTimeDesc) Less(i, j int) bool {
	return a[i].PgnParsed.ParsedEndtime.After(a[j].PgnParsed.ParsedEndtime)
}

type chessGame struct {
	ChessComFinishedGame *chessComFinishedGame
	ChessGame            *chess.Game `json:"-"`
	PgnParsed            pgnParsed   `json:"-"`
	URL                  string      `json:"-"`
	Image                string      `json:"-"`
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

	// Calculated fields
	ParsedEndtime      time.Time
	WhiteWon           bool
	BlackWon           bool
	WhiteWasCheckmated bool
	BlackWasCheckmated bool
	WhiteResigned      bool
	BlackResigned      bool
	WhiteTimedOut      bool
	BlackTimedOut      bool
	Draw               bool
}

type archiveResponse struct {
	Archives []string `json:"archives"`
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

	if parsedPgn.Result == PgnResultWhiteWin {
		parsedPgn.WhiteWon = true
	} else if parsedPgn.Result == PgnResultBlackWin {
		parsedPgn.BlackWon = true
	}

	game := chessGame{
		ChessGame: parsedChessGame,
		PgnParsed: parsedPgn,
	}

	return game, nil
}

type gameGroup struct {
	Month          time.Month
	Year           int
	ChessGames     []chessGame
	UserStatistics []userStats
}

type gameGroupsByYearMonthDesc []gameGroup

func (a gameGroupsByYearMonthDesc) Len() int      { return len(a) }
func (a gameGroupsByYearMonthDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a gameGroupsByYearMonthDesc) Less(i, j int) bool {

	if a[i].Year == a[j].Year {
		return a[i].Month > a[j].Month
	}

	return a[i].Year > a[j].Year
}

func getUnfinishedGamesForUsers(users []string) []gameGroup {
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

	return groupGamesForUsersByMonth(users, allGames)
}

func getFinishedGamesForUsers(users []string) []gameGroup {
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

	return groupGamesForUsersByMonth(users, allGames)
}

func groupGamesForUsersByMonth(users []string, allGames []chessGame) []gameGroup {

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

	type gameGroupWithStatsMap struct {
		gameGroup
		userStatsMap map[string]userStats
	}

	var err error
	gameGroupMap := make(map[string]gameGroupWithStatsMap)
	for _, game := range selectGames {

		game.Image, err = getGameImage(game)
		if err != nil {
			log.Printf("could not get game image: %s\n", err)
		}

		month := game.PgnParsed.ParsedEndtime.Month()
		year := game.PgnParsed.ParsedEndtime.Year()

		key := strconv.Itoa(year) + strconv.Itoa(int(month))
		group, ok := gameGroupMap[key]
		if !ok {

			// Initialize userStats map to be returned
			userStatsMap := make(map[string]userStats)
			for _, user := range users {
				userStatsMap[strings.ToLower(user)] = userStats{
					User: user,
				}
			}

			group = gameGroupWithStatsMap{
				gameGroup: gameGroup{
					Month: month,
					Year:  year,
				},
				userStatsMap: userStatsMap,
			}
		}
		userStatsMap := group.userStatsMap

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

		group.ChessGames = append(group.ChessGames, game)
		group.userStatsMap = userStatsMap
		gameGroupMap[key] = group
	}

	gameGroupSlice := make([]gameGroup, len(gameGroupMap))
	groupIndex := 0
	for _, group := range gameGroupMap {

		statsSlice := make([]userStats, 0)
		for _, stats := range group.userStatsMap {

			totalGamesPlayed := float64(stats.Wins) + float64(stats.Losses) + float64(stats.Draws)

			if totalGamesPlayed == 0 {
				continue
			}

			wins := float64(stats.Wins)

			stats.WinPercentage = math.Round(100*(wins/totalGamesPlayed)*100.0) / 100

			statsSlice = append(statsSlice, stats)
		}

		sort.Sort(userStatsByWinPercDesc(statsSlice))

		group.UserStatistics = statsSlice
		gameGroupSlice[groupIndex] = group.gameGroup
		groupIndex++

	}

	sort.Sort(gameGroupsByYearMonthDesc(gameGroupSlice))

	return gameGroupSlice
}

// getGameImage takes a game and returns the Image with a base64
// encoding of the svg file
func getGameImage(g chessGame) (string, error) {

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
	err := image.SVG(&svgBuffer, board, mark)
	if err != nil {
		return "", fmt.Errorf("could not get svg file: %w", err)
	}

	// Base64 encode the SVG image to be able to embed in HTML file
	svgBytes := svgBuffer.Bytes()
	svgBase64 := base64.StdEncoding.EncodeToString(svgBytes)

	return svgBase64, nil
}
