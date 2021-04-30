package main

import "net/http"

type route struct {
	name        string
	method      string
	pattern     string
	handlerFunc http.HandlerFunc
}

var routes = []route{
	{
		name:        "healthCheckHandler",
		method:      "GET",
		pattern:     "/health",
		handlerFunc: healthCheckHandler,
	},

	{
		name:        "getHomepage",
		method:      "GET",
		pattern:     "/",
		handlerFunc: getHomepage,
	},

	{
		name:        "getGamesForMonthHTML",
		method:      "GET",
		pattern:     "/monthgames",
		handlerFunc: getGamesForMonthHTML,
	},

	{
		name:        "getFaviconHandler",
		method:      "GET",
		pattern:     "/favicon.ico",
		handlerFunc: getFaviconHandler,
	},
}
