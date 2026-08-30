package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// EnvApprovalsDir overrides where pending approvals are recorded.
	EnvApprovalsDir = "RHINO_APPROVALS_DIR"

	approvalPollInterval = 250 * time.Millisecond
	pendingSuffix        = ".pending"
	decisionSuffix       = ".decision"
	decisionApproved     = "approved"
	decisionRejected     = "rejected"
)

// Approvals are file-backed: the workflow waits on a file that `rhino approve`
// writes, so the gate can be resolved from another process.
func approvalsDir() string {
	if dir := os.Getenv(EnvApprovalsDir); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".rhino", "approvals")
	}
	return filepath.Join(home, ".rhino", "approvals")
}

type pendingApproval struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type ApprovalProvider struct{}

func (p *ApprovalProvider) Name() string { return "approval" }

func (p *ApprovalProvider) Validate(args map[string]interface{}) error {
	for key := range args {
		switch key {
		case "message", "timeout":
			// valid
		default:
			return fmt.Errorf("approval provider validation failed: unknown parameter '%s'", key)
		}
	}
	return nil
}

func (p *ApprovalProvider) Run(ctx context.Context, args map[string]interface{}) (*TaskResult, error) {
	message := "Approval required"
	if m, ok := args["message"].(string); ok && m != "" {
		message = m
	}

	timeout := 30 * time.Minute
	if t, ok := args["timeout"].(string); ok && t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	id := fmt.Sprintf("approval-%d", time.Now().UnixNano())
	dir := approvalsDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("approval provider: failed to create approvals directory: %w", err)
	}

	pending, err := json.Marshal(pendingApproval{ID: id, Message: message, CreatedAt: time.Now()})
	if err != nil {
		return nil, fmt.Errorf("approval provider: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+pendingSuffix), pending, 0600); err != nil {
		return nil, fmt.Errorf("approval provider: failed to record pending approval: %w", err)
	}

	defer func() {
		os.Remove(filepath.Join(dir, id+pendingSuffix))
		os.Remove(filepath.Join(dir, id+decisionSuffix))
	}()

	fmt.Printf("\n  [APPROVAL REQUIRED] %s\n  Approve with: rhino approve %s\n  Reject with:  rhino reject %s\n\n", message, id, id)

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(approvalPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			decision, ok, err := readDecision(dir, id)
			if err != nil {
				return nil, fmt.Errorf("approval provider: %w", err)
			}
			if !ok {
				continue
			}
			if decision == decisionApproved {
				return &TaskResult{
					Output:   decisionApproved,
					Metadata: map[string]string{"approval_id": id, "decision": decisionApproved},
				}, nil
			}
			return nil, fmt.Errorf("approval rejected")
		case <-timer.C:
			return nil, fmt.Errorf("approval timed out after %s", timeout)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func readDecision(dir, id string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, id+decisionSuffix))
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to read approval decision: %w", err)
	}
	return strings.TrimSpace(string(data)), true, nil
}

// Approve approves a pending approval by ID.
func Approve(id string) error { return decide(id, decisionApproved) }

// Reject rejects a pending approval by ID.
func Reject(id string) error { return decide(id, decisionRejected) }

func decide(id, decision string) error {
	if !validApprovalID(id) {
		return fmt.Errorf("invalid approval id '%s'", id)
	}

	dir := approvalsDir()
	if _, err := os.Stat(filepath.Join(dir, id+pendingSuffix)); err != nil {
		return fmt.Errorf("approval '%s' not found or already resolved", id)
	}

	// Write the decision atomically so the waiting workflow never reads a partial file.
	tmp, err := os.CreateTemp(dir, id+decisionSuffix+"-*")
	if err != nil {
		return fmt.Errorf("failed to record approval decision: %w", err)
	}
	if _, err := tmp.WriteString(decision); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("failed to record approval decision: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("failed to record approval decision: %w", err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, id+decisionSuffix)); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("failed to record approval decision: %w", err)
	}
	return nil
}

// validApprovalID keeps an ID from escaping the approvals directory.
func validApprovalID(id string) bool {
	if id == "" || id != filepath.Base(id) || id == "." || id == ".." {
		return false
	}
	return !strings.ContainsAny(id, `/\`)
}

// ListPendingApprovals returns IDs of pending approvals.
func ListPendingApprovals() []string {
	entries, err := os.ReadDir(approvalsDir())
	if err != nil {
		return nil
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), pendingSuffix) {
			continue
		}
		ids = append(ids, strings.TrimSuffix(e.Name(), pendingSuffix))
	}
	sort.Strings(ids)
	return ids
}

func init() {
	Register(&ApprovalProvider{})
}
