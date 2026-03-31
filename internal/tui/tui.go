package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/y0anfa/rhino/internal/models"
	"github.com/y0anfa/rhino/internal/store"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	statusSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	statusFailed  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	statusRunning = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type view int

const (
	viewWorkflows view = iota
	viewHistory
	viewRunDetail
)

type model struct {
	workflows    []string
	runs         []*store.WorkflowRun
	taskExecs    []*store.TaskExecution
	selectedRun  *store.WorkflowRun
	currentView  view
	cursor       int
	width        int
	height       int
	err          error
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func NewModel() model {
	return model{
		currentView: viewWorkflows,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadWorkflows, m.loadHistory, tickCmd())
}

func (m model) loadWorkflows() tea.Msg {
	wfs, err := models.ListWorkflows()
	if err != nil {
		return errMsg{err}
	}
	return workflowsMsg(wfs)
}

func (m model) loadHistory() tea.Msg {
	s := store.Global()
	if s == nil {
		return runsMsg(nil)
	}
	runs, err := s.ListRuns(store.RunFilter{Limit: 100})
	if err != nil {
		return errMsg{err}
	}
	return runsMsg(runs)
}

type workflowsMsg []string
type runsMsg []*store.WorkflowRun
type taskExecsMsg []*store.TaskExecution
type errMsg struct{ err error }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.currentView == viewWorkflows {
				return m, tea.Quit
			}
			m.currentView = viewWorkflows
			m.cursor = 0
			return m, nil
		case "tab":
			if m.currentView == viewWorkflows {
				m.currentView = viewHistory
			} else {
				m.currentView = viewWorkflows
			}
			m.cursor = 0
			return m, nil
		case "j", "down":
			m.cursor++
			m.clampCursor()
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "enter":
			return m.handleEnter()
		case "esc":
			if m.currentView != viewWorkflows {
				m.currentView = viewWorkflows
				m.cursor = 0
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case workflowsMsg:
		m.workflows = msg
		return m, nil

	case runsMsg:
		m.runs = msg
		return m, nil

	case taskExecsMsg:
		m.taskExecs = msg
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.loadHistory, tickCmd())
	}

	return m, nil
}

func (m *model) clampCursor() {
	max := 0
	switch m.currentView {
	case viewWorkflows:
		max = len(m.workflows) - 1
	case viewHistory:
		max = len(m.runs) - 1
	case viewRunDetail:
		max = len(m.taskExecs) - 1
	}
	if max < 0 {
		max = 0
	}
	if m.cursor > max {
		m.cursor = max
	}
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	if m.currentView == viewHistory && len(m.runs) > 0 && m.cursor < len(m.runs) {
		run := m.runs[m.cursor]
		m.selectedRun = run
		m.currentView = viewRunDetail
		m.cursor = 0
		return m, func() tea.Msg {
			s := store.Global()
			if s == nil {
				return taskExecsMsg(nil)
			}
			execs, _ := s.GetTaskExecutions(run.ID)
			return taskExecsMsg(execs)
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var tabs string
	wfTab := " Workflows "
	histTab := " History "
	if m.currentView == viewWorkflows {
		wfTab = selectedStyle.Render(" Workflows ")
	} else if m.currentView == viewHistory || m.currentView == viewRunDetail {
		histTab = selectedStyle.Render(" History ")
	}
	tabs = headerStyle.Render(wfTab + " | " + histTab)

	title := titleStyle.Render("  Rhino Dashboard")
	header := title + "\n" + tabs + "\n" + dimStyle.Render(repeat("─", min(m.width, 80))) + "\n"

	var body string
	switch m.currentView {
	case viewWorkflows:
		body = m.viewWorkflows()
	case viewHistory:
		body = m.viewHistory()
	case viewRunDetail:
		body = m.viewRunDetail()
	}

	help := dimStyle.Render("\n  j/k: navigate  tab: switch view  enter: select  q: quit")

	return header + body + help
}

func (m model) viewWorkflows() string {
	if len(m.workflows) == 0 {
		return "  No workflows found.\n"
	}
	s := ""
	for i, wf := range m.workflows {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		s += style.Render(cursor+wf) + "\n"
	}
	return s
}

func (m model) viewHistory() string {
	if len(m.runs) == 0 {
		return "  No runs recorded yet.\n"
	}

	s := fmt.Sprintf("  %-14s  %-18s  %-9s  %s\n",
		headerStyle.Render("ID"),
		headerStyle.Render("WORKFLOW"),
		headerStyle.Render("STATUS"),
		headerStyle.Render("STARTED"))
	s += dimStyle.Render("  "+repeat("─", 65)) + "\n"

	for i, r := range m.runs {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		id := r.ID
		if len(id) > 14 {
			id = id[:14]
		}
		name := r.WorkflowName
		if len(name) > 18 {
			name = name[:18]
		}

		status := formatStatus(r.Status)
		if i == m.cursor {
			status = style.Render(string(r.Status))
		}

		started := r.StartedAt.Format("15:04:05")
		line := fmt.Sprintf("%-14s  %-18s  %-9s  %s", id, name, status, started)
		s += style.Render(cursor) + line + "\n"
	}
	return s
}

func (m model) viewRunDetail() string {
	if m.selectedRun == nil {
		return "  No run selected.\n"
	}
	r := m.selectedRun
	s := fmt.Sprintf("  Run:      %s\n", r.ID)
	s += fmt.Sprintf("  Workflow: %s\n", r.WorkflowName)
	s += fmt.Sprintf("  Status:   %s\n", formatStatus(r.Status))
	s += fmt.Sprintf("  Started:  %s\n", r.StartedAt.Format(time.RFC3339))
	if !r.CompletedAt.IsZero() {
		s += fmt.Sprintf("  Duration: %s\n", r.CompletedAt.Sub(r.StartedAt).Truncate(time.Millisecond))
	}
	if r.Error != "" {
		s += fmt.Sprintf("  Error:    %s\n", statusFailed.Render(r.Error))
	}

	if len(m.taskExecs) > 0 {
		s += "\n  " + headerStyle.Render("Tasks:") + "\n"
		for i, t := range m.taskExecs {
			cursor := "    "
			if i == m.cursor {
				cursor = "  > "
			}
			dur := time.Duration(t.DurationMs) * time.Millisecond
			status := formatTaskStatus(t.Status)
			s += fmt.Sprintf("%s%-18s  %s  %s\n", cursor, t.TaskName, status, dur)
			if t.Error != "" {
				s += fmt.Sprintf("      %s\n", statusFailed.Render(t.Error))
			}
		}
	}

	return s
}

func formatStatus(s store.RunStatus) string {
	switch s {
	case store.RunStatusSuccess:
		return statusSuccess.Render("success")
	case store.RunStatusFailed:
		return statusFailed.Render("failed")
	case store.RunStatusRunning:
		return statusRunning.Render("running")
	default:
		return string(s)
	}
}

func formatTaskStatus(s store.TaskStatus) string {
	switch s {
	case store.TaskStatusSuccess:
		return statusSuccess.Render("success")
	case store.TaskStatusFailed:
		return statusFailed.Render("failed")
	case store.TaskStatusRunning:
		return statusRunning.Render("running")
	case store.TaskStatusSkipped:
		return dimStyle.Render("skipped")
	default:
		return string(s)
	}
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
