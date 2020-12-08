package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
)

var (
	bind = flag.String("bind", ":1935", "bind address")

	log Logger
)

type App interface {
	Run() error
}

type Text struct {
}

func (t *Text) Run() error {
	select {}
}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Println("usage: prism URL [URL...]")
		os.Exit(1)
	}

	var a App
	if isatty.IsTerminal(os.Stdout.Fd()) {
		a = NewTUI()
		log = a.(*TUI)
	} else {
		a = &Text{}
		log = TextLogger{}
	}

	log.Log(0, "prism v0.2.0")
	log.Log(0, "Starting RTMP server...")

	setupRTMP()

	if err := a.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
