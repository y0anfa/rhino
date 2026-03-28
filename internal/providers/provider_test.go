package providers

import "testing"

func TestRegisterAndGet(t *testing.T) {
	// Shell and HTTP are registered via init()
	p, err := Get("shell")
	if err != nil {
		t.Fatalf("expected shell provider, got error: %v", err)
	}
	if p.Name() != "shell" {
		t.Errorf("expected name=shell, got %s", p.Name())
	}

	p, err = Get("http")
	if err != nil {
		t.Fatalf("expected http provider, got error: %v", err)
	}
	if p.Name() != "http" {
		t.Errorf("expected name=http, got %s", p.Name())
	}
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestList(t *testing.T) {
	list := List()
	if len(list) < 2 {
		t.Errorf("expected at least 2 providers, got %d", len(list))
	}
	found := map[string]bool{}
	for _, name := range list {
		found[name] = true
	}
	if !found["shell"] {
		t.Error("expected shell in provider list")
	}
	if !found["http"] {
		t.Error("expected http in provider list")
	}
}
