package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shadmeoli/kairos/internal/gitexec"
	"github.com/shadmeoli/kairos/internal/store"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type TimelineModel struct {
	RepoTop     string
	State       store.State
	CurrentHEAD string
	Width       int
	cursor      int
	quitting    bool
	jumpIndex   int
	jumpChosen  bool
	err         error
}

func NewTimelineModel(repoTop string, s store.State, head string) TimelineModel {
	mi := 0
	if s.Cursor >= 0 && s.Cursor < len(s.Checkpoints) {
		mi = s.Cursor
	}
	return TimelineModel{
		RepoTop:     repoTop,
		State:       s,
		CurrentHEAD: head,
		cursor:      mi,
		jumpIndex:   -1,
	}
}

func (m TimelineModel) Init() tea.Cmd {
	return nil
}

func (m TimelineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.State.Checkpoints)-1 {
				m.cursor++
			}
		case "enter":
			m.jumpIndex = m.cursor
			m.jumpChosen = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m TimelineModel) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("kairos timeline"))
	b.WriteString("\n\n")
	if len(m.State.Checkpoints) == 0 {
		b.WriteString(dimStyle.Render("no checkpoints — run: kairos save"))
		b.WriteString("\n")
		return b.String()
	}
	root := gitexec.RepoRoot(m.RepoTop)
	for i := len(m.State.Checkpoints) - 1; i >= 0; i-- {
		cp := m.State.Checkpoints[i]
		short, _ := gitexec.ShortHash(root, cp.HEAD)
		if short == "" {
			short = cp.HEAD[:min(7, len(cp.HEAD))]
		}
		ref := cp.Branch
		if cp.Detached || ref == "" {
			ref = "detached"
		}
		line := fmt.Sprintf("[%d] %-16s @ %s", i, ref, short)
		if cp.Label != "" {
			line += fmt.Sprintf("  (%s)", cp.Label)
		}
		prefix := "│ "
		if i == len(m.State.Checkpoints)-1 {
			prefix = "* "
		}
		suffix := ""
		if i == m.State.Cursor {
			suffix = dimStyle.Render("  ← timeline cursor")
		}
		if strings.HasPrefix(m.CurrentHEAD, cp.HEAD) || m.CurrentHEAD == cp.HEAD {
			suffix += dimStyle.Render("  [HEAD]")
		}
		if i == m.cursor {
			line = cursorStyle.Render(line)
		}
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString(suffix)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("j/k move · enter jump here · q quit"))
	b.WriteString("\n")
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ChosenJump returns index if user pressed enter, -1 otherwise.
func (m TimelineModel) ChosenJump() (int, bool) {
	return m.jumpIndex, m.jumpChosen
}
