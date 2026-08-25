package providers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type approvalOutcome struct {
	result *TaskResult
	err    error
}

func runApproval(t *testing.T, args map[string]interface{}) chan approvalOutcome {
	t.Helper()
	out := make(chan approvalOutcome, 1)
	go func() {
		p := &ApprovalProvider{}
		res, err := p.Run(context.Background(), args)
		out <- approvalOutcome{res, err}
	}()
	return out
}

func waitForPendingApproval(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ids := ListPendingApprovals(); len(ids) == 1 {
			return ids[0]
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no pending approval was recorded")
	return ""
}

// `rhino approve` runs in a different process than the workflow, so the decision
// has to travel through the approvals directory.
func TestApprovalProvider_ApproveOutOfProcess(t *testing.T) {
	t.Setenv(EnvApprovalsDir, t.TempDir())

	out := runApproval(t, map[string]interface{}{"message": "deploy?", "timeout": "10s"})
	id := waitForPendingApproval(t)

	if err := Approve(id); err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	select {
	case got := <-out:
		if got.err != nil {
			t.Fatalf("unexpected error: %v", got.err)
		}
		if got.result.Output != "approved" {
			t.Errorf("expected output 'approved', got %q", got.result.Output)
		}
		if got.result.Metadata["approval_id"] != id {
			t.Errorf("expected approval_id %q, got %q", id, got.result.Metadata["approval_id"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the approval task never observed the decision")
	}

	if ids := ListPendingApprovals(); len(ids) != 0 {
		t.Errorf("expected the pending approval to be cleaned up, got %v", ids)
	}
}

// The decision is picked up from the approvals directory, exactly as another
// process (the `rhino approve` CLI) writes it.
func TestApprovalProvider_ReadsDecisionWrittenByAnotherProcess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvApprovalsDir, dir)

	out := runApproval(t, map[string]interface{}{"message": "deploy?", "timeout": "10s"})
	id := waitForPendingApproval(t)

	if err := os.WriteFile(filepath.Join(dir, id+".decision"), []byte("approved"), 0600); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-out:
		if got.err != nil {
			t.Fatalf("unexpected error: %v", got.err)
		}
		if got.result.Output != "approved" {
			t.Errorf("expected output 'approved', got %q", got.result.Output)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the approval task never observed the decision file")
	}
}

func TestApprovalProvider_RejectOutOfProcess(t *testing.T) {
	t.Setenv(EnvApprovalsDir, t.TempDir())

	out := runApproval(t, map[string]interface{}{"message": "deploy?", "timeout": "10s"})
	id := waitForPendingApproval(t)

	if err := Reject(id); err != nil {
		t.Fatalf("Reject failed: %v", err)
	}

	select {
	case got := <-out:
		if got.err == nil || !strings.Contains(got.err.Error(), "rejected") {
			t.Fatalf("expected a rejection error, got %v", got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the approval task never observed the decision")
	}
}

func TestApprovalProvider_Timeout(t *testing.T) {
	t.Setenv(EnvApprovalsDir, t.TempDir())

	p := &ApprovalProvider{}
	_, err := p.Run(context.Background(), map[string]interface{}{"timeout": "300ms"})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected a timeout error, got %v", err)
	}
	if ids := ListPendingApprovals(); len(ids) != 0 {
		t.Errorf("expected the pending approval to be cleaned up, got %v", ids)
	}
}

func TestApprove_UnknownID(t *testing.T) {
	t.Setenv(EnvApprovalsDir, t.TempDir())

	if err := Approve("approval-does-not-exist"); err == nil {
		t.Error("expected an error for an unknown approval id")
	}
	if err := Approve("../escape"); err == nil {
		t.Error("expected an error for an id that escapes the approvals directory")
	}
}
