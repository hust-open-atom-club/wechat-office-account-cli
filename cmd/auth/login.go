package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	internalauth "github.com/mudongliang/weoa-cli/internal/auth"
)

var loginQR bool

// NewLoginCmd creates the "auth login" subcommand.
func NewLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to WeChat Official Account backend",
		Long: `Opens a browser window to mp.weixin.qq.com by default.
Use --qr to print the login QR code directly in the terminal.
Session (cookies + token) is saved to ~/.config/weoa-cli/session.json.`,
		RunE: runLogin,
	}
	cmd.Flags().BoolVar(&loginQR, "qr", false, "Print the login QR code in the terminal without opening a browser window")
	return cmd
}

func runLogin(cmd *cobra.Command, args []string) error {
	if loginQR {
		fmt.Println("Preparing QR login...")
	} else {
		fmt.Println("Launching browser for login...")
	}

	ctx := context.Background()
	result, err := internalauth.Login(ctx, internalauth.LoginOptions{
		PrintQR: loginQR,
	})
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
