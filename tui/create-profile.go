package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProfileModel struct {
	ti       textinput.Model
	width    int
	height   int
	savedMsg string
	errMsg   string
	done     bool
}

type RuleSet struct {
	Name      string       `json:"name"`
	CreatedAt string       `json:"created_at"`
	Commands  []string     `json:"commands"`
	Rules     []ruleFormat `json:"rules"`
}

type ProfileDone struct{}
type ProfCreateAudit struct{ Err error }

func NewProfileModel() *ProfileModel {
	ti := textinput.New()
	ti.Placeholder = "Profile Name"
	ti.Width = 40
	ti.Focus()
	ti.CharLimit = 64
	ti.Prompt = "Set name: "
	ti.Cursor.Style = lipgloss.NewStyle().Bold(true)
	return &ProfileModel{ti: ti}
}

func (m *ProfileModel) Init() tea.Cmd { return textinput.Blink }

func (m *ProfileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.done = true
			return m, func() tea.Msg { return ProfileDone{} }
		case "enter":
			name := strings.TrimSpace(m.ti.Value())
			if name == "" {
				m.errMsg = "Profile name cannot be empty"
				return m, func() tea.Msg { return ProfCreateAudit{Err: errors.New(m.errMsg)} }
			}
			path, err := saveEmptyRuleSet(name)
			if err != nil {
				m.errMsg = fmt.Sprintf("Error saving profile: %v", err)
				return m, func() tea.Msg { return ProfCreateAudit{Err: err} }
			} else {
				m.savedMsg = fmt.Sprintf("Profile saved to %s", path)
				m.done = true
				return m, func() tea.Msg { return ProfCreateAudit{} }
			}
		}
	}
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

var (
	boxStyleProfile = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Margin(0).
			Width(48)
	titleStyle       = lipgloss.NewStyle().Bold(true).Underline(true)
	hintStyleProfile = lipgloss.NewStyle().Faint(true)
	errStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
	okStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#5fff87"))
)

func maxSize(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *ProfileModel) View() string {
	content := titleStyle.Render("Create Ruleset") + "\n\n" +
		m.ti.View() + "\n\n" +
		hintStyleProfile.Render("Press Enter to save • Esc to cancel")

	if m.errMsg != "" {
		content += "\n\n" + errStyle.Render(m.errMsg)
	}

	if m.savedMsg != "" {
		content += "\n\n" + okStyle.Render(m.savedMsg)
	}

	box := boxStyleProfile.Render(content)

	return lipgloss.Place(
		maxSize(60, m.width),
		maxSize(12, m.height),
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

func saveEmptyRuleSet(name string) (string, error) {
	slug := slugify(name)
	if slug == "" {
		return "", errors.New("name cannot be empty or contain only invalid characters")
	}
	base, err := userProfilesDir()
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(base, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(base, slug+".json")
	finalPath := path
	for i := 2; fileExists(finalPath); i++ {
		finalPath = filepath.Join(base, fmt.Sprintf("%s-%d.json", slug, i))
	}

	if err = os.WriteFile(finalPath, nil, 0o644); err != nil {
		return "", err
	}
	return finalPath, nil
}

func userProfilesDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "tufwgo", "profiles"), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var validChars = regexp.MustCompile(`[^a-zA-Z0-9\-]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	s = validChars.ReplaceAllString(s, "")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return s

}
