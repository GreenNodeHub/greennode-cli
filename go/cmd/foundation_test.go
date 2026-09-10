package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/greennodehub/greennode-cli/internal/cli"
	"github.com/greennodehub/greennode-cli/internal/operation"
	"github.com/spf13/cobra"
)

func TestFoundationRootHelper(t *testing.T) {
	if os.Getenv("GRN_FOUNDATION_HELPER") != "1" {
		return
	}
	args := []string{}
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	rootCmd.AddCommand(&cobra.Command{Use: "foundation-confirm", Run: func(cmd *cobra.Command, args []string) {
		cli.Confirm(false, "confirm?")
	}})
	rootCmd.AddCommand(operation.NewCommand(operation.Spec[struct{}]{ServiceName: "fixture"},
		operation.Descriptor{Use: "foundation-dry-run", Short: "Offline fixture", Method: "DELETE", Path: "/fixture", Mutation: true, Destructive: true}))
	rootCmd.SetArgs(args)
	Execute()
	os.Exit(0)
}

func TestFoundationRootOfflineSafety(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    string
		success bool
	}{
		{"confirmation", []string{"foundation-confirm", "--non-interactive"}, "confirmation required", false},
		{"configure", []string{"configure", "--non-interactive"}, "requires interaction", false},
		{"login", []string{"login", "--non-interactive"}, "requires interaction", false},
		{"policy", []string{"vserver", "placement-group", "create", "--name", "offline", "--non-interactive"}, "--policy-id is required", false},
		{"dry-run", []string{"foundation-dry-run", "--dry-run", "--non-interactive"}, "DRY RUN", true},
		{"agentbase", []string{"agentbase", "runtime", "create", "--interactive", "--non-interactive"}, "required", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			args := append([]string{"-test.run=^TestFoundationRootHelper$", "--"}, tc.args...)
			child := exec.CommandContext(ctx, os.Args[0], args...)
			child.Env = []string{"GRN_FOUNDATION_HELPER=1", "HOME=" + t.TempDir()}
			// An open, empty pipe catches accidental prompt reads: EOF would hide them.
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			defer w.Close()
			child.Stdin = r
			out, err := child.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("command hung reading stdin: %s", out)
			}
			if (err == nil) != tc.success || !strings.Contains(string(out), tc.want) {
				t.Fatalf("exit=%v output=%s", err, out)
			}
		})
	}
}

func TestFoundationUsageErrorsExitTwo(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"invalid flag", []string{"--unknown"}},
		{"invalid output", []string{"--output", "xml"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"-test.run=^TestFoundationRootHelper$", "--"}, tc.args...)
			child := exec.Command(os.Args[0], args...)
			child.Env = []string{"GRN_FOUNDATION_HELPER=1", "HOME=" + t.TempDir()}
			out, err := child.CombinedOutput()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
				t.Fatalf("exit=%v output=%s", err, out)
			}
		})
	}
}
