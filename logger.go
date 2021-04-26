package main

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type responseObserver struct {
	http.ResponseWriter
	status      int
	written     int64
	wroteHeader bool
}

func (o *responseObserver) Write(p []byte) (n int, err error) {
	if !o.wroteHeader {
		o.WriteHeader(http.StatusOK)
	}
	n, err = o.ResponseWriter.Write(p)
	o.written += int64(n)
	return
}

func (o *responseObserver) WriteHeader(code int) {
	o.ResponseWriter.WriteHeader(code)
	if o.wroteHeader {
		return
	}
	o.wroteHeader = true
	o.status = code
}

func logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requestUUID, _ := uuid.NewRandom()
		requestID := requestUUID.String()

		fields := logrus.Fields{
			"requestID": requestID,
			"handler":   name,
		}

		logrus.WithFields(fields).Info("request received")

		// wrap the HTTP handler with a start stop timer to get the duration
		o := &responseObserver{ResponseWriter: w}
		start := time.Now()
		inner.ServeHTTP(o, r)
		duration := time.Since(start)

		logrus.WithFields(fields).WithFields(logrus.Fields{
			"duration": duration,
			"status":   o.status,
		}).Info("request completed")

	})
}
