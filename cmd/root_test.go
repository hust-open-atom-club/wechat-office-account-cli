package cmd

import (
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLeafCommandsRejectUnexpectedArgs(t *testing.T) {
	tests := []struct {
		name      string
		argv      []string
		flag      string
		flagValue string
	}{
		{name: "whoami", argv: []string{"whoami"}},
		{name: "auth login", argv: []string{"auth", "login", "--qr"}, flag: "qr", flagValue: "true"},
		{name: "auth logout", argv: []string{"auth", "logout"}},
		{name: "auth status", argv: []string{"auth", "status"}},
		{name: "publish list", argv: []string{"publish", "list", "--search", "kernel"}, flag: "search", flagValue: "kernel"},
		{name: "publish sync", argv: []string{"publish", "sync"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaf, _, err := rootCmd.Find(tt.argv)
			if err != nil || leaf == nil || leaf.RunE == nil {
				t.Fatalf("find runnable leaf for %v: %v", tt.argv, err)
			}
			originalRun := leaf.RunE
			t.Cleanup(func() {
				leaf.RunE = originalRun
				rootCmd.SetArgs(nil)
			})
			if tt.flag != "" {
				flag := leaf.Flags().Lookup(tt.flag)
				if flag == nil {
					t.Fatalf("missing flag %q", tt.flag)
				}
				value, changed := flag.Value.String(), flag.Changed
				t.Cleanup(func() {
					if err := flag.Value.Set(value); err != nil {
						t.Errorf("restore flag %q: %v", tt.flag, err)
					}
					flag.Changed = changed
				})
			}

			calls := 0
			leaf.RunE = func(command *cobra.Command, args []string) error {
				calls++
				if len(args) != 0 {
					t.Errorf("handler received unexpected positionals: %v", args)
				}
				if tt.flag != "" && command.Flags().Lookup(tt.flag).Value.String() != tt.flagValue {
					t.Errorf("flag %q was not parsed as %q", tt.flag, tt.flagValue)
				}
				return nil
			}

			rootCmd.SetArgs(tt.argv)
			if err := Execute(); err != nil || calls != 1 {
				t.Fatalf("valid invocation: handler calls = %d, error = %v", calls, err)
			}

			rootCmd.SetArgs(append(slices.Clone(tt.argv), "unexpected"))
			err = Execute()
			if err == nil || !strings.Contains(err.Error(), `unknown command "unexpected" for`) {
				t.Errorf("want Cobra's positional-argument error, got %v", err)
			}
			if calls != 1 {
				t.Errorf("invalid invocation reached the handler: calls = %d, want 1", calls)
			}
		})
	}
}
