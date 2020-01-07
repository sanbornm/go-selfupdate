# go-selfupdate

[![GoDoc](https://godoc.org/github.com/sanbornm/go-selfupdate/selfupdate?status.svg)](https://godoc.org/github.com/sanbornm/go-selfupdate/selfupdate)
[![Build Status](https://travis-ci.org/sanbornm/go-selfupdate.svg?branch=master)](https://travis-ci.org/sanbornm/go-selfupdate)

Enable your Golang applications to self update.  Inspired by Chrome based on Heroku's [hk](https://github.com/heroku/hk).

Requires Golang 1.8 or higher.

## Features

* Tested on Mac, Linux, Arm, and Windows
* Creates binary diffs with [bsdiff](http://www.daemonology.net/bsdiff/) allowing small incremental updates (which means you need to get bzip2 installed on your machine for this package to work)
* Falls back to full binary update if diff fails to match SHA

## QuickStart

### Install library and update/patch creation utility

`go get -u github.com/EliCDavis/go-selfupdate/...`

### Enable your App to Self Update

```golang
updater := selfupdate.NewUpdater(version, "http://updates.yourdomain.com/", "myapp")

// run an update in the background
go func () {
    updated, err := updater.Run()
    if err != nil {
        log.Printf("Error updating: %s", err.Error())
    } else {
        log.Printf("Update applied: %t", updated)
    }
}
```

If you prefer to instead of keeping this app updated, but a different file, just specify the path:

```golang
updater := selfupdate.NewUpdater(version, "http://updates.yourdomain.com/", "myapp").
    SetUpdatableResolver(NewSpecificFileUpdatableResolver("path/to/your/file"))

// run an update in the background
go func () {
    updated, err := updater.Run()
    if err != nil {
        log.Printf("Error updating: %s", err.Error())
    } else {
        log.Printf("Update applied: %t", updated)
    }
}
```

### Push Out and Update

```bash
go-selfupdate myapp 1.2
```

This will create a folder in your project called, *public* you can then rsync or transfer this to your webserver or S3.

If you are cross compiling you can specify a directory:

```bash
go-selfupdate /tmp/mybinares/ 1.2
```

The directory should contain files with the name, $GOOS-$ARCH. Example:

```text
windows-386
darwin-amd64
linux-arm
```

If you are using [goxc](https://github.com/laher/goxc) you can output the files with this naming format by specifying this config:

```text
"OutPath": "{{.Dest}}{{.PS}}{{.Version}}{{.PS}}{{.Os}}-{{.Arch}}",
```

## Development

Uses mockgen to generate mocks for our interfaces. If you make any changes to any of the interface signatures, you'll need to update the mocks using the command:

```bash
go generate ./...
```
