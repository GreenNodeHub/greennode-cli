package cliinput

import (
	"strings"
	"testing"
)

func setup(t *testing.T, input string, interactive_ bool) {
	t.Helper()
	t.Cleanup(func() {
		SetInteractive(false)
		SetReader(strings.NewReader(""))
	})
	SetInteractive(interactive_)
	SetReader(strings.NewReader(input))
}

func TestRequireOrPromptString_ValueProvided(t *testing.T) {
	setup(t, "", false)
	got, err := RequireOrPromptString("hello", "--name", "Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestRequireOrPromptString_NonInteractive_Empty(t *testing.T) {
	setup(t, "", false)
	_, err := RequireOrPromptString("", "--name", "Name")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRequireOrPromptString_Interactive_UserInput(t *testing.T) {
	setup(t, "my-name\n", true)
	got, err := RequireOrPromptString("", "--name", "Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "my-name" {
		t.Errorf("expected 'my-name', got %q", got)
	}
}

func TestRequireOrPromptStringWithPlaceholder_Interactive_UserInput(t *testing.T) {
	setup(t, "my-desc\n", true)
	got, err := RequireOrPromptStringWithPlaceholder("", "--description", "Description", "hint text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "my-desc" {
		t.Errorf("expected 'my-desc', got %q", got)
	}
}

func TestRequireOrPromptString_Interactive_EmptyInput(t *testing.T) {
	setup(t, "\n", true)
	_, err := RequireOrPromptString("", "--name", "Name")
	if err == nil {
		t.Fatal("expected error for empty interactive input")
	}
}

func TestRequireOrPromptSecret_ValueProvided(t *testing.T) {
	setup(t, "", false)
	got, err := RequireOrPromptSecret("s3cr3t", "--secret", "Secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "s3cr3t" {
		t.Errorf("expected 's3cr3t', got %q", got)
	}
}

func TestRequireOrPromptSecret_NonInteractive_Empty(t *testing.T) {
	setup(t, "", false)
	_, err := RequireOrPromptSecret("", "--secret", "Secret")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRequireOrPromptStringSlice_ValueProvided(t *testing.T) {
	setup(t, "", false)
	got, err := RequireOrPromptStringSlice([]string{"a", "b"}, "--scope", "Scopes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestRequireOrPromptStringSlice_NonInteractive_Empty(t *testing.T) {
	setup(t, "", false)
	_, err := RequireOrPromptStringSlice(nil, "--scope", "Scopes")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRequireOrPromptStringSlice_Interactive_CommaSeparated(t *testing.T) {
	setup(t, "read, write , admin\n", true)
	got, err := RequireOrPromptStringSlice(nil, "--scope", "Scopes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 || got[0] != "read" || got[1] != "write" || got[2] != "admin" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestRequireOrPromptStringSlice_Interactive_Empty(t *testing.T) {
	setup(t, "\n", true)
	_, err := RequireOrPromptStringSlice(nil, "--scope", "Scopes")
	if err == nil {
		t.Fatal("expected error for empty interactive input")
	}
}

func TestPromptIntDefault_Valid(t *testing.T) {
	setup(t, "3\n", true)
	got := PromptIntDefault("Replicas", 1)
	if got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestPromptIntDefault_Empty_UsesDefault(t *testing.T) {
	setup(t, "\n", true)
	got := PromptIntDefault("Replicas", 5)
	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestPromptIntDefault_Invalid_UsesDefault(t *testing.T) {
	setup(t, "notanumber\n", true)
	got := PromptIntDefault("Replicas", 2)
	if got != 2 {
		t.Errorf("expected 2 (default), got %d", got)
	}
}

func TestPromptChoice_SelectSecond(t *testing.T) {
	setup(t, "2\n", true)
	idx, err := PromptChoice("Pick one", []string{"alpha", "beta", "gamma"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 1 {
		t.Errorf("expected index 1 (beta), got %d", idx)
	}
}

func TestPromptChoice_EmptyInput_DefaultsToFirst(t *testing.T) {
	setup(t, "\n", true)
	idx, err := PromptChoice("Pick one", []string{"alpha", "beta"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestPromptChoice_OutOfRange_DefaultsToFirst(t *testing.T) {
	setup(t, "99\n", true)
	idx, err := PromptChoice("Pick one", []string{"alpha", "beta"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected index 0, got %d", idx)
	}
}

func TestPromptChoice_NoItems(t *testing.T) {
	setup(t, "", true)
	_, err := PromptChoice("Pick one", nil)
	if err == nil {
		t.Fatal("expected error for empty items list")
	}
}

func TestConfirm_Yes(t *testing.T) {
	setup(t, "y\n", true)
	if !Confirm("Proceed?") {
		t.Error("expected true for 'y'")
	}
}

func TestConfirm_UpperY(t *testing.T) {
	setup(t, "Y\n", true)
	if !Confirm("Proceed?") {
		t.Error("expected true for 'Y'")
	}
}

func TestConfirm_No(t *testing.T) {
	setup(t, "n\n", true)
	if Confirm("Proceed?") {
		t.Error("expected false for 'n'")
	}
}

func TestConfirm_Empty(t *testing.T) {
	setup(t, "\n", true)
	if Confirm("Proceed?") {
		t.Error("expected false for empty input")
	}
}

func TestPromptString(t *testing.T) {
	setup(t, "typed-value\n", true)
	got, err := PromptString("Enter value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "typed-value" {
		t.Errorf("expected 'typed-value', got %q", got)
	}
}

func TestIsInteractive(t *testing.T) {
	SetInteractive(false)
	if IsInteractive() {
		t.Error("expected false")
	}
	SetInteractive(true)
	if !IsInteractive() {
		t.Error("expected true")
	}
	SetInteractive(false)
}
