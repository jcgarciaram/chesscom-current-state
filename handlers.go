package main

import (
	"fmt"
	"net/http"
)

func getHomepage(w http.ResponseWriter, r *http.Request) {

	users := []string{
		"PipoGambit",
		"dalmu7",
		"elcubanoaj",
		// "cdalmeida",
	}

	unfinishedGameGroups := getUnfinishedGamesForUsers(users)
	finishedGameGroups := getFinishedGamesForUsers(users)

	// Finally, get HTML page to display the selectGames
	htmlBytes, err := getIndexHTML(unfinishedGameGroups, finishedGameGroups)
	if err != nil {
		http.Error(w, fmt.Sprintf("There was an error processing your request: %s", err), http.StatusInternalServerError)
		return
	}

	// Write HTML page back to caller
	w.Write(htmlBytes)
}

func getFaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, faviconFile)
}
