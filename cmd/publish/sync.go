package publish

import (
	"fmt"

	"github.com/mudongliang/weoa-cli/internal/auth"
	"github.com/mudongliang/weoa-cli/internal/client"
	"github.com/mudongliang/weoa-cli/internal/publish"
	"github.com/mudongliang/weoa-cli/internal/storage"
	"github.com/spf13/cobra"
)

type syncStore interface {
	HasAppMsgID(appMsgID int64) (bool, error)
	InsertArticle(a publish.Article) (bool, error)
}

// NewSyncCmd creates the "publish sync" subcommand.
func NewSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync published articles to local database",
		Long: `Syncs all published articles from the WeChat backend to the local SQLite database.

On first run, it fetches and stores all articles.
On subsequent runs, it only syncs new articles (incremental sync).`,
		RunE: runSync,
	}
}

func runSync(cmd *cobra.Command, args []string) error {
	session, err := auth.Load()
	if err != nil {
		return fmt.Errorf("not logged in — run 'weoa-cli auth login' first")
	}

	c, err := client.New(session)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	initialCount, err := store.ActiveArticleCount()
	if err != nil {
		return fmt.Errorf("count existing active articles: %w", err)
	}
	fullSync := initialCount == 0

	svc := publish.NewService(c)

	fmt.Println("Syncing...")

	newCount := 0
	detectedCount := 0
	deletedCount := 0
	var syncErr error
	seenThisRun := make(map[int64]bool)
	_, err = svc.ListAll(func(articles []publish.Article) bool {
		detectedCount += len(articles)
		deletedCount += countDeletedArticles(articles)

		stop, pageErr := syncArticlePage(store, articles, seenThisRun, func(a publish.Article) {
			newCount++
			fmt.Printf("+ %s\n", a.Title)
		})
		if pageErr != nil {
			syncErr = pageErr
		}
		return shouldStopSync(fullSync, stop)
	})
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if syncErr != nil {
		return syncErr
	}

	fmt.Println()

	total, err := store.ActiveArticleCount()
	if err != nil {
		return fmt.Errorf("count active articles: %w", err)
	}
	fmt.Printf("Done. Detected %d published records, %d deleted, %d new articles synced (total: %d).\n",
		detectedCount, deletedCount, newCount, total)

	return nil
}

func shouldStopSync(fullSync, pageStop bool) bool {
	return !fullSync && pageStop
}

func countDeletedArticles(articles []publish.Article) int {
	count := 0
	for _, a := range articles {
		if a.IsDeleted {
			count++
		}
	}
	return count
}

func syncArticlePage(store syncStore, articles []publish.Article, seenThisRun map[int64]bool, onInsert func(publish.Article)) (bool, error) {
	seenExisting := false
	for _, a := range articles {
		exists, checkErr := store.HasAppMsgID(a.AppMsgID)
		if checkErr != nil {
			return true, fmt.Errorf("check article %d: %w", a.AppMsgID, checkErr)
		}
		if exists {
			if seenThisRun == nil || !seenThisRun[a.AppMsgID] {
				seenExisting = true
			}
			continue
		}

		if a.IsDeleted {
			continue
		}

		inserted, insertErr := store.InsertArticle(a)
		if insertErr != nil {
			return true, fmt.Errorf("insert article %d: %w", a.AppMsgID, insertErr)
		}
		if inserted {
			if seenThisRun != nil {
				seenThisRun[a.AppMsgID] = true
			}
			if onInsert != nil {
				onInsert(a)
			}
		}
	}
	// Stop before the next page, but only after this page has been fully processed.
	return seenExisting, nil
}
