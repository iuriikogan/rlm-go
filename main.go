package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/iuriikogan/rlm-go/pkg/client"
	"github.com/iuriikogan/rlm-go/pkg/rlm"
)

type completionRequest struct {
	Prompt  string      `json:"prompt"`
	Context interface{} `json:"context,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	modelName := os.Getenv("GEMINI_MODEL_NAME")

	geminiClient, err := client.NewGeminiClient(apiKey, modelName)
	if err != nil {
		log.Fatalf("failed to create gemini client: %v", err)
	}

	rlmEngine := rlm.NewRLM(geminiClient, 10)

	http.HandleFunc("/completion", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req completionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		resp, err := rlmEngine.Completion(ctx, req.Prompt, req.Context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
