package cmd

import (
	"fmt"
	"strings"

	"github.com/mudongliang/weoa-cli/internal/auth"
	"github.com/mudongliang/weoa-cli/internal/client"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current logged-in account info",
	Long:  `Fetches the WeChat Official Account settings page and displays detailed account profile information.`,
	RunE:  runWhoami,
}

func runWhoami(cmd *cobra.Command, args []string) error {
	session, err := auth.Load()
	if err != nil {
		return fmt.Errorf("not logged in — run 'weoa-cli auth login' first")
	}

	info, err := client.FetchAccountInfo(session)
	if err != nil {
		return fmt.Errorf("fetch account info: %w", err)
	}

	// Build display
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s\n", info.Nickname))
	sb.WriteString("────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("  名称:     %s\n", info.Nickname))
	if info.WeChatID != "" {
		sb.WriteString(fmt.Sprintf("  微信号:   %s\n", info.WeChatID))
	}
	sb.WriteString(fmt.Sprintf("  原始ID:   %s\n", info.OriginalID))
	if info.Signature != "" {
		sb.WriteString(fmt.Sprintf("  简介:     %s\n", info.Signature))
	}
	if info.Email != "" {
		sb.WriteString(fmt.Sprintf("  邮箱:     %s\n", info.Email))
	}
	sb.WriteString(fmt.Sprintf("  粉丝数:   %d\n", info.FansCount))
	if len(info.Categories) > 0 {
		sb.WriteString(fmt.Sprintf("  分类:     %s\n", strings.Join(info.Categories, ", ")))
	}
	if info.Location != "" {
		sb.WriteString(fmt.Sprintf("  所在地:   %s\n", info.Location))
	}
	if info.FinderNickname != "" {
		sb.WriteString(fmt.Sprintf("  视频号:   %s\n", info.FinderNickname))
	}
	if info.WxNickname != "" {
		sb.WriteString(fmt.Sprintf("  管理员:   %s\n", info.WxNickname))
	}

	// Status flags
	statuses := make([]string, 0)
	if info.Verified {
		statuses = append(statuses, "已认证")
	}
	if info.Searchable {
		statuses = append(statuses, "允许搜索")
	}
	if len(statuses) > 0 {
		sb.WriteString(fmt.Sprintf("  状态:     %s\n", strings.Join(statuses, ", ")))
	}

	// Token (masked)
	sb.WriteString(fmt.Sprintf("  Token:    %s...\n", maskToken(info.Token)))

	fmt.Print(sb.String())
	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
