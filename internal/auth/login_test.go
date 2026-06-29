package auth

import "testing"

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
	}{
		{
			name: "standard home URL",
			url:  "https://mp.weixin.qq.com/cgi-bin/home?t=home/index&lang=zh_CN&token=123456789",
			want: "123456789",
		},
		{
			name: "token at end",
			url:  "https://mp.weixin.qq.com/cgi-bin/home?token=abc123def",
			want: "abc123def",
		},
		{
			name: "token with ampersand after",
			url:  "https://mp.weixin.qq.com/cgi-bin/home?token=xyz789&lang=zh_CN&t=home/index",
			want: "xyz789",
		},
		{
			name: "no token",
			url:  "https://mp.weixin.qq.com/cgi-bin/home?t=home/index",
			want: "",
		},
		{
			name: "empty string",
			url:  "",
			want: "",
		},
		{
			name: "token with special chars",
			url:  "https://mp.weixin.qq.com/cgi-bin/home?token=a1b2C3d4E5_f6g7h8&lang=zh_CN",
			want: "a1b2C3d4E5_f6g7h8",
		},
		{
			name: "realistic WeChat token",
			url:  "https://mp.weixin.qq.com/cgi-bin/home?t=home/index&lang=zh_CN&token=2147483647",
			want: "2147483647",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToken(tt.url)
			if got != tt.want {
				t.Errorf("extractToken(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
