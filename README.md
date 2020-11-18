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
