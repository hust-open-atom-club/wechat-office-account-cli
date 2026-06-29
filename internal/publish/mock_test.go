package publish

import (
	"github.com/go-resty/resty/v2"
)

// mockClient implements PublishClient for testing.
type mockClient struct {
	baseURL string
}

func (m *mockClient) GetWithParams(path string, params map[string]string) (*resty.Response, error) {
	r := resty.New().SetBaseURL(m.baseURL).R()
	for k, v := range params {
		r.SetQueryParam(k, v)
	}
	return r.Get(path)
}

// Compile-time check that mockClient satisfies PublishClient
var _ PublishClient = (*mockClient)(nil)
