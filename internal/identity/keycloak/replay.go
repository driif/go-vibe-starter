package keycloak

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

type replay struct {
	base    http.RoundTripper
	ttl     time.Duration
	mu      sync.Mutex
	cache   map[string]replayed
	keysURL string
}

type replayed struct {
	at       time.Time
	response *http.Response
	body     []byte
	err      error
}

func (r *replay) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return r.base.RoundTrip(req)
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.cache[req.URL.String()]
	if !ok || time.Since(entry.at) >= r.ttl {
		entry = replayed{}
		entry.response, entry.err = r.base.RoundTrip(req)
		if entry.err == nil {
			entry.body, entry.err = io.ReadAll(io.LimitReader(entry.response.Body, (1<<20)+1))
			closeErr := entry.response.Body.Close()
			if entry.err == nil {
				entry.err = closeErr
			}
			if len(entry.body) > 1<<20 {
				entry.err = errors.New("identity provider response exceeds 1 MiB")
			}
		}
		if entry.err == nil && req.URL.String() == r.keysURL && entry.response.StatusCode == http.StatusOK {
			entry.err = checkSigningKeys(entry.body)
		}
		entry.at = time.Now()
		if r.cache == nil {
			r.cache = make(map[string]replayed)
		}
		r.cache[req.URL.String()] = entry
	}
	if entry.err != nil {
		return nil, entry.err
	}
	response := *entry.response
	response.Header = entry.response.Header.Clone()
	response.Body = io.NopCloser(bytes.NewReader(entry.body))
	response.Request = req
	return &response, nil
}
