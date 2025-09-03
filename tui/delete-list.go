package tui

import (
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"bufio"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DelListModel struct {
	paginator paginator.Model
	items     []string
	delete    textinput.Model
}

var fieldBoxStyle = lipgloss.NewStyle().
	Padding(1, 2).
	Border(lipgloss.RoundedBorder()).
	Copy().BorderForeground(lipgloss.Color("#ffffff"))

type DeleteConfirmation struct {
	number int
	error  error
}

type DeleteExecuted struct{}

func (d DelListModel) collectNumber() (int, error) {
	str := d.delete.Value()
	num, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errors.New("invalid number")
	}
	return int(num), nil
}

func DeleteList() DelListModel {
	items, err := readUFWStatusForDeletion()
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

	d := DelListModel{}
	d.paginator = p
	d.items = items
	d.delete = textinput.New()
	d.delete.Placeholder = "rule number e.g. 1"
	d.delete.Prompt = ""
	d.delete.CharLimit = 4
	d.delete.Width = 20
	d.delete.Focus()
	return d
}

func (d DelListModel) Init() tea.Cmd { return textinput.Blink }

func (d DelListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return d, tea.Quit
		case "r":
			DeleteList()
		case "enter":
			num, err := d.collectNumber()
			return d, func() tea.Msg { return DeleteConfirmation{number: num, error: err} }
		}
	}
	d.paginator, cmd = d.paginator.Update(msg)
	d.delete, cmd = d.delete.Update(msg)
	return d, cmd
}

func (d DelListModel) View() string {
	var b strings.Builder
	if SSHActive {
		if err := sshCheckup(); err != nil {
			b.WriteString("\n  Delete UFW Rules On Remote Client\n\n")
		}
		b.WriteString(fmt.Sprintf("\n  Delete UFW Rules On Remote Client: %s\n\n", ssh.GlobalHost))
	} else {
		b.WriteString("\n  Delete UFW Rules\n\n")
	}

	header := padRight("#", colNumberWidth) + padRight("To", colToWidth) + padRight("Action", colActionWidth) + "From"
	b.WriteString(header + "\n")

	start, end := d.paginator.GetSliceBounds(len(d.items))
	for _, item := range d.items[start:end] {
		b.WriteString(item + "\n\n")
	}
	b.WriteString("  " + d.paginator.View() + "\n\n")

	deleteField := renderDeleteField("Enter a rule number to delete it", d.delete.View())
	b.WriteString(deleteField + "\n")

	b.WriteString("\n\n  ←/→ page • r: reload • esc: back\n")
	return b.String()
}

func renderDeleteField(label, body string) string {
	return fieldBoxStyle.Render(labelStyle.Foreground(accent).Render(label) + "\n" + body)
}

type ufwRuleWithNumbering struct {
	To     string
	Action string
	From   string
	Number string
}

func readUFWStatusForDeletion() ([]string, error) {
	var stdout string
	if SSHActive {
		if err := sshCheckup(); err != nil {
			return []string{"Could not retrieve rules from remote host."}, nil
		}
		stdout, _ = ssh.CommandStream("ufw status numbered | grep -v \"(v6)\"")
	} else {
		stdout, _ = local.RunCommand("ufw status numbered | grep -v \"(v6)\"")
	}

	rules := parseUFWStatusForDeletion(stdout)
	if len(rules) == 0 {
		return []string{"No rules found."}, nil
	}

	items := make([]string, 0, len(rules))
	for _, r := range rules {
		items = append(items, formatRuleForDeletion(r))
	}
	return items, nil
}

var leadingNum = regexp.MustCompile(`^\[\s*(\d+)\]\s+`)

func parseUFWStatusForDeletion(stdout string) []ufwRuleWithNumbering {
	sc := bufio.NewScanner(strings.NewReader(stdout))
	foundCols := false
	rules := []ufwRuleWithNumbering{}

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

		m := leadingNum.FindStringSubmatch(line)
		if len(m) != 2 {
			continue
		}
		num := m[1]
		rest := strings.TrimSpace(line[len(m[0]):])

		fields := splitColumns(rest)
		if len(fields) < 3 {
			continue
		}

		rules = append(rules, ufwRuleWithNumbering{
			Number: num,
			To:     fields[0],
			Action: fields[1],
			From:   strings.Join(fields[2:], " "),
		})
	}
	return rules
}

const colNumberWidth = 6

func formatRuleForDeletion(r ufwRuleWithNumbering) string {
	return padRight(r.Number, colNumberWidth) + padRight(r.To, colToWidth) + padRight(r.Action, colActionWidth) + r.From
}
