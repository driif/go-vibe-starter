package test

import (
	"context"
	"testing"

	"github.com/driif/go-vibe-starter/internal/api/router"
	"github.com/driif/go-vibe-starter/internal/server"
	"github.com/driif/go-vibe-starter/internal/server/config"
)

func E2e(t *testing.T, closure func(s *server.Server)) {
	// Code here
	t.Helper()
	conf := config.DefaultServiceConfigFromEnv()
	conf.Server.ListenAddr = ":0"

	testDB := NewDBInstance(t, conf)
	testDB.ApplyFixtures(t)

	s := server.NewWithConfig(conf)
	s.DB = testDB.DB

	if err := s.Initialize(); err != nil {
		t.Fatalf("failed to initialize server: %v", err)
	}
	router.RegisterHandlersV1(s)

	closure(s)

	// echo is managed and should close automatically after running the test
	if err := s.Shutdown(context.TODO()); err != nil {
		t.Fatalf("failed to shutdown server: %v", err)
	}

	testDB.Close()
	s = nil
}
