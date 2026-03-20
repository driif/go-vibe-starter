package tests_test

import (
	"testing"

	"github.com/driif/go-vibe-starter/pkg/tests"
)

func TestRunningInTest(t *testing.T) {
	if !tests.RunningInTest() {
		t.Fatal("expected RunningInTest to be true during go test")
	}
}
