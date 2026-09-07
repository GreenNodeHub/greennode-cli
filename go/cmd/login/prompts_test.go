package login

import (
	"bufio"
	"errors"
	"strings"
	"testing"
)

func TestPromptMenu(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{"choice 1 -> index 0", "1\n", 0},
		{"choice 2 -> index 1", "2\n", 1},
		{"Enter -> default 0", "\n", 0},
		{"EOF -> default 0", "", 0},
		{"invalid then valid", "9\n2\n", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tc.input))
			got, err := promptMenu(r, "Choose:", []string{"pkce", "sso"})
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPromptSSOLink(t *testing.T) {
	t.Run("valid https accepted", func(t *testing.T) {
		r := bufio.NewReader(strings.NewReader("https://sso.example/auth\n"))
		got, err := promptSSOLink(r)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != "https://sso.example/auth" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("non-https then valid", func(t *testing.T) {
		r := bufio.NewReader(strings.NewReader("http://insecure\nhttps://sso.example/auth\n"))
		got, err := promptSSOLink(r)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != "https://sso.example/auth" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("three invalid -> error", func(t *testing.T) {
		r := bufio.NewReader(strings.NewReader("bad1\nbad2\nbad3\n"))
		_, err := promptSSOLink(r)
		if err == nil {
			t.Fatal("expected error after 3 invalid, got nil")
		}
	})
	t.Run("EOF empty -> error", func(t *testing.T) {
		r := bufio.NewReader(strings.NewReader(""))
		_, err := promptSSOLink(r)
		if err == nil {
			t.Fatal("expected error on EOF, got nil")
		}
	})
}

func TestConfirmOpen(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantOpen bool
		wantErr  bool
	}{
		{"Enter -> open", "\n", true, false},
		{"y -> open", "y\n", true, false},
		{"s -> skip", "s\n", false, false},
		{"S -> skip", "S\n", false, false},
		{"unknown -> skip", "abc\n", false, false},
		{"EOF -> cancel", "", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tc.input))
			open, err := confirmOpen(r)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, errLoginCanceled) {
					t.Errorf("err=%v, want errLoginCanceled", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if open != tc.wantOpen {
				t.Errorf("open=%v, want %v", open, tc.wantOpen)
			}
		})
	}
}
