package tui

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/bidi"
)

func visualText(line string) string {
	if !hasRTLText(line) {
		return line
	}

	var paragraph bidi.Paragraph
	if _, err := paragraph.SetString(line); err != nil {
		return line
	}
	ordering, err := paragraph.Order()
	if err != nil {
		return line
	}

	var out strings.Builder
	for i := 0; i < ordering.NumRuns(); i++ {
		run := ordering.Run(i)
		text := run.String()
		if run.Direction() == bidi.RightToLeft {
			text = bidi.ReverseString(text)
		}
		out.WriteString(text)
	}
	return out.String()
}

func hasRTLText(value string) bool {
	for _, r := range value {
		if unicode.In(r, unicode.Arabic, unicode.Hebrew) {
			return true
		}
	}
	return false
}
