package publish

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	internalpublish "github.com/mudongliang/weoa-cli/internal/publish"
	"github.com/mudongliang/weoa-cli/internal/storage"
	"github.com/spf13/cobra"
)

var (
	exportLimit  int
	exportSearch string
)

type exportStore interface {
	ListActiveArticles(limit int) ([]internalpublish.Article, error)
	SearchActiveArticles(query string, limit int) ([]internalpublish.Article, error)
	Close() error
}

var newExportStore = func() (exportStore, error) {
	return storage.New()
}

// NewExportCmd creates the "publish export" subcommand.
func NewExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export published articles",
		Long: `Exports non-deleted published articles from the local database.

Run "weoa-cli publish sync" before exporting.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Help()
			return fmt.Errorf("export format required: json, csv, or markdown")
		},
	}

	cmd.PersistentFlags().IntVarP(&exportLimit, "limit", "n", 0, "Number of articles to export (0 means all)")
	cmd.PersistentFlags().StringVar(&exportSearch, "search", "", "Search local articles by title, digest, or URL before exporting")
	cmd.AddCommand(newExportJSONCmd())
	cmd.AddCommand(newExportCSVCmd())
	cmd.AddCommand(newExportMarkdownCmd())
	return cmd
}

func newExportJSONCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "json",
		Short: "Export articles as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			articles, err := loadExportArticles()
			if err != nil {
				return err
			}
			return printExportJSON(os.Stdout, articles)
		},
	}
}

func newExportCSVCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "csv",
		Short: "Export articles as CSV",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			articles, err := loadExportArticles()
			if err != nil {
				return err
			}
			return printExportCSV(os.Stdout, articles)
		},
	}
}

func newExportMarkdownCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "markdown",
		Aliases: []string{"md"},
		Short:   "Export articles as Markdown",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			articles, err := loadExportArticles()
			if err != nil {
				return err
			}
			printExportMarkdown(os.Stdout, articles)
			return nil
		},
	}
}

func loadExportArticles() ([]internalpublish.Article, error) {
	store, err := newExportStore()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	if exportSearch != "" {
		return store.SearchActiveArticles(exportSearch, exportLimit)
	}
	return store.ListActiveArticles(exportLimit)
}

type exportArticleJSON struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	PublishTime string `json:"publish_time"`
	Cover       string `json:"cover,omitempty"`
	Digest      string `json:"digest,omitempty"`
	ReadNum     int    `json:"read_num"`
	LikeNum     int    `json:"like_num"`
}

func printExportJSON(w io.Writer, articles []internalpublish.Article) error {
	output := make([]exportArticleJSON, 0, len(articles))
	for _, a := range articles {
		output = append(output, exportArticleJSON{
			Title:       a.Title,
			URL:         a.URL,
			PublishTime: formatPublishDate(a.PublishTime),
			Cover:       a.Cover,
			Digest:      a.Digest,
			ReadNum:     a.ReadNum,
			LikeNum:     a.LikeNum,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func printExportCSV(w io.Writer, articles []internalpublish.Article) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"publish_time", "title", "url", "digest", "cover", "read_num", "like_num"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}
	for _, a := range articles {
		if err := cw.Write([]string{
			formatPublishDate(a.PublishTime),
			a.Title,
			a.URL,
			a.Digest,
			a.Cover,
			strconv.Itoa(a.ReadNum),
			strconv.Itoa(a.LikeNum),
		}); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}
	return nil
}

func printExportMarkdown(w io.Writer, articles []internalpublish.Article) {
	fmt.Fprintln(w, "| Publish Time | Title | URL | Digest | Read | Like |")
	fmt.Fprintln(w, "|---|---|---|---|---:|---:|")
	for _, a := range articles {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %d | %d |\n",
			escapeMarkdownTable(formatPublishDate(a.PublishTime)),
			escapeMarkdownTable(a.Title),
			escapeMarkdownTable(a.URL),
			escapeMarkdownTable(a.Digest),
			a.ReadNum,
			a.LikeNum,
		)
	}
}

func escapeMarkdownTable(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
