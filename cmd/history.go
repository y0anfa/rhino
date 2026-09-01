package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/y0anfa/rhino/internal/store"
)

var (
	historyWorkflow string
	historyStatus   string
	historySince    string
	historyLimit    int
	historyFormat   string
)

var historyCmd = &cobra.Command{
	Use:   "history [run-id]",
	Short: "View workflow execution history",
	Long:  `List recent workflow runs or view details of a specific run by ID.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.NewSQLiteStore(store.DefaultDBPath())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening history: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()

		if len(args) == 1 {
			showRunDetail(s, args[0])
			return
		}

		listRuns(s)
	},
}

func showRunDetail(s *store.SQLiteStore, runID string) {
	run, err := s.GetRun(runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if historyFormat == "json" {
		tasks, _ := s.GetTaskExecutions(runID)
		data, _ := json.MarshalIndent(map[string]interface{}{
			"run":   run,
			"tasks": tasks,
		}, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Run:      %s\n", run.ID)
	fmt.Printf("Workflow: %s\n", run.WorkflowName)
	fmt.Printf("Status:   %s\n", run.Status)
	fmt.Printf("Trigger:  %s\n", run.TriggerType)
	fmt.Printf("Started:  %s\n", run.StartedAt.Format(time.RFC3339))
	if !run.CompletedAt.IsZero() {
		fmt.Printf("Ended:    %s\n", run.CompletedAt.Format(time.RFC3339))
		fmt.Printf("Duration: %s\n", run.CompletedAt.Sub(run.StartedAt).Truncate(time.Millisecond))
	}
	if run.Error != "" {
		fmt.Printf("Error:    %s\n", run.Error)
	}
	if len(run.Inputs) > 0 {
		names := make([]string, 0, len(run.Inputs))
		for name := range run.Inputs {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Println("Inputs:")
		for _, name := range names {
			fmt.Printf("  %s=%s\n", name, run.Inputs[name])
		}
	}

	tasks, err := s.GetTaskExecutions(runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
		return
	}

	if len(tasks) > 0 {
		fmt.Println("\nTasks:")
		for _, t := range tasks {
			dur := time.Duration(t.DurationMs) * time.Millisecond
			fmt.Printf("  %-20s %-10s %10s", t.TaskName, t.Status, dur)
			if t.Retries > 0 {
				fmt.Printf("  (retries: %d)", t.Retries)
			}
			fmt.Println()
			if t.Error != "" {
				fmt.Printf("    error: %s\n", t.Error)
			}
		}
	}

	if showWorkflowYAML && run.WorkflowYAML != "" {
		fmt.Printf("\nWorkflow definition at time of execution:\n%s\n", run.WorkflowYAML)
	}
}

var showWorkflowYAML bool

func listRuns(s *store.SQLiteStore) {
	filter := store.RunFilter{
		WorkflowName: historyWorkflow,
		Status:       store.RunStatus(historyStatus),
		Limit:        historyLimit,
	}

	if historySince != "" {
		d, err := time.ParseDuration(historySince)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid --since duration: %v\n", err)
			os.Exit(1)
		}
		filter.Since = d
	}

	runs, err := s.ListRuns(filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing runs: %v\n", err)
		os.Exit(1)
	}

	if len(runs) == 0 {
		fmt.Println("No runs found.")
		return
	}

	if historyFormat == "json" {
		data, _ := json.MarshalIndent(runs, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("%-14s  %-20s  %-10s  %-8s  %s\n", "ID", "WORKFLOW", "STATUS", "TRIGGER", "STARTED")
	fmt.Printf("%-14s  %-20s  %-10s  %-8s  %s\n", "──────────────", "────────────────────", "──────────", "────────", "───────────────────")
	for _, r := range runs {
		id := r.ID
		if len(id) > 14 {
			id = id[:14]
		}
		name := r.WorkflowName
		if len(name) > 20 {
			name = name[:20]
		}
		fmt.Printf("%-14s  %-20s  %-10s  %-8s  %s\n",
			id, name, r.Status, r.TriggerType,
			r.StartedAt.Format("2006-01-02 15:04:05"))
	}
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete old history entries",
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.NewSQLiteStore(store.DefaultDBPath())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()

		before := time.Now().Add(-30 * 24 * time.Hour) // default 30 days
		if beforeStr, _ := cmd.Flags().GetString("before"); beforeStr != "" {
			d, err := time.ParseDuration(beforeStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid --before duration: %v\n", err)
				os.Exit(1)
			}
			before = time.Now().Add(-d)
		}

		deleted, err := s.DeleteRunsBefore(before)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing history: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted %d run(s).\n", deleted)
	},
}

func init() {
	historyCmd.Flags().StringVar(&historyWorkflow, "workflow", "", "Filter by workflow name")
	historyCmd.Flags().StringVar(&historyStatus, "status", "", "Filter by status (success, failed, running)")
	historyCmd.Flags().StringVar(&historySince, "since", "", "Filter runs since duration (e.g. 24h, 7d)")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 50, "Max number of runs to show")
	historyCmd.Flags().StringVarP(&historyFormat, "output", "o", "text", "Output format: text, json")
	historyCmd.Flags().BoolVar(&showWorkflowYAML, "show-workflow", false, "Show workflow YAML for a specific run")

	historyClearCmd.Flags().String("before", "720h", "Delete runs older than this duration")
	historyCmd.AddCommand(historyClearCmd)
	rootCmd.AddCommand(historyCmd)
}
