package auth

import (
	"fmt"

	"github.com/mudongliang/weoa-cli/internal/auth"
	"github.com/mudongliang/weoa-cli/internal/client"
	"github.com/spf13/cobra"
)

// NewStatusCmd creates the "auth status" subcommand.
func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show login status",
		Long:  `Checks whether a saved session exists and optionally verifies it against the backend.`,
		RunE: runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	if !auth.Exists() {
		fmt.Println("Not logged in.")
		fmt.Println("Run 'weoa-cli auth login' to log in.")
		return nil
	}

	session, err := auth.Load()
	if err != nil {
		return fmt.Errorf("load session: %w — try 'weoa-cli auth login'", err)
	}

	// Try to fetch account info to verify the session is still valid
	info, err := client.FetchAccountInfo(session)
	if err != nil {
		fmt.Println("Session expired or invalid.")
		fmt.Println("Run 'weoa-cli auth login' to re-authenticate.")
		return nil
	}

	fmt.Printf("✓ Logged in\n")
	fmt.Printf("  Account:  %s\n", info.Nickname)
	fmt.Printf("  WeChat:   %s\n", info.WeChatID)
	fmt.Printf("  Session:  ~/.config/weoa-cli/session.json\n")

	return nil
}
