package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	h := testHandler(t, func(d *Deps) { d.Metrics = reg })
	for _, tt := range []struct{ path, token string }{
		{"/api.v1.WhoAmIService/WhoAmI", "valid"},
		{"/api.v1.WhoAmIService/WhoAmI", ""},
		{"/api.v1.WhoAmIService/WhoAmI", "down"},
		{"/api.v1.WhoAmIService/CallerChosen", "valid"},
		{"/test/plain", "valid"},
	} {
		r := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		if tt.token != "" {
			r.Header.Set("Authorization", "Bearer "+tt.token)
		}
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Len(t, families[0].Metric, 3)
	seen := map[string]string{}
	for _, metric := range families[0].Metric {
		require.EqualValues(t, 1, metric.GetHistogram().GetSampleCount())
		labels := map[string]string{}
		for _, label := range metric.Label {
			labels[label.GetName()] = label.GetValue()
		}
		seen[labels["code"]] = labels["procedure"]
	}
	require.Equal(t, map[string]string{"ok": "/api.v1.WhoAmIService/WhoAmI", "unauthenticated": "", "unavailable": ""}, seen)
}
