package env

import (

	"bufio"

	"context"

	"encoding/json"

	"fmt"

		"io"

		"os"

		"os/exec"

	"sync"

	"time"



	"github.com/user/rlm-go/pkg/types"

)



type Environment interface {

	ExecuteCode(ctx context.Context, code string) types.REPLResult

	Cleanup()

}



type PythonREPL struct {

	cmd           *exec.Cmd

	stdin         io.WriteCloser

	stdout        *bufio.Reader

	handlerAddr   string

	pendingCalls  []types.RLMChatCompletion

	mu            sync.Mutex

	tempDir       string

}

func NewPythonREPL(ctx context.Context, handlerAddr string) (*PythonREPL, error) {
	tempDir, err := os.MkdirTemp("", "rlm-repl-*")
	if err != nil {
		return nil, err
	}

	// This is a minimal Python wrapper to maintain state and handle llm_query
	wrapper := `
import sys
import json
import urllib.request

_globals = {}
_locals = {}

def llm_query(prompt, model=None):
    data = json.dumps({"prompt": prompt, "model": model}).encode("utf-8")
    req = urllib.request.Request(f"http://%s/query", data=data, method="POST")
    with urllib.request.urlopen(req) as f:
        resp = json.loads(f.read().decode("utf-8"))
        return resp["response"]

def llm_query_batched(prompts, model=None):
    data = json.dumps({"prompts": prompts, "model": model}).encode("utf-8")
    req = urllib.request.Request(f"http://%s/query_batched", data=data, method="POST")
    with urllib.request.urlopen(req) as f:
        resp = json.loads(f.read().decode("utf-8"))
        return resp["responses"]

_globals["llm_query"] = llm_query
_globals["llm_query_batched"] = llm_query_batched
_globals["print"] = print

while True:
    line = sys.stdin.readline()
    if not line:
        break
    try:
        code_info = json.loads(line)
        code = code_info["code"]
        
        from io import StringIO
        new_stdout = StringIO()
        new_stderr = StringIO()
        old_stdout = sys.stdout
        old_stderr = sys.stderr
        sys.stdout = new_stdout
        sys.stderr = new_stderr
        
        try:
            exec(code, _globals, _locals)
            stdout = new_stdout.getvalue()
            stderr = new_stderr.getvalue()
        except Exception as e:
            stdout = new_stdout.getvalue()
            stderr = new_stderr.getvalue() + str(e)
        finally:
            sys.stdout = old_stdout
            sys.stderr = old_stderr
            
        print(json.dumps({"stdout": stdout, "stderr": stderr, "done": True}))
        sys.stdout.flush()
    except Exception as e:
        print(json.dumps({"error": str(e), "done": True}))
        sys.stdout.flush()
`
	wrapper = fmt.Sprintf(wrapper, handlerAddr, handlerAddr)
	wrapperPath := tempDir + "/wrapper.py"
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0644); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "python3", wrapperPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	        return &PythonREPL{
	                cmd:         cmd,
	                stdin:       stdin,
	                stdout:      bufio.NewReader(stdoutPipe),
	                handlerAddr: handlerAddr,
	                tempDir:     tempDir,
	        }, nil
	}
type replResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Error  string `json:"error"`
	Done   bool   `json:"done"`
}

func (r *PythonREPL) ExecuteCode(ctx context.Context, code string) types.REPLResult {
	start := time.Now()
	
	r.mu.Lock()
	r.pendingCalls = nil // Clear pending calls
	r.mu.Unlock()

	req := map[string]string{"code": code}
	reqData, _ := json.Marshal(req)
	_, _ = r.stdin.Write(append(reqData, '\n'))

	line, err := r.stdout.ReadString('\n')
	if err != nil {
		return types.REPLResult{Stderr: "Failed to read from REPL: " + err.Error()}
	}

	var resp replResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return types.REPLResult{Stderr: "Failed to parse REPL response: " + err.Error()}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	
	return types.REPLResult{
		Stdout:        resp.Stdout,
		Stderr:        resp.Stderr + resp.Error,
		ExecutionTime: time.Since(start).Seconds(),
		RLMCalls:      r.pendingCalls,
	}
}

func (r *PythonREPL) AddPendingCall(call types.RLMChatCompletion) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pendingCalls = append(r.pendingCalls, call)
}
func (r *PythonREPL) Cleanup() {
	if r.stdin != nil {
		r.stdin.Close()
	}
	if r.cmd != nil {
		r.cmd.Process.Kill()
	}
	os.RemoveAll(r.tempDir)
}
