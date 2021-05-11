package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func main() {

	router := mux.NewRouter().StrictSlash(true)

	for _, r := range routes {

		handler := logger(r.handlerFunc, r.name)

		router.
			Methods(r.method).
			Path(r.pattern).
			Name(r.name).
			Handler(handler)
	}

	router.PathPrefix("/js/").Handler(websiteFolderAssetHandler())
	router.PathPrefix("/img/").Handler(websiteFolderAssetHandler())
	router.PathPrefix("/css/").Handler(websiteFolderAssetHandler())

	port := ":8889"
	server := &http.Server{
		Addr: port,
		// Good practice to set timeouts to avoid Slowloris attacks.
		// WriteTimeout: time.Second * 15,
		// ReadTimeout:  time.Second * 15,
		// IdleTimeout:  time.Second * 60,

		// Pass our instance of gorilla/mux in
		Handler: router,
	}

	go func() {
		logrus.Infof("server started on port %s", port)
		logrus.WithError(server.ListenAndServe()).Fatal("failed to start server")
	}()

	// graceful shutdown when termination signals received
	sigquit := make(chan os.Signal, 1)
	signal.Notify(sigquit, os.Interrupt, syscall.SIGTERM)

	sig := <-sigquit
	logrus.WithField("signal", sig).Info("caught interrupt signal, gracefully shutting down server")

	// shutdown the API server, waiting for any outstanding requests to complete
	server.Shutdown(context.Background())
	logrus.Info("graceful server shutdown complete, exiting")
	os.Exit(0)
}
