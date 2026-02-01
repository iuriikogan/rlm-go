package observability

import (
	"testing"
)

func TestSetupLogger(t *testing.T) {
	logger := SetupLogger()
	if logger == nil {
		t.Fatal("Expected logger to be non-nil")
	}
}
