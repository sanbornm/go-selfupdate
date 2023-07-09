package main

import (
	"flag"
	"log"
	"net/http"
)

var servePath = flag.String("dir", "./public", "path to serve")

type logHandler struct {
	handler http.Handler
}

func (lh *logHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	log.Printf("(example-server) received request %s\n", r.URL.RequestURI())
	lh.handler.ServeHTTP(rw, r)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	// Simple static webserver with logging:
	log.Printf("Starting HTTP server on :8080 serving path %q Ctrl + C to close and quit", *servePath)
	log.Fatal(http.ListenAndServe(":8080", &logHandler{
		handler: http.FileServer(http.Dir(*servePath))},
	))
}
