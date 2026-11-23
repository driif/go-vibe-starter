package keycloak

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReplay_RepeatsFailureThenRecovers(t *testing.T) {
	var calls int
	down := true
	r := &replay{
		ttl: time.Hour,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if down {
				return nil, errors.New("connection refused")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"keys":[]}`)), Request: req}, nil
		}),
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://issuer.test/discovery", nil)
	require.NoError(t, err)
	for range 20 {
		resp, err := r.RoundTrip(req)
		if resp != nil {
			resp.Body.Close()
		}
		require.Error(t, err)
	}
	assert.Equal(t, 1, calls)
	down = false
	r.cache[req.URL.String()] = replayed{at: time.Now().Add(-2 * time.Hour)}
	for range 2 {
		response, requestErr := r.RoundTrip(req)
		require.NoError(t, requestErr)
		body, readErr := io.ReadAll(response.Body)
		require.NoError(t, readErr)
		require.NoError(t, response.Body.Close())
		assert.Equal(t, `{"keys":[]}`, string(body))
		response.Header.Set("Content-Type", "changed")
	}
	assert.Equal(t, 2, calls)
	assert.Equal(t, "application/json", r.cache[req.URL.String()].response.Header.Get("Content-Type"))
}

func TestReplay_BodyLimit(t *testing.T) {
	r := &replay{ttl: time.Hour, base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", (1<<20)+1))), Request: req}, nil
	})}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://issuer.test/keys", nil)
	require.NoError(t, err)
	resp, err := r.RoundTrip(req)
	if resp != nil {
		resp.Body.Close()
	}
	require.ErrorContains(t, err, "exceeds 1 MiB")
}

func TestReplay_ConcurrentFetches(t *testing.T) {
	var calls int
	r := &replay{ttl: time.Hour, base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("keys")), Request: req}, nil
	})}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://issuer.test/keys", nil)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			response, err := r.RoundTrip(req)
			assert.NoError(t, err)
			if response != nil {
				assert.NoError(t, response.Body.Close())
			}
		})
	}
	wg.Wait()
	assert.Equal(t, 1, calls)
}

func TestReplayCachesSlowFailureAfterCompletion(t *testing.T) {
	var calls int
	r := &replay{ttl: 50 * time.Millisecond, base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		time.Sleep(100 * time.Millisecond)
		return nil, errors.New("provider unavailable")
	})}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://issuer.test/keys", nil)
	require.NoError(t, err)
	for range 2 {
		resp, err := r.RoundTrip(req)
		if resp != nil {
			resp.Body.Close()
		}
		require.Error(t, err)
	}
	require.Equal(t, 1, calls)
}
