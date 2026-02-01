package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE2E_Server(t *testing.T) {
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	// Build the server
	tempDir := t.TempDir()
	serverBin := filepath.Join(tempDir, "server")

	// Assuming we are in the root of the repo
	cmd := exec.Command("go", "build", "-o", serverBin, "../cmd/server/main.go")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build server: %v\n%s", err, out)
	}

	// Start the server
	port := "8081" // Use a different port
	serverCmd := exec.Command(serverBin)
	serverCmd.Env = append(os.Environ(), "PORT="+port)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer serverCmd.Process.Kill()

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

		// Make a request
		reqBody := []byte(`{"prompt": "Calculate 1+1", "max_iterations": 2}`)
		resp, err := http.Post(fmt.Sprintf("http://localhost:%s/completion", port), "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()
	
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}
	
		if resp.StatusCode != http.StatusOK {
					// Check if it's a quota error
					if errMsg, ok := result["error"].(string); ok {
						if resp.StatusCode == http.StatusInternalServerError && (strings.Contains(errMsg, "Quota exceeded") || strings.Contains(errMsg, "429")) {
							t.Skipf("Quota exceeded, skipping test: %s", errMsg)
						}
					}
					t.Errorf("Expected status 200, got %d. Response: %v", resp.StatusCode, result)
				}
			}
