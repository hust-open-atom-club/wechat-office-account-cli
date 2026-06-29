package publish

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mudongliang/weoa-cli/internal/auth"
	"github.com/mudongliang/weoa-cli/internal/client"
	"github.com/mudongliang/weoa-cli/internal/publish"
	"github.com/mudongliang/weoa-cli/internal/storage"
	"github.com/spf13/cobra"
)

var (
	listLimit int
	listJSON  bool
	listAll   bool
	listFromDB bool
)

// NewListCmd creates the "publish list" subcommand.
func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List published articles",
		Long: `Fetches and displays published articles.

By default, shows the most recent page from the WeChat backend.
Use --all to fetch everything, or --from-db to list from the local database.`,
		RunE: runList,
	}

	cmd.Flags().IntVarP(&listLimit, "limit", "n", 20, "Number of articles per page (default 20)")
	cmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	cmd.Flags().BoolVarP(&listAll, "all", "a", false, "Fetch and show all published articles")
	cmd.Flags().BoolVar(&listFromDB, "from-db", false, "List articles from local database (run sync first)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	if listFromDB {
		return listFromDatabase()
	}
	return listFromAPI()
}

func listFromAPI() error {
	session, err := auth.Load()
	if err != nil {
		return fmt.Errorf("not logged in — run 'weoa-cli auth login' first")
	}

	c, err := client.New(session)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	svc := publish.NewService(c)

	if listAll {
		return listAllFromAPI(svc)
	}

	// Default: single page
	result, err := svc.List(0, listLimit)
	if err != nil {
		return fmt.Errorf("list articles: %w", err)
	}

	if listJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result.Articles)
	}

	printTable(result.Articles)
	fmt.Printf("\nShowing %d of %d articles (use --all to fetch all, --from-db to list local)\n", len(result.Articles), result.TotalCount)
	return nil
}

func listAllFromAPI(svc *publish.Service) error {
	fmt.Fprintf(os.Stderr, "Fetching all articles...")
	articles, err := svc.ListAll(nil)
	if err != nil {
		return fmt.Errorf("fetch all articles: %w", err)
	}
	fmt.Fprintf(os.Stderr, " done.\n\n")

	if listJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(articles)
	}

	printTable(articles)
	fmt.Printf("\nTotal: %d articles\n", len(articles))
	return nil
}

func listFromDatabase() error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	limit := listLimit
	if listAll {
		limit = 0 // 0 means no limit in ListArticles
	}

	articles, err := store.ListArticles(limit)
	if err != nil {
		return fmt.Errorf("query database: %w", err)
	}

	if listJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(articles)
	}

	printTable(articles)
	total, _ := store.ArticleCount()
	if !listAll && limit > 0 && len(articles) < total {
		fmt.Printf("\nShowing %d of %d articles (use --all for full list)\n", len(articles), total)
	} else {
		fmt.Printf("\nTotal: %d articles\n", len(articles))
	}
	return nil
}

func printTable(articles []publish.Article) {
	fmt.Printf("%-12s  %s\n", "TIME", "TITLE")
	fmt.Println("──────────────────────────────────────────────────")

	for _, a := range articles {
		t := time.Unix(a.PublishTime, 0).Format("2006-01-02")
		status := ""
		if a.IsDeleted {
			status = " [DELETED]"
		}
		fmt.Printf("%-12s  %s%s\n", t, a.Title, status)
		if a.URL != "" {
			fmt.Printf("%-12s  %s\n", "", a.URL)
		}
	}
}
