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
		name:        "getHomepage",
		method:      "GET",
		pattern:     "/",
		handlerFunc: getHomepage,
	},

	{
		name:        "getFaviconHandler",
		method:      "GET",
		pattern:     "/favicon.ico",
		handlerFunc: getFaviconHandler,
	},
}
