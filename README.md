# go-selfupdate

[![GoDoc](https://godoc.org/github.com/sanbornm/go-selfupdate/selfupdate?status.svg)](https://godoc.org/github.com/sanbornm/go-selfupdate/selfupdate)
![CI/CD](https://github.com/sanbornm/go-selfupdate/actions/workflows/ci.yml/badge.svg)

Enable your Golang applications to self update.  Inspired by Chrome based on Heroku's [hk](https://github.com/heroku/hk).

## Features

* Tested on Mac, Linux, Arm, and Windows
* Creates binary diffs with [bsdiff](http://www.daemonology.net/bsdiff/) allowing small incremental updates
* Falls back to full binary update if diff fails to match SHA

## QuickStart

### Install library and update/patch creation utility

`go install github.com/sanbornm/go-selfupdate/cmd/go-selfupdate@latest`

### Enable your App to Self Update

`go get -u github.com/sanbornm/go-selfupdate/...`

	var updater = &selfupdate.Updater{
		CurrentVersion: version, // the current version of your app used to determine if an update is necessary
		// these endpoints can be the same if everything is hosted in the same place
		ApiURL:         "http://updates.yourdomain.com/", // endpoint to get update manifest
		BinURL:         "http://updates.yourdomain.com/", // endpoint to get full binaries
		DiffURL:        "http://updates.yourdomain.com/", // endpoint to get binary diff/patches
		Dir:            "update/",                        // directory relative to your app to store temporary state files related to go-selfupdate
		CmdName:        "myapp",                          // your app's name (must correspond to app name hosting the updates)
		// app name allows you to serve updates for multiple apps on the same server/endpoint
	}

    // go look for an update when your app starts up
	go updater.BackgroundRun()
	// your app continues to run...

### Push Out and Update

	go-selfupdate path-to-your-app the-version
    go-selfupdate myapp 1.2

By default this will create a folder in your project called *public*. You can then rsync or transfer this to your webserver or S3. To change the output directory use `-o` flag.

If you are cross compiling you can specify a directory:

    go-selfupdate /tmp/mybinares/ 1.2

The directory should contain files with the name, $GOOS-$ARCH. Example:

    windows-386
    darwin-amd64
    linux-arm

If you are using [goxc](https://github.com/laher/goxc) you can output the files with this naming format by specifying this config:

    "OutPath": "{{.Dest}}{{.PS}}{{.Version}}{{.PS}}{{.Os}}-{{.Arch}}",

## Update Protocol

Updates are fetched from an HTTP(s) server. AWS S3 or static hosting can be used. A JSON manifest file is pulled first which points to the wanted version (usually latest) and matching metadata. SHA256 hash is currently the only metadata but new fields may be added here like signatures. `go-selfupdate` isn't aware of any versioning schemes. It doesn't know major/minor versions. It just knows the target version by name and can apply diffs based on current version and version you wish to move to. For example 1.0 to 5.0 or 1.0 to 1.1. You don't even need to use point numbers. You can use hashes, dates, etc for versions.

	GET yourserver.com/appname/linux-amd64.json

	200 ok
	{
		"Version": "2",
		"Sha256": "..." // base64
	}

	then

	GET patches.yourserver.com/appname/1.1/1.2/linux-amd64

	200 ok
	[bsdiff data]

	or

	GET fullbins.yourserver.com/appname/1.0/linux-amd64.gz

	200 ok
	[gzipped executable data]

The only required files are `<appname>/<os>-<arch>.json` and `<appname>/<latest>/<os>-<arch>.gz` everything else is optional. If you wanted to you could skip using go-selfupdate CLI tool and generate these two files manually or with another tool.

## Config

Updater Config options:

	type Updater struct {
		CurrentVersion string    // Currently running version. `dev` is a special version here and will cause the updater to never update.
		ApiURL         string    // Base URL for API requests (JSON files).
		CmdName        string    // Command name is appended to the ApiURL like http://apiurl/CmdName/. This represents one binary.
		BinURL         string    // Base URL for full binary downloads.
		DiffURL        string    // Base URL for diff downloads.
		Dir            string    // Directory to store selfupdate state.
		ForceCheck     bool      // Check for update regardless of cktime timestamp
		CheckTime      int       // Time in hours before next check
		RandomizeTime  int       // Time in hours to randomize with CheckTime
		Requester      Requester // Optional parameter to override existing HTTP request handler
		Info           struct {
			Version string
			Sha256  []byte
		}
		OnSuccessfulUpdate func() // Optional function to run after an update has successfully taken place
	}

### Restart on update

It is common for an app to want to restart to apply the update. `go-selfupdate` gives you a hook to do that but leaves it up to you on how and when to restart as it differs for all apps. If you have a service restart application like Docker or systemd you can simply exit and let the upstream app start/restart your application. Just set the `OnSuccessfulUpdate` hook:

	u.OnSuccessfulUpdate = func() { os.Exit(0) }

Or maybe you have a fancy graceful restart library/func:

	u.OnSuccessfulUpdate = func() { gracefullyRestartMyApp() }

## State

go-selfupdate will keep a Go time.Time formatted timestamp in a file named `cktime` in folder specified by `Updater.Dir`. This can be useful for debugging to see when the next update can be applied or allow other applications to manipulate it.