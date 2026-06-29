package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	internalauth "github.com/mudongliang/weoa-cli/internal/auth"
)

// NewLoginCmd creates the "auth login" subcommand.
func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in to WeChat Official Account backend",
		Long: `Opens a browser window to mp.weixin.qq.com.
Scan the QR code with WeChat to log in.
Session (cookies + token) is saved to ~/.config/weoa-cli/session.json.`,
		RunE: runLogin,
	}
}

func runLogin(cmd *cobra.Command, args []string) error {
	fmt.Println("Launching browser for login...")

	ctx := context.Background()
	result, err := internalauth.Login(ctx)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	fmt.Printf("\n✓ Login successful\n")
	if result.AccountName != "" {
		fmt.Printf("  Account: %s\n", result.AccountName)
	}
	fmt.Printf("  Session saved to ~/.config/weoa-cli/session.json\n")

	return nil
}
