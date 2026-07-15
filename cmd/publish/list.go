package publish

import (
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
	listLimit  int
	listAll    bool
	listRemote bool
	listSearch string
)

// NewListCmd creates the "publish list" subcommand.
func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List published articles",
		Long: `Fetches and displays published articles.

By default, reads from the local database.
Use --remote to fetch from the WeChat backend.`,
		RunE: runList,
	}

	cmd.Flags().IntVarP(&listLimit, "limit", "n", 20, "Number of articles to show")
	cmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all published articles")
	cmd.Flags().BoolVar(&listRemote, "remote", false, "Fetch articles from the WeChat backend")
	cmd.Flags().StringVar(&listSearch, "search", "", "Search local articles by title, digest, or URL")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	if listRemote {
		if listSearch != "" {
			return fmt.Errorf("--search is only supported for local database queries")
		}
		return listFromAPI()
	}
	return listFromDatabase()
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

	result, err := svc.List(0, listLimit)
	if err != nil {
		return fmt.Errorf("list articles: %w", err)
	}
	articles := filterDeletedArticles(result.Articles)

	printTable(articles)
	fmt.Print(formatRemoteListSummary(len(result.Articles), countDeletedArticles(result.Articles), len(articles)))
	return nil
}

func listAllFromAPI(svc *publish.Service) error {
	fmt.Fprintf(os.Stderr, "Fetching all articles...")
	articles, err := svc.ListAll(nil)
	if err != nil {
		return fmt.Errorf("fetch all articles: %w", err)
	}
	detectedCount := len(articles)
	deletedCount := countDeletedArticles(articles)
	articles = filterDeletedArticles(articles)
	fmt.Fprintf(os.Stderr, " done.\n\n")

	printTable(articles)
	fmt.Print(formatRemoteListSummary(detectedCount, deletedCount, len(articles)))
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
		limit = 0
	}

	var articles []publish.Article
	if listSearch != "" {
		articles, err = store.SearchActiveArticles(listSearch, limit)
	} else {
		articles, err = store.ListActiveArticles(limit)
	}
	if err != nil {
		return fmt.Errorf("query database: %w", err)
	}

	printTable(articles)
	if listSearch != "" {
		total, countErr := store.SearchActiveArticleCount(listSearch)
		if countErr != nil {
			return fmt.Errorf("count search results: %w", countErr)
		}
		if !listAll && limit > 0 && len(articles) < total {
			fmt.Printf("\nShowing %d of %d matched articles for %q\n", len(articles), total, listSearch)
		} else {
			fmt.Printf("\nMatched %d articles for %q\n", total, listSearch)
		}
		return nil
	}
	total, _ := store.ActiveArticleCount()
	if !listAll && limit > 0 && len(articles) < total {
		fmt.Printf("\nShowing %d of %d articles (use --all for full list)\n", len(articles), total)
	} else {
		fmt.Printf("\nTotal: %d articles\n", len(articles))
	}
	return nil
}

func filterDeletedArticles(articles []publish.Article) []publish.Article {
	active := articles[:0]
	seen := make(map[int64]bool)
	for _, a := range articles {
		if a.IsDeleted || seen[a.AppMsgID] {
			continue
		}
		seen[a.AppMsgID] = true
		active = append(active, a)
	}
	return active
}

func printTable(articles []publish.Article) {
	fmt.Printf("%-12s  %s\n", "TIME", "TITLE")
	fmt.Println("──────────────────────────────────────────────────")

	for _, a := range articles {
		t := formatPublishDate(a.PublishTime)
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

func formatPublishDate(publishTime int64) string {
	if publishTime <= 0 {
		return "-"
	}
	return time.Unix(publishTime, 0).Format("2006-01-02")
}

func formatRemoteListSummary(detectedCount, deletedCount, shownCount int) string {
	return fmt.Sprintf("\nDone. Detected %d published records, %d deleted, %d articles listed.\n",
		detectedCount, deletedCount, shownCount)
}
