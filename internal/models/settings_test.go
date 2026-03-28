package models

import "testing"

func TestNewSettings_Defaults(t *testing.T) {
	s := NewSettings(-1, "")
	if s.MaxTries != MaxTriesDefault {
		t.Errorf("expected MaxTries=%d, got %d", MaxTriesDefault, s.MaxTries)
	}
	if s.Timeout != TimeoutDefault {
		t.Errorf("expected Timeout=%s, got %s", TimeoutDefault, s.Timeout)
	}
}

func TestNewSettings_ZeroMaxTries(t *testing.T) {
	s := NewSettings(0, "5s")
	if s.MaxTries != 0 {
		t.Errorf("expected MaxTries=0 (no fallback for zero), got %d", s.MaxTries)
	}
}

func TestNewSettings_NegativeMaxTries(t *testing.T) {
	s := NewSettings(-1, "10s")
	if s.MaxTries != MaxTriesDefault {
		t.Errorf("expected MaxTries=%d for negative input, got %d", MaxTriesDefault, s.MaxTries)
	}
	if s.Timeout != "10s" {
		t.Errorf("expected Timeout=10s, got %s", s.Timeout)
	}
}

func TestNewSettings_CustomValues(t *testing.T) {
	s := NewSettings(5, "30s")
	if s.MaxTries != 5 {
		t.Errorf("expected MaxTries=5, got %d", s.MaxTries)
	}
	if s.Timeout != "30s" {
		t.Errorf("expected Timeout=30s, got %s", s.Timeout)
	}
}
