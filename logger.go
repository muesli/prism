package main

import "fmt"

type Logger interface {
	Log(status int, message string)
	Replace(status int, message string)
}

type Log struct {
	Status  int
	Message string
}

type TextLogger struct {
}

func (t TextLogger) Log(status int, message string) {
	fmt.Println(message)
}

func (t TextLogger) Replace(status int, message string) {
	t.Log(status, message)
}
