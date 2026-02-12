package main

import (
	"bytes"
	"encoding/json"
	"strings"
)

func tryFormatJSON(s string) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(s), "", "  "); err != nil {
		return s
	}
	return prettyJSON.String()
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}
