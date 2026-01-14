package device

import (
	"os"
	"testing"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

// TestMain is the entry point for all tests in this package
func TestMain(m *testing.M) {
	// Initialize logger for tests
	if err := logging.InitLogger("info", false); err != nil {
		panic(err)
	}
	defer logging.Sync()

	os.Exit(m.Run())
}
