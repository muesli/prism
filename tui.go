package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

var color = termenv.ColorProfile().Color

type logMsg struct {
	status  int
	message string
	replace bool
}

type errMsg error

type TUI struct {
	logs     []Log
	activity chan logMsg
	spinner  spinner.Model

	err error
}

func NewTUI() *TUI {
	s := spinner.NewModel()
	s.Spinner = spinner.Globe

	return &TUI{
		spinner:  s,
		activity: make(chan logMsg),
	}
}

func (t TUI) Run() error {
	return tea.NewProgram(t).Start()
}

func (t TUI) Log(status int, message string) {
	go func() {
		t.activity <- logMsg{status, message, false}
	}()
}

func (t TUI) Replace(status int, message string) {
	go func() {
		t.activity <- logMsg{status, message, true}
	}()
}

func (t TUI) Init() tea.Cmd {
	return tea.Batch(t.waitForLogging, spinner.Tick)
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			fallthrough
		case "esc":
			fallthrough
		case "ctrl+c":
			return t, tea.Quit
		default:
			return t, nil
		}

	case errMsg:
		t.err = msg
		return t, nil

	case logMsg:
		if msg.replace {
			l := len(t.logs)
			if l > 0 && t.logs[l-1].Status == msg.status {
				t.logs = append(t.logs[:l-1], t.logs[l:]...)
			}
		}
		t.logs = append(t.logs, Log{msg.status, msg.message})
		return t, t.waitForLogging

	default:
		var cmd tea.Cmd
		t.spinner, cmd = t.spinner.Update(msg)
		return t, cmd
	}
}

func (t TUI) View() string {
	if t.err != nil {
		return t.err.Error()
	}

	var s string
	for _, l := range t.logs {
		var status termenv.Style
		switch l.Status {
		case 0:
			status = termenv.String("✔ ").Foreground(color("40"))
		case 1:
			status = termenv.String("❌").Foreground(color("196"))
		case 2:
			status = termenv.String("⏪").Foreground(color("33"))
		case 3:
			status = termenv.String("⏩").Foreground(color("165"))
		case 4:
			status = termenv.String(t.spinner.View()).Foreground(color("115"))
		default:
			panic("unknown status")
		}

		s += " " + status.String() + " " + l.Message + "\n"
	}

	return "\n" + s + "\n"
}

func (t TUI) waitForLogging() tea.Msg {
	return <-t.activity
}
