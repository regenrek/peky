package app

import "strings"

var singleLineReplacer = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ")

func singleLine(text string) string {
	if text == "" {
		return ""
	}
	return strings.TrimSpace(singleLineReplacer.Replace(text))
}
