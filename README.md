# prism

[![Latest Release](https://img.shields.io/github/release/muesli/prism.svg)](https://github.com/muesli/prism/releases)
[![Build Status](https://github.com/muesli/prism/workflows/build/badge.svg)](https://github.com/muesli/prism/actions)
[![Go ReportCard](http://goreportcard.com/badge/muesli/prism)](http://goreportcard.com/report/muesli/prism)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/muesli/prism)

An RTMP stream recaster / splitter

## Usage

Calling prism with one or multiple RTMP URLs will listen for an incoming RTMP
connection, which will then get re-cast to all given URLs:

    prism URL [URL...]

If you want prism to listen on a different port than the default 1935, call it
with the `--bind` flag:

    prism --bind :1337 ...

By default prism will generate a key to protect the rtmp endpoint.  If you want to prevent it from being generated you can use flag:

    prism --no-key

If you would like to provide your own key to protect the rtmp endpoint you can use flag:

    prism --key="MySuperAwesomeKey"

When you start prism you will see something like:

    RTMP stream key set to: f977bc5a-21f6-44e7-869e-00807430480b
    Starting RTMP server...
    Waiting for incoming connection...

To begin broadcasting point OBS or your tool of choice to:

    rtmp://your-host:1935/live/publish/f977bc5a-21f6-44e7-869e-00807430480b

Replace host and key with your own. Actual path can be what ever you want so long as key is at the end.
