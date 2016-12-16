package main

import (
	"log"
	"github.com/sanbornm/go-selfupdate/selfupdate"
)

var version string

var updater = &selfupdate.Updater{
    CurrentVersion: version,                   // Manually update the const, or set it using `go build -ldflags="-X main.VERSION=<newver>" -o hello-updater src/hello-updater/main.go`
    ApiURL:         "http://localhost:8080/",  // The server hosting `$CmdName/$GOOS-$ARCH.json` which contains the checksum for the binary
    BinURL:         "http://localhost:8080/",  // The server hosting the zip file containing the binary application which is a fallback for the patch method
    DiffURL:        "http://localhost:8080/",  // The server hosting the binary patch diff for incremental updates
    Dir:            "update/",                 // The directory created by the app when run which stores the cktime file
    CmdName:        "hello-updater",           // The app name which is appended to the ApiURL to look for an update
    ForceCheck: true,                          // For this example, always check for an update unless the version is "dev"
}

func main() {
	log.Printf("Hello world I am currently version %v", updater.CurrentVersion)
	updater.BackgroundRun()
	log.Printf("Next run, I should be %v", updater.Info.Version)
}
