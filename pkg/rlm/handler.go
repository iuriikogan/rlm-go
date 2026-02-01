package rlm

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"

	"github.com/user/rlm-go/pkg/types"
)

type LMHandler struct {
	client      interface {
		Completion(ctx context.Context, messages []types.Message) (string, error)
		ModelName() string
	}
	repl interface {
		AddPendingCall(call types.RLMChatCompletion)
	}
	server *http.Server
	addr   string
}

func NewLMHandler(client interface {
	Completion(ctx context.Context, messages []types.Message) (string, error)
	ModelName() string
}, repl interface {
	AddPendingCall(call types.RLMChatCompletion)
}) *LMHandler {
	return &LMHandler{
		client: client,
		repl:   repl,
	}
}

func (h *LMHandler) Start() (string, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/query", h.handleQuery)
	mux.HandleFunc("/query_batched", h.handleQueryBatched)

	// Listen on a random port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	h.addr = ln.Addr().String()
	h.server = &http.Server{Handler: mux}

	go h.server.Serve(ln)

	return h.addr, nil
}

func (h *LMHandler) Stop() {
	if h.server != nil {
		h.server.Shutdown(context.Background())
	}
}

type queryRequest struct {
	Prompt string  `json:"prompt"`
	Model  *string `json:"model"`
}

type queryResponse struct {
	Response string `json:"response"`
}

func (h *LMHandler) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	messages := []types.Message{{Role: "user", Content: req.Prompt}}
	resp, err := h.client.Completion(r.Context(), messages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.repl.AddPendingCall(types.RLMChatCompletion{
		RootModel: h.client.ModelName(),
		Prompt:    req.Prompt,
		Response:  resp,
	})

	json.NewEncoder(w).Encode(queryResponse{Response: resp})
}

type batchedQueryRequest struct {
	Prompts []string `json:"prompts"`
	Model   *string  `json:"model"`
}

type batchedQueryResponse struct {
	Responses []string `json:"responses"`
}

func (h *LMHandler) handleQueryBatched(w http.ResponseWriter, r *http.Request) {
	var req batchedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	responses := make([]string, len(req.Prompts))

	for i, prompt := range req.Prompts {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			messages := []types.Message{{Role: "user", Content: p}}
			resp, err := h.client.Completion(r.Context(), messages)
			if err != nil {
				return
			}
			responses[i] = resp
			h.repl.AddPendingCall(types.RLMChatCompletion{
				RootModel: h.client.ModelName(),
				Prompt:    p,
				Response:  resp,
			})
		}(i, prompt)
	}
	wg.Wait()

	json.NewEncoder(w).Encode(batchedQueryResponse{Responses: responses})
}
