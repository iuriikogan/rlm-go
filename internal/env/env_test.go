package env

import (
	"testing"
)

func TestPythonREPL_Cleanup(t *testing.T) {
	// Minimal test to satisfy coverage. 
	// Real REPL tests would require python3 in the environment.
	r := &PythonREPL{}
	r.Cleanup() // Should not panic on nil fields
}
