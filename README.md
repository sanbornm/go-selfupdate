# go-selfupdate

[![GoDoc](https://godoc.org/github.com/sanbornm/go-selfupdate/selfupdate?status.svg)](https://godoc.org/github.com/sanbornm/go-selfupdate/selfupdate)
[![Build Status](https://travis-ci.org/sanbornm/go-selfupdate.svg?branch=master)](https://travis-ci.org/sanbornm/go-selfupdate)

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
		Dir:            "update/",                        // directory to store temporary state files related to go-selfupdate
		CmdName:        "myapp",                          // your app's name (must correspond to app name hosting the updates)
		// app name allows you to serve updates for multiple apps on the same server/endpoint
	}

    // go look for an update when your app starts up
	if updater != nil {
		go updater.BackgroundRun()
	}
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