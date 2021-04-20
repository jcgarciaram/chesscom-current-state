package main

import (
	"log"
	"net/http"
	"strings"
)

var (
	users = []string{
		"pipogambit",
		"dalmu7",
		"elcubanoaj",
	}
)

func handler(w http.ResponseWriter, r *http.Request) {

	// Loop through all users that are in the chess club
	// and get all their current games.
	// This will include games against players not in the club
	// which will be filtered out later.
	allGames := []game{}
	for _, user := range users {
		games, err := getUserGames(user)
		if err != nil {
			continue
		}
		allGames = append(allGames, games...)
	}

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

	// Finally, get HTML page to display the selectGames
	htmlBytes, err := getIndexHTML(selectGames)
	if err != nil {
		log.Fatal(err)
	}

	// Write HTML page back to caller
	w.Write(htmlBytes)
}

func main() {
	// Super simple web server.
	// Only has a single handler which simply serves an html page.
	http.HandleFunc("/", handler)

	// Listen for connections on port 8889
	log.Fatal(http.ListenAndServe(":8889", nil))
}
