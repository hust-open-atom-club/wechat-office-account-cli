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
	HasArticle(appMsgID int64, url string) (bool, error)
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
	stats, err := syncArticles(store, svc, func(a publish.Article) {
		fmt.Printf("+ %s\n", a.Title)
	})
	if err != nil {
		return err
	}

	total, err := store.ActiveArticleCount()
	if err != nil {
		return fmt.Errorf("count active articles: %w", err)
	}
	fmt.Printf("\nDone. Detected %d published records, %d deleted, %d new articles synced (total: %d).\n",
		stats.detected, stats.deleted, stats.inserted, total)
	return nil
}

type syncStateStore interface {
	syncStore
	ActiveArticleCount() (int, error)
	NeedsFullSync() (bool, error)
	MarkFullSyncNeeded() error
	MarkFullSyncComplete() error
}

type articleLister interface {
	ListAll(func([]publish.Article) bool) ([]publish.Article, error)
}

type syncStats struct {
	detected int
	deleted  int
	inserted int
}

func syncArticles(store syncStateStore, svc articleLister, onInsert func(publish.Article)) (syncStats, error) {
	var stats syncStats
	initialCount, err := store.ActiveArticleCount()
	if err != nil {
		return stats, fmt.Errorf("count existing active articles: %w", err)
	}
	needsFullSync, err := store.NeedsFullSync()
	if err != nil {
		return stats, fmt.Errorf("read sync state: %w", err)
	}
	fullSync := initialCount == 0 || needsFullSync
	// Even an incremental run can insert a partial page before failing. Persist
	// the marker before any writes so its retry traverses past cached articles.
	if err := store.MarkFullSyncNeeded(); err != nil {
		return stats, err
	}

	var syncErr error
	seenThisRun := make(map[articleIdentity]bool)
	_, err = svc.ListAll(func(articles []publish.Article) bool {
		stats.detected += len(articles)
		stats.deleted += countDeletedArticles(articles)
		stop, pageErr := syncArticlePage(store, articles, seenThisRun, func(a publish.Article) {
			stats.inserted++
			if onInsert != nil {
				onInsert(a)
			}
		})
		if pageErr != nil {
			syncErr = pageErr
			return true
		}
		return shouldStopSync(fullSync, stop)
	})
	if syncErr != nil {
		return stats, syncErr
	}
	if err != nil {
		return stats, fmt.Errorf("sync: %w", err)
	}
	if err := store.MarkFullSyncComplete(); err != nil {
		return stats, err
	}
	return stats, nil
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

func syncArticlePage(store syncStore, articles []publish.Article, seenThisRun map[articleIdentity]bool, onInsert func(publish.Article)) (bool, error) {
	seenExisting := false
	for _, a := range articles {
		identity := identityOf(a)
		exists, checkErr := store.HasArticle(a.AppMsgID, a.URL)
		if checkErr != nil {
			return true, fmt.Errorf("check article %d: %w", a.AppMsgID, checkErr)
		}
		if exists {
			if seenThisRun == nil || !seenThisRun[identity] {
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
				seenThisRun[identity] = true
			}
			if onInsert != nil {
				onInsert(a)
			}
		}
	}
	// Stop before the next page, but only after this page has been fully processed.
	return seenExisting, nil
}
