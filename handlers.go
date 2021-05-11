package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	usernames = []string{
		"PipoGambit",
		"dalmu7",
		"elcubanoaj",
		"cdalmeida",
		"maximuni",
	}
)

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func getGamesForMonthHTML(w http.ResponseWriter, r *http.Request) {

	// Parse form to get query params
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("There was an error processing your request: %s", err), http.StatusInternalServerError)
		return
	}

	// Get monthString
	monthString := r.FormValue("month")
	if monthString == "" {
		http.Error(w, "Invalid month query param passed in request", http.StatusBadRequest)
		return
	}

	// Get yearString
	yearString := r.FormValue("year")
	if yearString == "" {
		http.Error(w, "Invalid year query param passed in request", http.StatusBadRequest)
		return
	}

	// Convert monthString to int
	month, err := strconv.Atoi(monthString)
	if err != nil {
		http.Error(w, "Invalid month query param passed in request", http.StatusBadRequest)
		return
	}

	// Convert yearString to int
	year, err := strconv.Atoi(yearString)
	if err != nil {
		http.Error(w, "Invalid year query param passed in request", http.StatusBadRequest)
		return
	}

	if month < 1 && month > 12 {
		http.Error(w, "Invalid year query param passed in request", http.StatusBadRequest)
		return
	}

	finishedGameGroup, nextYear, nextMonth := getFinishedGamesForUsersForYearMonth(usernames, year, month)

	finalFinishedGameGroups := []gameGroup{}
	if finishedGameGroup != nil {
		finalFinishedGameGroups = append(finalFinishedGameGroups, *finishedGameGroup)
	}

	if nextYear > 0 && nextMonth > 0 {

		yearFound := 0
		monthFound := 0
		if finishedGameGroup != nil {
			yearFound = finishedGameGroup.Year
			monthFound = int(finishedGameGroup.Month)
		}

		if year != yearFound || month != monthFound {
			finalFinishedGameGroups = append(finalFinishedGameGroups, gameGroup{
				Year:  year,
				Month: time.Month(month),
			})
		}

		year, month = getPreviousMonth(year, month)

		for nextYear != year || nextMonth != month {

			finalFinishedGameGroups = append(finalFinishedGameGroups, gameGroup{
				Year:  year,
				Month: time.Month(month),
			})
			year, month = getPreviousMonth(year, month)
		}
	}

	if finishedGameGroup == nil && nextYear == 0 && nextMonth == 0 {
		finalFinishedGameGroups = append(finalFinishedGameGroups, gameGroup{
			OverallNoGamesFound: true,
		})
	}

	// Finally, get HTML page to display the selectGames
	htmlBytes, err := getGamesForMonthHTMLBytes(finalFinishedGameGroups)
	if err != nil {
		http.Error(w, fmt.Sprintf("There was an error processing your request: %s", err), http.StatusInternalServerError)
		return
	}

	ret := struct {
		HTML      string `json:"html"`
		NextYear  int    `json:"next_year"`
		NextMonth int    `json:"next_month"`
	}{
		HTML:      string(htmlBytes),
		NextYear:  nextYear,
		NextMonth: nextMonth,
	}

	if err := json.NewEncoder(w).Encode(ret); err != nil {
		logrus.WithError(err).Warn("Error encoding result")
	}
}

func getHomepage(w http.ResponseWriter, r *http.Request) {

	unfinishedGameGroups := getUnfinishedGamesForUsers(usernames)

	// Finally, get HTML page to display the selectGames
	htmlBytes, err := getIndexHTMLBytes(unfinishedGameGroups)
	if err != nil {
		http.Error(w, fmt.Sprintf("There was an error processing your request: %s", err), http.StatusInternalServerError)
		return
	}

	// Write HTML page back to caller
	w.Write(htmlBytes)
}

func getFaviconHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(faviconFile)
}
