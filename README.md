go-selfupdate
=============

Enable your Golang applications to self update.  Inspired by Chrome based on Heroku's [hk](https://github.com/heroku/hk).

## Features

* Tested on Mac, Linux, Arm, and Windows
* Creates binary diffs with bsdiff allowing small incremental updates
* Falls back to full binary update if diff fails to match SHA

## QuickStart

### Enable your App to Self Update

	var updater = &selfupdate.Updater{
		CurrentVersion: version,
		ApiURL:         "http://updates.yourdomain.com/",
		BinURL:         "http://updates.yourdownmain.com/",
		DiffURL:        "http://updates.yourdomain.com/",
		Dir:            "update/",
		CmdName:        "myapp", // app name
	}

	if updater != nil {
		go updater.BackgroundRun()
	}

### Push Out and Update

    go-selfupdate myapp 1.2

This will create a folder in your project called, *public* you can then rsync or transfer this to your webserver or S3.
