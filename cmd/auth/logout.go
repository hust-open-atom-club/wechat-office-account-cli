package auth

import (
	"fmt"

	"github.com/mudongliang/weoa-cli/internal/auth"
	"github.com/spf13/cobra"
)

// NewLogoutCmd creates the "auth logout" subcommand.
func NewLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear saved session",
		RunE:  runLogout,
	}
}

func runLogout(cmd *cobra.Command, args []string) error {
	if !auth.Exists() {
		fmt.Println("Already logged out.")
		return nil
	}

	if err := auth.Clear(); err != nil {
		return fmt.Errorf("clear session: %w", err)
	}

	fmt.Println("✓ Logged out. Session cleared.")
	return nil
}
