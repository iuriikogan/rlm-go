package utils

import (
	"reflect"
	"testing"
)

func TestFindCodeBlocks(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "Single block",
			text: "Here is code:\n```repl\nprint('hi')\n```",
			want: []string{"print('hi')"},
		},
		{
			name: "Multiple blocks",
			text: "One:\n```repl\na=1\n```\nTwo:\n```repl\nb=2\n```",
			want: []string{"a=1", "b=2"},
		},
		{
			name: "No blocks",
			text: "Just text",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FindCodeBlocks(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindCodeBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindFinalAnswer(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "Simple final",
			text: "Final answer is FINAL(42)",
			want: "42",
		},
		{
			name: "Final with newline",
			text: "Result: FINAL(Done\nSuccess)",
			want: "Done\nSuccess",
		},
		{
			name: "No final",
			text: "Thinking...",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FindFinalAnswer(tt.text); got != tt.want {
				t.Errorf("FindFinalAnswer() = %v, want %v", got, tt.want)
			}
		})
	}
}
