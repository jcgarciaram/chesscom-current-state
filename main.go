package main

import (
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {

	users := []string{
		"PipoGambit",
		"dalmu7",
		"elcubanoaj",
	}

	unfinishedGames, _ := getUnfinishedGamesForUsers(users)
	finishedGames, stats := getFinishedGamesForUsers(users)

	// Finally, get HTML page to display the selectGames
	htmlBytes, err := getIndexHTML(unfinishedGames, finishedGames, stats)
	if err != nil {
		log.Fatal(err)
	}

	// Write HTML page back to caller
	w.Write(htmlBytes)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, faviconFile)
}

func main() {
	// Super simple web server.
	// Only has a single handler which simply serves an html page.
	http.HandleFunc("/", handler)
	http.HandleFunc("/favicon.ico", faviconHandler)

	// Listen for connections on port 8889
	log.Fatal(http.ListenAndServe(":8889", nil))
}
