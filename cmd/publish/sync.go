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

	svc := publish.NewService(c)

	fmt.Println("Syncing...")

	newCount := 0
	var syncErr error
	_, err = svc.ListAll(func(articles []publish.Article) bool {
		stop, pageErr := syncArticlePage(store, articles, func(a publish.Article) {
			newCount++
			fmt.Printf("+ %s\n", a.Title)
		})
		if pageErr != nil {
			syncErr = pageErr
		}
		return stop
	})
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if syncErr != nil {
		return syncErr
	}

	fmt.Println()

	total, err := store.ArticleCount()
	if err != nil {
		return fmt.Errorf("count articles: %w", err)
	}
	if newCount > 0 {
		fmt.Printf("Done. %d new articles synced (total: %d).\n", newCount, total)
	} else {
		fmt.Printf("No new articles. (%d articles in database)\n", total)
	}

	return nil
}

func syncArticlePage(store syncStore, articles []publish.Article, onInsert func(publish.Article)) (bool, error) {
	seenExisting := false
	for _, a := range articles {
		exists, checkErr := store.HasAppMsgID(a.AppMsgID)
		if checkErr != nil {
			return true, fmt.Errorf("check article %d: %w", a.AppMsgID, checkErr)
		}
		if exists {
			seenExisting = true
			continue
		}

		inserted, insertErr := store.InsertArticle(a)
		if insertErr != nil {
			return true, fmt.Errorf("insert article %d: %w", a.AppMsgID, insertErr)
		}
		if inserted {
			if onInsert != nil {
				onInsert(a)
			}
		}
	}
	// Stop before the next page, but only after this page has been fully processed.
	return seenExisting, nil
}
