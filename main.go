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

	allGames := []game{}
	for _, user := range users {
		games, err := getUserGames(user)
		if err != nil {
			continue
		}
		allGames = append(allGames, games...)
	}

	gameIDMap := make(map[string]struct{})
	selectGames := []game{}
	for _, game := range allGames {

		gameURLSplit := strings.Split(game.URL, "/")
		if len(gameURLSplit) == 0 {
			continue
		}

		game.ID = gameURLSplit[len(gameURLSplit)-1]

		_, ok := gameIDMap[game.URL]
		if ok {
			continue
		}
		gameIDMap[game.URL] = struct{}{}

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

		if usernamesFound == 2 {
			selectGames = append(selectGames, game)
		}

	}

	htmlBytes, err := getIndexHTML(selectGames)
	if err != nil {
		log.Fatal(err)
	}

	w.Write(htmlBytes)
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8889", nil))
}
