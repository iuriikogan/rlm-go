package types

import (
	"encoding/json"
	"testing"
)

func TestMessageJSON(t *testing.T) {
	msg := Message{Role: "user", Content: "hello"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal Message: %v", err)
	}
	
	var msg2 Message
	if err := json.Unmarshal(data, &msg2); err != nil {
		t.Fatalf("Failed to unmarshal Message: %v", err)
	}
	
	if msg2.Role != msg.Role || msg2.Content != msg.Content {
		t.Errorf("Mismatch after marshal/unmarshal: %+v vs %+v", msg, msg2)
	}
}
