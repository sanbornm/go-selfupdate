package main

import (
    "log"
    "net/http"
)

type logHandler struct {
    handler http.Handler
}

func (lh *logHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
    log.Printf("\n\texample-server received request %s\n", r.URL.RequestURI())
    lh.handler.ServeHTTP(rw,r)
}

func main() {
    // Simple static webserver with logging:
    log.Fatal(http.ListenAndServe(":8080", &logHandler{handler:http.FileServer(http.Dir("./public"))}))
}
