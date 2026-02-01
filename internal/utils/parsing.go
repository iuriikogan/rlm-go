package utils

import (
	"regexp"
	"strings"
)

var replRegex = regexp.MustCompile("(?s)```repl\n(.*?)\n```")
var finalRegex = regexp.MustCompile(`(?s)FINAL\((.*?)\)`)
var finalVarRegex = regexp.MustCompile(`(?s)FINAL_VAR\((.*?)\)`)

func FindCodeBlocks(text string) []string {
	matches := replRegex.FindAllStringSubmatch(text, -1)
	var blocks []string
	for _, m := range matches {
		blocks = append(blocks, m[1])
	}
	return blocks
}

func FindFinalAnswer(text string) string {
	// Check for FINAL(answer)
	matches := finalRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Check for FINAL_VAR(varname) - in this simplified version, we might just return the tag
	// or have the REPL handle it. For now let's just support FINAL().
	return ""
}
