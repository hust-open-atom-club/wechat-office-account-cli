package publish

import (
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"
)

const (
	appmsgPublishPath = "/cgi-bin/appmsgpublish"
	pageSize          = 10
)

// PublishClient is the interface for the HTTP client that PublishService needs.
type PublishClient interface {
	GetWithParams(path string, params map[string]string) (*resty.Response, error)
}

// Service provides publish-related operations.
type Service struct {
	client PublishClient
}

// NewService creates a new PublishService.
func NewService(c PublishClient) *Service {
	return &Service{client: c}
}

// ListResult holds the result of a List call.
type ListResult struct {
	Articles    []Article `json:"articles"`
	TotalCount  int       `json:"total_count"`
	Begin       int       `json:"begin"`
	RecordCount int       `json:"-"` // Number of raw publish records, before article extraction.
}

// List fetches a page of published articles.
func (s *Service) List(begin, count int) (*ListResult, error) {
	params := map[string]string{
		"sub":   "list",
		"begin": strconv.Itoa(begin),
		"count": strconv.Itoa(count),
		"f":     "json",
	}

	resp, err := s.client.GetWithParams(appmsgPublishPath, params)
	if err != nil {
		return nil, fmt.Errorf("request appmsgpublish: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("appmsgpublish returned status %d: %s", resp.StatusCode(), resp.String())
	}

	page, articles, err := parseArticlePage(resp.Body())
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &ListResult{
		Articles:    articles,
		TotalCount:  page.TotalCount,
		RecordCount: len(page.PublishList),
		Begin:       begin,
	}, nil
}

// ListAll fetches all published articles by paginating until empty.
// It stops early if shouldStop returns true (used for incremental sync).
func (s *Service) ListAll(shouldStop func([]Article) bool) ([]Article, error) {
	var all []Article
	begin := 0

	for {
		result, err := s.List(begin, pageSize)
		if err != nil {
			return nil, fmt.Errorf("list page %d: %w", begin, err)
		}

		if result.RecordCount == 0 {
			break
		}

		all = append(all, result.Articles...)

		if shouldStop != nil && shouldStop(result.Articles) {
			break
		}

		begin += pageSize
	}

	return all, nil
}
