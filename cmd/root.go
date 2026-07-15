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

	rootCmd.AddCommand(authCmd())
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(publishCmd())
}

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
		Long:  `Manage login sessions for the WeChat Official Account backend.`,
	}
	cmd.AddCommand(authcmd.NewLoginCmd())
	cmd.AddCommand(authcmd.NewLogoutCmd())
	cmd.AddCommand(authcmd.NewStatusCmd())
	return cmd
}

func publishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Published article management",
		Long:  `List and sync published articles from the WeChat Official Account backend.`,
	}
	cmd.AddCommand(publishcmd.NewListCmd())
	cmd.AddCommand(publishcmd.NewSyncCmd())
	cmd.AddCommand(publishcmd.NewExportCmd())
	return cmd
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
