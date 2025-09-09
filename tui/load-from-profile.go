package tui

import (
	"TUFWGo/system/local"
	"encoding/json"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"path/filepath"
	"strings"
)

type profileLoadModel struct {
	title  string
	dd     dropdown
	files  []string
	width  int
	height int
	err    string
}
type LoadProfile struct{ Path string }
type EmptyProfile struct{}
type ExecuteProfile struct{ RawCommands []string }

func LoadFromProfile() *profileLoadModel {
	files := listJSONProfiles(filepath.Join(baseDir, "tufwgo/profiles"))
	dd := newDropdown("Choose a ruleset", files)
	return &profileLoadModel{
		title: "Select Ruleset",
		dd:    dd,
		files: files,
	}
}

func (m *profileLoadModel) Init() tea.Cmd { return nil }
func (m *profileLoadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.KeyMsg:
		switch v.String() {
		case "esc", "q":
			return m, tea.Quit
		case "enter":
			if !m.dd.Open {
				m.dd.Open = true
				return m, nil
			}

			chosen := strings.TrimSpace(m.dd.Value())
			if chosen == "" {
				m.err = "No ruleset selected."
				return m, nil
			}

			path := filepath.Join(baseDir+"/tufwgo/profiles", chosen)
			sealed, err := profileIsSealed(path)
			if err != nil {
				m.err = fmt.Sprintf("Failed to check profile: %v", err)
				return m, nil
			}
			if !sealed {
				return m, func() tea.Msg { return EmptyProfile{} }
			}

			return m, func() tea.Msg { return LoadProfile{Path: path} }
		case "up", "down":
			if !m.dd.Open {
				m.dd.Open = true
			}
		}
	}
	m.dd.Update(msg) // arrow keys etc.
	return m, nil
}
func (m *profileLoadModel) View() string {
	var b strings.Builder
	b.WriteString(focusStyle.Render(m.title) + "\n")
	b.WriteString(hintStyle.Render("Enter to open/confirm • ↑/↓ to choose • Esc to close") + "\n")
	b.WriteString(sepStyle.Render(strings.Repeat("─", 80)) + "\n\n")
	b.WriteString(m.dd.View())
	if m.err != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(errorColor).Render(m.err))
	}
	return b.String()
}

func showRulesFromProfile(path string) (string, string, string, []string, error) {
	fullPath := filepath.Join(baseDir, "tufwgo/profiles", path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", "", "", nil, err
	}

	var rs RuleSet
	if err = json.Unmarshal(data, &rs); err != nil {
		return "", "", "", nil, err
	}

	rawCommands := rs.Commands
	commands := strings.Join(rs.Commands, "\n")
	return rs.Name, rs.CreatedAt, commands, rawCommands, nil
}

func executeProfile(commands []string) error {
	for _, cmd := range commands {
		_, err := local.RunCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to execute command %q: %w", cmd, err)
		}
	}
	return nil
}
