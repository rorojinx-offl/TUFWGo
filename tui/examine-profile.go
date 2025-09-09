package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"path/filepath"
	"strings"
)

type examineSelectModel struct {
	title    string
	dd       dropdown
	files    []string
	baseDir  string
	onChoose func(path string) tea.Msg
	err      string
}

type examineChosen struct{ Path string }
type examineEmpty struct{} // file exists but not sealed (no data)

func NewExamineSelect(baseDir string, onChoose func(path string) tea.Msg) *examineSelectModel {
	files := listJSONProfiles(filepath.Join(baseDir, "tufwgo", "profiles")) // same lister you have
	dd := newDropdown("Choose a profile to examine", files)
	return &examineSelectModel{
		title:    "Examine Profile",
		dd:       dd,
		files:    files,
		baseDir:  baseDir,
		onChoose: onChoose,
	}
}

func (m *examineSelectModel) Init() tea.Cmd { return nil }

func (m *examineSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "esc", "q":
			return m, tea.Quit
		case "enter":
			if !m.dd.Open {
				m.dd.Open = true
				return m, nil
			}
			name := strings.TrimSpace(m.dd.Value())
			if name == "" {
				m.err = "No profile selected."
				return m, nil
			}
			path := filepath.Join(m.baseDir, "tufwgo", "profiles", name)
			sealed, err := examineProfileHasData(path)
			if err != nil {
				m.err = fmt.Sprintf("Failed to open profile: %v", err)
				return m, nil
			}
			if !sealed {
				return newErrorBoxModel("Profile is empty",
					"This profile has no stored commands/rules yet.", m), nil
			}
			if m.onChoose != nil {
				return m, func() tea.Msg { return m.onChoose(path) }
			}
			return m, nil
		case "up", "down":
			if !m.dd.Open {
				m.dd.Open = true
			}
		}
	}
	m.dd.Focused = true
	m.dd.Update(msg)
	return m, nil
}

func (m *examineSelectModel) View() string {
	var b strings.Builder
	b.WriteString(focusStyle.Render(m.title) + "\n")
	b.WriteString(hintStyle.Render("Enter: open/confirm • ↑/↓: choose • Esc: back") + "\n")
	b.WriteString(sepStyle.Render(strings.Repeat("─", 80)) + "\n\n")
	b.WriteString(m.dd.View())
	if m.err != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(errorColor).Render(m.err))
	}
	return b.String()
}

type examineProbe struct {
	Commands []string     `json:"commands"`
	Rules    []ruleFormat `json:"rules"`
}

func examineProfileHasData(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return false, nil
	}
	var p examineProbe
	if err := json.Unmarshal(b, &p); err != nil {
		// invalid JSON -> treat as empty for examine
		return false, nil
	}
	return len(p.Commands) > 0 || len(p.Rules) > 0, nil
}

// ---------- Examine paginator ----------

type examineModel struct {
	name      string
	createdAt string
	commands  []string
	rules     []ruleFormat
	p         paginator.Model
}

func NewExamineModel(path string) (*examineModel, error) {
	rs, err := loadRuleSet(path)
	if err != nil {
		return nil, err
	}
	p := paginator.New()
	p.PerPage = 1 // one command per page
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	total := len(rs.Commands)
	if total == 0 && len(rs.Rules) > 0 {
		total = len(rs.Rules)
	}
	if total == 0 {
		total = 1
	}
	p.SetTotalPages(total)
	return &examineModel{
		name:      rs.Name,
		createdAt: rs.CreatedAt,
		commands:  rs.Commands,
		rules:     rs.Rules,
		p:         p,
	}, nil
}

func (m *examineModel) Init() tea.Cmd { return nil }

func (m *examineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "q", "esc":
			return m, tea.Quit
		}
	}
	m.p, cmd = m.p.Update(msg)
	return m, cmd
}

func (m *examineModel) View() string {
	var b strings.Builder

	// Header (like your other screens)
	b.WriteString(focusStyle.Render("Profile: ") + m.name + "  " + hintStyle.Render("Created: "+m.createdAt) + "\n")
	b.WriteString(sepStyle.Render(strings.Repeat("─", 80)) + "\n\n")

	// Current page index
	i := m.p.Page
	// Safeguard index
	cmdStr := ""
	if i >= 0 && i < len(m.commands) {
		cmdStr = m.commands[i]
	}
	var ruleStr string
	if i >= 0 && i < len(m.rules) {
		ruleStr = prettyRule(m.rules[i])
	}

	// Content
	if cmdStr != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Command") + ":\n  " + cmdStr + "\n\n")
	}
	if ruleStr != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Rule") + ":\n" + ruleStr + "\n")
	}
	if cmdStr == "" && ruleStr == "" {
		b.WriteString("No data on this page.\n")
	}

	// Footer paginator
	b.WriteString("\n  " + m.p.View())
	b.WriteString("\n\n  ←/→ page • esc: back\n")
	return b.String()
}

// Pretty, multi-line rule view (plain English-ish)
func prettyRule(r ruleFormat) string {
	lines := []string{
		fmt.Sprintf("  • Action:    %s", nz(r.Action, "—")),
		fmt.Sprintf("  • Direction: %s", nz(r.Direction, "—")),
		fmt.Sprintf("  • Interface: %s", nz(r.Interface, "—")),
		fmt.Sprintf("  • From:      %s", nz(r.FromIP, "—")),
		fmt.Sprintf("  • To:        %s", nz(r.ToIP, "—")),
		fmt.Sprintf("  • Port:      %s", nz(r.Port, "—")),
		fmt.Sprintf("  • Protocol:  %s", nz(r.Protocol, "—")),
	}
	return strings.Join(lines, "\n")
}

func nz(s, alt string) string {
	if strings.TrimSpace(s) == "" {
		return alt
	}
	return s
}

// Read a profile file to RuleSet
func loadRuleSet(path string) (*RuleSet, error) {
	var rs RuleSet
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &rs); err != nil {
		return nil, err
	}
	return &rs, nil
}

// ---------- Flow wrapper (like your add-to-profile flow) ----------

type examineFlow struct {
	child tea.Model
}

func NewExamineFlow() *examineFlow {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return &examineFlow{child: newErrorBoxModel("Error", "Could not access user config dir.", nil)}
	}
	onChoose := func(path string) tea.Msg { return examineChosen{Path: path} }
	return &examineFlow{
		child: NewExamineSelect(cfg, onChoose),
	}
}

func (m *examineFlow) Init() tea.Cmd { return nil }

func (m *examineFlow) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.child != nil {
		switch v := msg.(type) {
		case examineChosen:
			em, err := NewExamineModel(v.Path)
			if err != nil {
				m.child = newErrorBoxModel("Failed to load profile", err.Error(), m.child)
				return m, nil
			}
			m.child = em
			return m, nil
		}
		next, cmd := m.child.Update(msg)
		m.child = next
		return m, cmd
	}
	return m, nil
}

func (m *examineFlow) View() string {
	if m.child != nil {
		return m.child.View()
	}
	return ""
}
