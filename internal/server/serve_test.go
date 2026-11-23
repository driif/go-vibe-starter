package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	srv := New(":8080", http.NotFoundHandler())
	require.True(t, srv.Protocols.HTTP1())
	require.True(t, srv.Protocols.UnencryptedHTTP2())
	require.Equal(t, 10*time.Second, srv.ReadHeaderTimeout)
	require.Equal(t, 120*time.Second, srv.IdleTimeout)
	require.Zero(t, srv.ReadTimeout)
	require.Zero(t, srv.WriteTimeout)
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })
	return l
}

func TestServe(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	app := New("", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(204)
	}))
	management := New("", Management(prometheus.NewRegistry()))
	appListener, managementListener := listen(t), listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- Serve(ctx, time.Second, map[*http.Server]net.Listener{app: appListener, management: managementListener})
	}()
	for _, path := range []string{"/metrics", "/debug/pprof/"} {
		res, err := http.Get("http://" + managementListener.Addr().String() + path)
		require.NoError(t, err)
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		require.Equal(t, 200, res.StatusCode)
	}
	response := make(chan *http.Response, 1)
	requestErr := make(chan error, 1)
	go func() {
		res, err := http.Get("http://" + appListener.Addr().String())
		response <- res
		requestErr <- err
	}()
	<-started
	cancel()
	select {
	case err := <-result:
		t.Fatalf("shutdown skipped active request: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-requestErr)
	res := <-response
	res.Body.Close()
	require.Equal(t, 204, res.StatusCode)
	require.NoError(t, <-result)
}

func TestServe_GraceExceeded(t *testing.T) {
	started := make(chan struct{})
	srv := New("", http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	listener := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- Serve(ctx, 20*time.Millisecond, map[*http.Server]net.Listener{srv: listener}) }()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		res, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			res.Body.Close()
		}
	}()
	<-started
	cancel()
	require.ErrorIs(t, <-result, context.DeadlineExceeded)
	<-requestDone
}

func TestServe_StopsAllWhenOneFails(t *testing.T) {
	bad, good := listen(t), listen(t)
	require.NoError(t, bad.Close())
	err := Serve(t.Context(), time.Second, map[*http.Server]net.Listener{New("", http.NotFoundHandler()): bad, New("", http.NotFoundHandler()): good})
	require.Error(t, err)
	_, err = net.DialTimeout("tcp", good.Addr().String(), 50*time.Millisecond)
	require.Error(t, err)
}
