package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/q5n/download-proxy/internal/config"
	"github.com/q5n/download-proxy/internal/security"
)

const testSecret = "test-secret"

func startUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("hello proxy")); err != nil {
			t.Fatalf("upstream write failed: %v", err)
		}
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/file")
		w.WriteHeader(http.StatusFound)
		if _, err := w.Write([]byte("<a href=\"/file\">Found</a>.\n\n")); err != nil {
			t.Fatalf("upstream write failed: %v", err)
		}
	})
	mux.HandleFunc("/bad-redirect", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://evil.com/file")
		w.WriteHeader(http.StatusFound)
		if _, err := w.Write([]byte("<a href=\"https://evil.com/file\">Found</a>.\n\n")); err != nil {
			t.Fatalf("upstream write failed: %v", err)
		}
	})
	return httptest.NewServer(mux)
}

func startProxy(t *testing.T, upstream *httptest.Server) *httptest.Server {
	t.Helper()
	u, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream url: %v", err)
	}
	cfg := &config.Config{
		Secret:           testSecret,
		MaxExpireSeconds: 3600,
		AllowedDomains:   []string{u.Hostname()},
	}
	p := New(cfg)
	return httptest.NewServer(http.HandlerFunc(p.Handler))
}

func signURL(target string, timestamp int64) string {
	return security.Sign(target, timestamp, testSecret)
}

func buildProxyURL(proxy, target string, timestamp int64) string {
	return fmt.Sprintf("%s/download?url=%s&time=%d&sign=%s",
		proxy, url.QueryEscape(target), timestamp, signURL(target, timestamp))
}

func TestProxyEndToEnd(t *testing.T) {
	upstream := startUpstream(t)
	t.Cleanup(upstream.Close)

	proxy := startProxy(t, upstream)
	t.Cleanup(proxy.Close)

	tests := []struct {
		name            string
		makeURL         func() string
		method          string
		wantStatus      int
		wantBody        string
		wantContentType string
	}{
		{
			name: "normal download",
			makeURL: func() string {
				timestamp := time.Now().Unix()
				return buildProxyURL(proxy.URL, upstream.URL+"/file", timestamp)
			},
			method:          http.MethodGet,
			wantStatus:      http.StatusOK,
			wantBody:        "hello proxy",
			wantContentType: "application/octet-stream",
		},
		{
			name: "redirect to allowed host",
			makeURL: func() string {
				timestamp := time.Now().Add(time.Hour).Unix()
				return buildProxyURL(proxy.URL, upstream.URL+"/redirect", timestamp)
			},
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   "hello proxy",
		},
		{
			name: "redirect to disallowed host",
			makeURL: func() string {
				timestamp := time.Now().Add(time.Hour).Unix()
				return buildProxyURL(proxy.URL, upstream.URL+"/bad-redirect", timestamp)
			},
			method:     http.MethodGet,
			wantStatus: http.StatusBadGateway,
		},
		{
			name: "invalid signature",
			makeURL: func() string {
				timestamp := time.Now().Add(time.Hour).Unix()
				target := upstream.URL + "/file"
				return fmt.Sprintf("%s/download?url=%s&time=%d&sign=invalid",
					proxy.URL, url.QueryEscape(target), timestamp)
			},
			method:     http.MethodGet,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "expired timestamp",
			makeURL: func() string {
				timestamp := time.Now().Add(-2 * time.Hour).Unix()
				return buildProxyURL(proxy.URL, upstream.URL+"/file", timestamp)
			},
			method:     http.MethodGet,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "future timestamp allowed",
			makeURL: func() string {
				timestamp := time.Now().Add(time.Hour).Unix()
				return buildProxyURL(proxy.URL, upstream.URL+"/file", timestamp)
			},
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   "hello proxy",
		},
		{
			name: "blocked domain",
			makeURL: func() string {
				timestamp := time.Now().Add(time.Hour).Unix()
				return buildProxyURL(proxy.URL, "https://evil.com/file", timestamp)
			},
			method:     http.MethodGet,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "missing parameter",
			makeURL: func() string {
				return fmt.Sprintf("%s/download", proxy.URL)
			},
			method:     http.MethodGet,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "disallowed method",
			makeURL: func() string {
				timestamp := time.Now().Add(time.Hour).Unix()
				return buildProxyURL(proxy.URL, upstream.URL+"/file", timestamp)
			},
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.makeURL(), nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantContentType != "" {
				if got := resp.Header.Get("Content-Type"); got != tt.wantContentType {
					t.Fatalf("Content-Type = %q, want %q", got, tt.wantContentType)
				}
			}
			if tt.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				if string(body) != tt.wantBody {
					t.Fatalf("body = %q, want %q", string(body), tt.wantBody)
				}
			}
		})
	}
}
