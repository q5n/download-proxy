package proxy

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/q5n/download-proxy/internal/config"
	"github.com/q5n/download-proxy/internal/security"
)

// Proxy wraps the configuration and HTTP client used to forward signed download requests.
type Proxy struct {
	Config *config.Config
	Client *http.Client
}

// New creates a new proxy instance with redirect protection enabled.
func New(cfg *config.Config) *Proxy {
	p := &Proxy{Config: cfg}
	p.Client = &http.Client{
		// Limit redirect hops and block redirects to disallowed hosts.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			if !p.allowedURL(req.URL) {
				return errors.New("redirect target blocked")
			}
			return nil
		},
	}
	return p
}

// Handler serves the download endpoint, validates the signature, and streams the upstream response.
func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	target := q.Get("url")
	expireStr := q.Get("expire")
	sign := q.Get("sign")

	if target == "" || expireStr == "" || sign == "" {
		http.Error(w, "missing parameter", http.StatusBadRequest)
		return
	}

	var expire int64
	_, err := fmt.Sscan(expireStr, &expire)
	if err != nil {
		http.Error(w, "invalid expire", http.StatusBadRequest)
		return
	}

	if !security.Verify(target, expire, sign, p.Config.Secret, p.Config.MaxExpireSeconds) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}
	if !p.allowed(target) {
		http.Error(w, "domain blocked", http.StatusForbidden)
		return
	}

	// Build the upstream request and forward the client headers.
	req, err := http.NewRequest(r.Method, target, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()

	resp, err := p.Client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	// Stream the response body directly to the client.
	_, copyErr := io.Copy(w, resp.Body)
	if copyErr != nil {
		log.Printf("stream copy error: %v", copyErr)
	}
}

// allowed checks whether a target URL is permitted by the configured domain allowlist.
func (p *Proxy) allowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return p.allowedURL(u)
}

// allowedURL validates a parsed URL against the configured allowed domains.
func (p *Proxy) allowedURL(u *url.URL) bool {
	host := u.Hostname()
	for _, domain := range p.Config.AllowedDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

// copyHeader copies response headers while filtering out hop-by-hop headers.
func copyHeader(dst http.Header, src http.Header) {

	for k, v := range src {
		switch strings.ToLower(k) {
		case "connection",
			"transfer-encoding",
			"keep-alive":
			continue
		}

		for _, vv := range v {
			dst.Add(k, vv)
		}

	}

}
