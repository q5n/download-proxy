package proxy

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
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
	log.Printf("download request method=%s path=%s remote=%s", r.Method, r.URL.RequestURI(), r.RemoteAddr)

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		log.Printf("rejecting unsupported method method=%s", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	target := q.Get("url")
	timestampStr := q.Get("time")
	sign := q.Get("sign")

	if target == "" || timestampStr == "" || sign == "" {
		log.Printf("missing request parameters target=%q time=%q sign=%q", target, timestampStr, sign != "")
		http.Error(w, "missing parameter", http.StatusBadRequest)
		return
	}

	var timestamp int64
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		log.Printf("invalid time parameter target=%s time=%s err=%v", target, timestampStr, err)
		http.Error(w, "invalid time", http.StatusBadRequest)
		return
	}

	if !security.Verify(target, timestamp, sign, p.Config.Secret, p.Config.MaxExpireSeconds) {
		log.Printf("signature verification failed target=%s timestamp=%d", target, timestamp)
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}
	if !p.allowed(target) {
		log.Printf("blocked target domain target=%s", target)
		http.Error(w, "domain blocked", http.StatusForbidden)
		return
	}

	// Build the upstream request and forward the client headers.
	req, err := http.NewRequest(r.Method, target, nil)
	if err != nil {
		log.Printf("failed to create upstream request target=%s err=%v", target, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()

	log.Printf("proxying upstream target=%s", target)
	resp, err := p.Client.Do(req)
	if err != nil {
		log.Printf("upstream request failed target=%s err=%v", target, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	log.Printf("upstream response received target=%s status=%d", target, resp.StatusCode)
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	// Stream the response body directly to the client.
	_, copyErr := io.Copy(w, resp.Body)
	if copyErr != nil {
		log.Printf("stream copy error target=%s err=%v", target, copyErr)
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
