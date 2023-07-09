package main

import (
	"log"

	"github.com/sanbornm/go-selfupdate/selfupdate"
)

// The purpose of this app is to provide a simple example that just prints
// its version and updates to the latest version from example-server
// on localhost:8080.

// the app's version. This will be set on build.
var version string

// go-selfupdate setup and config
var updater = &selfupdate.Updater{
	CurrentVersion: version,                  // Manually update the const, or set it using `go build -ldflags="-X main.VERSION=<newver>" -o hello-updater src/hello-updater/main.go`
	ApiURL:         "http://localhost:8080/", // The server hosting `$CmdName/$GOOS-$ARCH.json` which contains the checksum for the binary
	BinURL:         "http://localhost:8080/", // The server hosting the zip file containing the binary application which is a fallback for the patch method
	DiffURL:        "http://localhost:8080/", // The server hosting the binary patch diff for incremental updates
	Dir:            "update/",                // The directory created by the app when run which stores the cktime file
	CmdName:        "hello-updater",          // The app name which is appended to the ApiURL to look for an update
	ForceCheck:     true,                     // For this example, always check for an update unless the version is "dev"
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// print the current version
	log.Printf("(hello-updater) Hello world! I am currently version: %q", updater.CurrentVersion)

	// try to update
	err := updater.BackgroundRun()
	if err != nil {
		log.Fatalln("Failed to update app:", err)
	}

	// print out latest version available
	log.Printf("(hello-updater) Latest version available: %q", updater.Info.Version)
}
