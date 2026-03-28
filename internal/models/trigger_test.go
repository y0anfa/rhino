package models

import "testing"

func TestNewTrigger_Manual(t *testing.T) {
	trigger := NewTrigger("t1", "manual trigger", TriggerManual, "")
	if trigger == nil {
		t.Fatal("expected non-nil trigger for manual type")
	}
	if trigger.Type != TriggerManual {
		t.Errorf("expected type=%s, got %s", TriggerManual, trigger.Type)
	}
	if trigger.Schedule != "" {
		t.Errorf("expected empty schedule for manual trigger, got %s", trigger.Schedule)
	}
}

func TestNewTrigger_Scheduled(t *testing.T) {
	trigger := NewTrigger("t1", "cron trigger", TriggerScheduled, "*/5 * * * *")
	if trigger == nil {
		t.Fatal("expected non-nil trigger for scheduled type")
	}
	if trigger.Type != TriggerScheduled {
		t.Errorf("expected type=%s, got %s", TriggerScheduled, trigger.Type)
	}
	if trigger.Schedule != "*/5 * * * *" {
		t.Errorf("expected schedule=*/5 * * * *, got %s", trigger.Schedule)
	}
}

func TestNewTrigger_UnknownType(t *testing.T) {
	trigger := NewTrigger("t1", "unknown", "unknown", "")
	if trigger != nil {
		t.Error("expected nil trigger for unknown type")
	}
}

func TestNewTrigger_Webhook(t *testing.T) {
	trigger := NewTrigger("t1", "webhook trigger", TriggerWebhook, "")
	if trigger != nil {
		t.Error("expected nil trigger for webhook type (not handled in constructor)")
	}
}
