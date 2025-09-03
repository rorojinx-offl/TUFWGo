package tui

import (
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func NewModel() EnumModel {
	items, err := readUFWStatus()
	if err != nil && len(items) == 0 {
		lipgloss.NewStyle().
			Align(lipgloss.Center).
			Render("No UFW rules found/an error occurred.")
	}

	p := paginator.New()
	p.PerPage = 10
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	p.SetTotalPages(len(items))

	return EnumModel{
		paginator: p,
		items:     items,
	}
}

type EnumModel struct {
	items     []string
	paginator paginator.Model
}

func (m EnumModel) Init() tea.Cmd {
	return nil
}

func (m EnumModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "r":
			NewModel()
		}
	}
	m.paginator, cmd = m.paginator.Update(msg)
	return m, cmd
}

func (m EnumModel) View() string {
	var b strings.Builder
	if SSHActive {
		if err := sshCheckup(); err != nil {
			b.WriteString("\n  Active UFW Rules On Remote Client\n\n")
		}
		b.WriteString(fmt.Sprintf("\n  Active UFW Rules On Remote Client: %s\n\n", ssh.GlobalHost))
	} else {
		b.WriteString("\n  Active UFW Rules\n\n")
	}

	header := padRight("To", colToWidth) + padRight("Action", colActionWidth) + "From"
	b.WriteString(header + "\n")

	start, end := m.paginator.GetSliceBounds(len(m.items))
	for _, item := range m.items[start:end] {
		b.WriteString(item + "\n\n")
	}
	b.WriteString("  " + m.paginator.View())
	b.WriteString("\n\n  ←/→ page • r: reload • esc: back\n")
	return b.String()
}

type ufwRule struct {
	To     string
	Action string
	From   string
}

func readUFWStatus() ([]string, error) {
	var stdout string
	if SSHActive {
		if err := sshCheckup(); err != nil {
			return []string{"Could not retrieve rules from remote host."}, nil
		}
		stdout, _ = ssh.CommandStream("ufw status | grep -v \"(v6)\"")
	} else {
		stdout, _ = local.RunCommand("ufw status | grep -v \"(v6)\"")
	}
	rules := parseUFWStatus(stdout)
	if len(rules) == 0 {
		return []string{"No rules found."}, nil
	}

	items := make([]string, 0, len(rules))
	for _, r := range rules {
		items = append(items, formatRule(r))
	}
	return items, nil
}

func parseUFWStatus(stdout string) []ufwRule {
	sc := bufio.NewScanner(strings.NewReader(stdout))
	foundCols := false
	rules := []ufwRule{}

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		if !foundCols {
			if strings.HasPrefix(line, "To") && strings.Contains(line, "Action") && strings.Contains(line, "From") {
				foundCols = true
			}
			continue
		}

		if allDashes(line) {
			continue
		}

		fields := splitColumns(line)
		if len(fields) < 3 {
			continue
		}

		rules = append(rules, ufwRule{
			To:     fields[0],
			Action: fields[1],
			From:   strings.Join(fields[2:], " "),
		})
	}
	return rules
}

func allDashes(s string) bool {
	for _, r := range s {
		if r != '-' && r != '—' {
			return false
		}
	}
	return len(s) > 0
}

var colSplit = regexp.MustCompile(`\s{2,}`)

func splitColumns(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return colSplit.Split(s, -1)
}

const (
	colToWidth     = 24
	colActionWidth = 12
	colFromWidth   = 24
)

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func formatRule(r ufwRule) string {
	return " " + padRight(r.To, colToWidth) + padRight(r.Action, colActionWidth) + r.From
}
