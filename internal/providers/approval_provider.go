package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var (
	pendingApprovals = make(map[string]chan bool)
	approvalMu       sync.Mutex
)

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

	ch := make(chan bool, 1)
	approvalMu.Lock()
	pendingApprovals[id] = ch
	approvalMu.Unlock()

	defer func() {
		approvalMu.Lock()
		delete(pendingApprovals, id)
		approvalMu.Unlock()
	}()

	fmt.Printf("\n  [APPROVAL REQUIRED] %s\n  Approve with: rhino approve %s\n  Reject with:  rhino reject %s\n\n", message, id, id)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case approved := <-ch:
		if approved {
			return &TaskResult{
				Output:   "approved",
				Metadata: map[string]string{"approval_id": id, "decision": "approved"},
			}, nil
		}
		return nil, fmt.Errorf("approval rejected")
	case <-timer.C:
		return nil, fmt.Errorf("approval timed out after %s", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Approve approves a pending approval by ID.
func Approve(id string) error {
	approvalMu.Lock()
	ch, ok := pendingApprovals[id]
	approvalMu.Unlock()
	if !ok {
		return fmt.Errorf("approval '%s' not found or already resolved", id)
	}
	ch <- true
	return nil
}

// Reject rejects a pending approval by ID.
func Reject(id string) error {
	approvalMu.Lock()
	ch, ok := pendingApprovals[id]
	approvalMu.Unlock()
	if !ok {
		return fmt.Errorf("approval '%s' not found or already resolved", id)
	}
	ch <- false
	return nil
}

// ListPendingApprovals returns IDs of pending approvals.
func ListPendingApprovals() []string {
	approvalMu.Lock()
	defer approvalMu.Unlock()
	ids := make([]string, 0, len(pendingApprovals))
	for id := range pendingApprovals {
		ids = append(ids, id)
	}
	return ids
}

func init() {
	Register(&ApprovalProvider{})
}
