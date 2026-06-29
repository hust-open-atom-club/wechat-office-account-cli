package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	authcmd "github.com/mudongliang/weoa-cli/cmd/auth"
	publishcmd "github.com/mudongliang/weoa-cli/cmd/publish"
)

var rootCmd = &cobra.Command{
	Use:   "weoa-cli",
	Short: "Manage WeChat Official Account from the terminal",
	Long: `weoa-cli is a CLI tool for managing a WeChat Official Account (公众号) backend.

It provides commands to log in, view account info, list and sync published
articles, and more — all from the terminal.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	// Register subcommands
	rootCmd.AddCommand(authCmd())
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(publishCmd())
}

// authCmd creates the "auth" parent command.
func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
		Long:  `Manage login sessions for the WeChat Official Account backend.`,
	}
	cmd.AddCommand(authcmd.NewLoginCmd())
	return cmd
}

// publishCmd creates the "publish" parent command.
func publishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Published article management",
		Long:  `List and sync published articles from the WeChat Official Account backend.`,
	}
	cmd.AddCommand(publishcmd.NewListCmd())
	cmd.AddCommand(publishcmd.NewSyncCmd())
	return cmd
}

// fatalf prints an error message and exits with code 1.
func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
