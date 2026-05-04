package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// Config holds the configuration for the TUI.
type Config struct {
	Namespace   string
	KubeContext string
	TailLines   int64
	MaxEvents   int
}

type scanFinishedMsg struct {
	report *analyzer.NamespaceReport
	err    error
}

type treeNode struct {
	node     *analyzer.MapNode
	depth    int
	isLast   bool
	expanded bool
}

type model struct {
	config    Config
	loading   bool
	spinner   spinner.Model
	report    *analyzer.NamespaceReport
	err       error
	width     int
	height    int
	
	// Navigation
	cursor    int
	expanded  map[string]bool // key: Kind/Name
	treeList  []treeNode
}

func NewModel(cfg Config) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		config:   cfg,
		loading:  true,
		spinner:  s,
		expanded: make(map[string]bool),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.runScan,
	)
}

func (m model) runScan() tea.Msg {
	client, err := kube.NewClient(m.config.KubeContext)
	if err != nil {
		return scanFinishedMsg{err: err}
	}

	report, err := analyzer.CollectNamespaceReport(
		client,
		m.config.Namespace,
		m.config.TailLines,
		m.config.MaxEvents,
	)
	return scanFinishedMsg{report: report, err: err}
}

func (m *model) flattenTree() {
	if m.report == nil {
		return
	}

	m.treeList = []treeNode{}
	for i, root := range m.report.Map.RootNodes {
		m.addNode(root, 0, i == len(m.report.Map.RootNodes)-1)
	}
}

func (m *model) addNode(n *analyzer.MapNode, depth int, isLast bool) {
	key := fmt.Sprintf("%s/%s", n.Kind, n.Name)
	
	// Default groups to expanded
	if n.Kind == "Group" && m.expanded[key] == false {
		// This is a bit of a hack for first run initialization
		if _, ok := m.expanded[key]; !ok {
			m.expanded[key] = true
		}
	}

	tn := treeNode{
		node:     n,
		depth:    depth,
		isLast:   isLast,
		expanded: m.expanded[key],
	}
	m.treeList = append(m.treeList, tn)

	if m.expanded[key] {
		for i, child := range n.Children {
			m.addNode(child, depth+1, i == len(n.Children)-1)
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.report = nil
			m.err = nil
			return m, m.runScan
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.treeList)-1 {
				m.cursor++
			}
		case "right", "l", " ", "enter":
			if len(m.treeList) > 0 {
				n := m.treeList[m.cursor]
				key := fmt.Sprintf("%s/%s", n.node.Kind, n.node.Name)
				m.expanded[key] = !m.expanded[key]
				m.flattenTree()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case scanFinishedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.report = msg.report
			m.flattenTree()
		}
		return m, nil
	}

	return m, nil
}

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("15"))
	
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	detailTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
)

func (m model) getSelectedResult() *analyzer.AnalysisResult {
	if len(m.treeList) == 0 || m.report == nil {
		return nil
	}
	tn := m.treeList[m.cursor]
	
	if tn.node.Kind == "Group" || tn.node.Kind == "Pods" {
		return nil
	}

	searchKey := strings.ToLower(tn.node.Kind) + "/" + tn.node.Name
	for _, res := range m.report.Results {
		if res.Resource == searchKey {
			return &res
		}
	}
	
	// Sometimes services are stored without the "service/" prefix in trace commands?
	// But in scan they have "service/". Let's check both just in case.
	for _, res := range m.report.Results {
		if res.Resource == tn.node.Name {
			return &res
		}
	}

	return nil
}

func renderDetailView(res *analyzer.AnalysisResult) string {
	var s strings.Builder

	// Header
	s.WriteString(lipgloss.NewStyle().Bold(true).Render(res.Resource))
	
	statusColor := "15"
	if res.Severity == "critical" {
		statusColor = "9"
	} else if res.Severity == "warning" {
		statusColor = "11"
	} else if res.Severity == "healthy" {
		statusColor = "10"
	}
	s.WriteString("\nStatus: " + lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(res.Status) + "\n\n")

	// Summary
	if len(res.Summary) > 0 {
		s.WriteString(detailTitleStyle.Render("● Summary"))
		s.WriteString("\n  " + strings.Join(res.Summary, "\n  ") + "\n\n")
	}

	// Findings
	if len(res.Findings) > 0 {
		s.WriteString(detailTitleStyle.Render("● Findings"))
		s.WriteString("\n")
		for _, f := range res.Findings {
			s.WriteString("  ✗ " + f.Message + "\n")
		}
		s.WriteString("\n")
	}

	// Evidence
	if len(res.Evidence) > 0 {
		s.WriteString(detailTitleStyle.Render("● Evidence"))
		s.WriteString("\n")
		for _, e := range res.Evidence {
			s.WriteString(fmt.Sprintf("  • %s: %s\n", e.Label, e.Value))
		}
		s.WriteString("\n")
	}

	// Fix Commands
	if len(res.FixCommands) > 0 {
		s.WriteString(detailTitleStyle.Render("● Suggested Fixes"))
		s.WriteString("\n")
		for _, f := range res.FixCommands {
			if f.Description != "" {
				s.WriteString("  # " + f.Description + "\n")
			}
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("  $ " + f.Command) + "\n\n")
		}
	}

	return s.String()
}

func (m model) View() string {
	var s strings.Builder

	// Header
	s.WriteString(headerStyle.Render(fmt.Sprintf("Kubectl-Why Dash · Namespace: %s", m.config.Namespace)))
	s.WriteString("\n")

	if m.err != nil {
		s.WriteString(fmt.Sprintf("\n  Error: %v\n", m.err))
	} else if m.loading {
		s.WriteString(fmt.Sprintf("\n  %s Scanning namespace...\n", m.spinner.View()))
	} else if len(m.treeList) > 0 {
		
		var treePane strings.Builder
		for i, tn := range m.treeList {
			// Add blank line before top-level groups (except the first one) to improve spacing
			if tn.depth == 0 && i > 0 {
				treePane.WriteString("\n")
			}

			prefix := strings.Repeat("  ", tn.depth)
			
			// Selection indicator
			cursor := " "
			if i == m.cursor {
				cursor = "❯"
			}

			// Expanded/Collapsed indicator
			toggle := " "
			if len(tn.node.Children) > 0 {
				if tn.expanded {
					toggle = "▼"
				} else {
					toggle = "▶"
				}
			}

			// Status Icon
			icon := "🟢"
			switch tn.node.Severity {
			case "critical": icon = "🔴"
			case "warning": icon = "🟡"
			}

			lineText := fmt.Sprintf("%s %s%s %s %s: %s", 
				cursor, prefix, toggle, icon, tn.node.Kind, tn.node.Name)
			
			if i == m.cursor {
				treePane.WriteString(selectedStyle.Render(lineText))
			} else {
				treePane.WriteString(lineText)
			}

			if tn.node.Status != "Healthy" && tn.node.Status != "" {
				treePane.WriteString(fmt.Sprintf(" [%s]", tn.node.Status))
			}
			
			treePane.WriteString("\n")
		}

		var detailPane strings.Builder
		res := m.getSelectedResult()
		if res != nil {
			detailPane.WriteString(renderDetailView(res))
		} else if len(m.treeList) > 0 {
			tn := m.treeList[m.cursor]
			detailPane.WriteString(lipgloss.NewStyle().Bold(true).Render(tn.node.Kind + "/" + tn.node.Name) + "\n\n")
			if tn.node.Message != "" {
				detailPane.WriteString(tn.node.Message + "\n")
			} else {
				detailPane.WriteString(dimStyle.Render("Select a specific resource to view details."))
			}
		}

		leftWidth := m.width / 2
		if leftWidth < 50 { leftWidth = 50 } // minimum width for tree
		if leftWidth > 80 { leftWidth = 80 } // maximum width for tree

		treeView := lipgloss.NewStyle().Width(leftWidth).Render(treePane.String())
		
		// Add left border to detail view
		detailView := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("240")).
			PaddingLeft(2).
			Width(m.width - leftWidth - 4).
			Render(detailPane.String())

		content := lipgloss.JoinHorizontal(lipgloss.Top, treeView, detailView)
		s.WriteString(content)

	} else {
		s.WriteString("\n  No resources found.\n")
	}

	// Footer
	s.WriteString("\n\n  (arrows: navigate · space/enter: expand · r: refresh · q: quit)")

	return s.String()
}

func Start(cfg Config) error {
	p := tea.NewProgram(NewModel(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
