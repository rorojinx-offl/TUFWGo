package tui

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Styling -----------------------------------------------------------------
var (
	accent     = lipgloss.Color("#7D56F4")
	muted      = lipgloss.Color("#888888")
	errorColor = lipgloss.Color("#FF5C5C")

	labelStyle   = lipgloss.NewStyle().Foreground(accent)
	hintStyle    = lipgloss.NewStyle().Foreground(muted)
	focusStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	boxStyle     = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder())
	focusBox     = boxStyle.Copy().BorderForeground(accent)
	dropdownItem = lipgloss.NewStyle()
	selectedItem = lipgloss.NewStyle().Bold(true).Underline(true)
	sepStyle     = lipgloss.NewStyle().Foreground(muted)
)

// --- Simple dropdown component -----------------------------------------------
// A compact, keyboard-only dropdown that mimics a select box.
// Controls: Enter to open/confirm, ↑/↓ to navigate when open, Esc to close.
// Tab / Shift+Tab to move focus handled by parent model.

type dropdown struct {
	Label    string
	Options  []string
	Selected int
	Open     bool
	Focused  bool
	Width    int
	Err      string
}

func newDropdown(label string, options []string) dropdown {
	return dropdown{Label: label, Options: options, Selected: 0, Open: false, Width: 28}
}

func (d dropdown) Value() string {
	if len(d.Options) == 0 {
		return ""
	}
	if d.Selected < 0 || d.Selected >= len(d.Options) {
		return ""
	}
	return d.Options[d.Selected]
}

func (d dropdown) View() string {
	current := d.Value()
	header := labelStyle.Render(d.Label)
	box := boxStyle
	if d.Focused {
		box = focusBox
	}

	arrow := "▾"
	if d.Open {
		arrow = "▴"
	}

	line := fmt.Sprintf("%s %s", padRightForm(current, d.Width), arrow)

	var items []string
	if d.Open {
		for i, opt := range d.Options {
			style := dropdownItem
			if i == d.Selected {
				style = selectedItem
			}
			items = append(items, "  "+style.Render(opt))
		}
	}

	errLine := ""
	if d.Err != "" {
		errLine = "\n" + lipgloss.NewStyle().Foreground(errorColor).Render(d.Err)
	}

	content := header + "\n" + line
	if len(items) > 0 {
		content += "\n" + strings.Join(items, "\n")
	}
	return box.Render(content) + errLine
}

func (d *dropdown) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.Type {
		case tea.KeyEnter:
			// Toggle open if focused, otherwise no-op
			d.Open = !d.Open
			return nil
		case tea.KeyEsc:
			d.Open = false
			return nil
		case tea.KeyUp:
			if d.Open && len(d.Options) > 0 {
				d.Selected = (d.Selected - 1 + len(d.Options)) % len(d.Options)
			}
			return nil
		case tea.KeyDown:
			if d.Open && len(d.Options) > 0 {
				d.Selected = (d.Selected + 1) % len(d.Options)
			}
			return nil
		}
	}
	return nil
}

// --- Root form model ----------------------------------------------------------

type focusIndex int

const (
	fAction focusIndex = iota
	fDirection
	fInterface
	fFromIP
	fToIP
	fPort
	fProtocol
	fApp
	fSubmit
	fCount
)

type model struct {
	// Dropdowns
	action    dropdown
	direction dropdown
	iface     dropdown
	app       dropdown

	// Text inputs
	fromIP   textinput.Model
	toIP     textinput.Model
	port     textinput.Model
	protocol textinput.Model

	focused focusIndex
	width   int
	height  int
	err     string
}

func initialModel() model {
	// Developer-defined actions and directions
	actions := []string{"allow", "deny", "reject", "limit"}
	directions := []string{"default", "in", "out"}

	// Discover interfaces & app profiles from the system
	ifaces := []string{"default"}
	ifaces = append(ifaces, listInterfaces()...)
	apps := []string{"(none)"}
	apps = append(apps, listUFWApps()...)

	// Initialize model

	m := model{
		action:    newDropdown("Action", actions),
		direction: newDropdown("Direction", directions),
		iface:     newDropdown("Interface", ifaces),
		app:       newDropdown("App Profile", apps),
		focused:   fAction,
	}

	m.fromIP = textinput.New()
	m.fromIP.Placeholder = "e.g. 192.168.1.10 or 10.0.0.0/24"
	m.fromIP.Prompt = ""
	m.fromIP.CharLimit = 64
	m.fromIP.Width = 40

	m.toIP = textinput.New()
	m.toIP.Placeholder = "e.g. any or 203.0.113.5"
	m.toIP.Prompt = ""
	m.toIP.CharLimit = 64
	m.toIP.Width = 40

	m.port = textinput.New()
	m.port.Placeholder = "e.g. 22 or 80,443"
	m.port.Prompt = ""
	m.port.CharLimit = 32
	m.port.Width = 20

	m.protocol = textinput.New()
	m.protocol.Placeholder = "tcp | udp | any"
	m.protocol.Prompt = ""
	m.protocol.CharLimit = 10
	m.protocol.Width = 12

	m.updateFocus()
	return m
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "tab", "esc":
			if msg.String() == "tab" {
				m.focused = (m.focused + 1) % fCount
			} else {
				m.focused = (m.focused - 1 + fCount) % fCount
			}
			m.updateFocus()
			return m, nil
		case "enter":
			if m.focused == fSubmit {
				return m, tea.Quit // no-op submit for now
			}
		}
	}

	// Route key events to focused control
	switch m.focused {
	case fAction:
		m.action.Update(msg)
	case fDirection:
		m.direction.Update(msg)
	case fInterface:
		m.iface.Update(msg)
	case fApp:
		m.app.Update(msg)
	case fFromIP:
		var cmd tea.Cmd
		m.fromIP, cmd = m.fromIP.Update(msg)
		return m, cmd
	case fToIP:
		var cmd tea.Cmd
		m.toIP, cmd = m.toIP.Update(msg)
		return m, cmd
	case fPort:
		var cmd tea.Cmd
		m.port, cmd = m.port.Update(msg)
		return m, cmd
	case fProtocol:
		var cmd tea.Cmd
		m.protocol, cmd = m.protocol.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) updateFocus() {
	m.action.Focused = m.focused == fAction
	m.direction.Focused = m.focused == fDirection
	m.iface.Focused = m.focused == fInterface
	m.app.Focused = m.focused == fApp

	m.fromIP.Blur()
	m.toIP.Blur()
	m.port.Blur()
	m.protocol.Blur()
	switch m.focused {
	case fFromIP:
		m.fromIP.Focus()
	case fToIP:
		m.toIP.Focus()
	case fPort:
		m.port.Focus()
	case fProtocol:
		m.protocol.Focus()
	}
}

func (m model) View() string {
	// Layout: two columns if wide enough, otherwise single column
	cols := []string{
		m.action.View(),
		m.direction.View(),
		m.iface.View(),
		boxStyle.Render(labelStyle.Render("From IP") + "\n" + m.fromIP.View()),
		boxStyle.Render(labelStyle.Render("To IP") + "\n" + m.toIP.View()),
		boxStyle.Render(labelStyle.Render("Port") + "\n" + m.port.View()),
		boxStyle.Render(labelStyle.Render("Protocol") + "\n" + m.protocol.View()),
		m.app.View(),
	}

	var b strings.Builder
	b.WriteString(focusStyle.Render("UFW Rule Form") + "\n")
	b.WriteString(hintStyle.Render("Tab/Esc to move • Enter to open/close a dropdown • ↑/↓ to select • q to close • Enter on Submit to exit") + "\n")
	b.WriteString(sepStyle.Render(strings.Repeat("─", 80)) + "\n\n")

	// Grid
	left := strings.Join(cols[:4], "\n\n")
	right := strings.Join(cols[4:], "\n\n")

	row := lipgloss.JoinHorizontal(lipgloss.Top, left+"\n\n", right)
	b.WriteString(row)

	// Submit button
	btn := "[ Submit ]"
	if m.focused == fSubmit {
		btn = focusStyle.Render(btn)
	} else {
		btn = hintStyle.Render(btn)
	}
	b.WriteString("\n\n" + btn + "\n")

	return b.String()
}

// --- Helpers: system discovery ------------------------------------------------

func listInterfaces() []string {
	cmd := exec.Command("sh", "-c", "ip -o link show | awk -F': ' '{print $2}' | sed 's/@.*//' ")
	out, err := cmd.Output()
	if err != nil {
		return []string{"lo"}
	}
	var ifaces []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}
		// Skip loopback in options if you prefer; leave in for completeness
		ifaces = append(ifaces, name)
	}
	// Deduplicate & sort
	ifaces = uniqSorted(ifaces)
	if len(ifaces) == 0 {
		ifaces = []string{"default"}
	}
	return ifaces
}

func listUFWApps() []string {
	cmd := exec.Command("sh", "-c", "ufw app list 2>/dev/null")
	out, err := cmd.Output()
	if err != nil {
		return []string{"(none)"}
	}
	// Output format typically:
	// Available applications:
	//   OpenSSH
	//   CUPS
	//   ...
	lines := strings.Split(string(out), "\n")
	re := regexp.MustCompile(`^\s+(.+)`)
	var apps []string
	for _, ln := range lines {
		ln = strings.TrimRight(ln, "\r")
		m := re.FindStringSubmatch(ln)
		if len(m) == 2 {
			apps = append(apps, strings.TrimSpace(m[1]))
		}
	}
	apps = uniqSorted(apps)
	if len(apps) == 0 {
		apps = []string{"(none)"}
	}
	return apps
}

func uniqSorted(xs []string) []string {
	m := map[string]struct{}{}
	for _, x := range xs {
		m[x] = struct{}{}
	}
	var out []string
	for x := range m {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func padRightForm(s string, w int) string {
	if len([]rune(s)) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len([]rune(s)))
}
