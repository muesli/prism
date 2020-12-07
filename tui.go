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

type model struct {
	app     *TUI
	logs    []Log
	spinner spinner.Model

	quitting bool
	err      error
}

func initialModel(a *TUI) model {
	s := spinner.NewModel()
	s.Spinner = spinner.Globe

	return model{
		app:     a,
		spinner: s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.awaitLog, spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			fallthrough
		case "esc":
			fallthrough
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	case errMsg:
		m.err = msg
		return m, nil

	case logMsg:
		if msg.replace {
			m.replace(msg.status, msg.message)
		} else {
			m.log(msg.status, msg.message)
		}
		return m, m.awaitLog

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	var s string
	for _, l := range m.logs {
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
			status = termenv.String(m.spinner.View()).Foreground(color("115"))
		default:
			panic("unknown status")
		}

		s += " " + status.String() + " " + l.Message + "\n"
	}

	return "\n" + s + "\n"
}

func (m *model) log(status int, message string) {
	m.logs = append(m.logs, Log{status, message})
}

func (m *model) replace(status int, message string) {
	l := len(m.logs)
	if l > 0 && m.logs[l-1].Status == status {
		m.logs = append(m.logs[:l-1], m.logs[l:]...)
	}

	m.logs = append(m.logs, Log{status, message})
}

func (m model) awaitLog() tea.Msg {
	return <-m.app.logs
}

type TUI struct {
	logs chan logMsg
}

func NewTUI() *TUI {
	return &TUI{make(chan logMsg)}
}

func (t *TUI) Run() error {
	p := tea.NewProgram(initialModel(t))
	return p.Start()
}

func (t *TUI) Log(status int, message string) {
	go t.log(status, message, false)
}

func (t *TUI) Replace(status int, message string) {
	go t.log(status, message, true)
}

func (t *TUI) log(status int, message string, replace bool) {
	t.logs <- logMsg{status, message, replace}
}
