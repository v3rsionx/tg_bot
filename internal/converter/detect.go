package converter

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

var candidateDelimiters = []rune{',', ';', '|', '\t'}

// detectDelimiter chooses the most consistent delimiter from a text sample.
func detectDelimiter(sample string) (rune, DelimiterName) {
	best := ','
	bestName := DelimiterComma
	bestScore := -1.0

	lines := splitSampleLines(sample, 20)
	if len(lines) == 0 {
		return best, bestName
	}

	for _, d := range candidateDelimiters {
		score := scoreDelimiter(lines, d)
		if score > bestScore {
			bestScore = score
			best = d
			bestName = delimiterName(d)
		}
	}
	return best, bestName
}

func delimiterName(d rune) DelimiterName {
	switch d {
	case ';':
		return DelimiterSemicolon
	case '|':
		return DelimiterPipe
	case '\t':
		return DelimiterTab
	default:
		return DelimiterComma
	}
}

func splitSampleLines(sample string, max int) []string {
	raw := strings.Split(sample, "\n")
	out := make([]string, 0, max)
	for _, line := range raw {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
		if len(out) >= max {
			break
		}
	}
	return out
}

func scoreDelimiter(lines []string, d rune) float64 {
	counts := make([]int, 0, len(lines))
	for _, line := range lines {
		counts = append(counts, strings.Count(line, string(d)))
	}
	if len(counts) == 0 {
		return -1
	}
	// Prefer delimiters that appear often and consistently.
	mode := counts[0]
	modeFreq := 0
	freq := map[int]int{}
	sum := 0
	for _, c := range counts {
		sum += c
		freq[c]++
		if freq[c] > modeFreq {
			modeFreq = freq[c]
			mode = c
		}
	}
	if mode == 0 {
		return -1
	}
	consistency := float64(modeFreq) / float64(len(counts))
	avg := float64(sum) / float64(len(counts))
	return consistency*10 + avg
}

// looksLikeHeader reports whether the first row is likely a header.
func looksLikeHeader(fields []string) bool {
	if len(fields) == 0 {
		return false
	}
	known := 0
	for _, f := range fields {
		if _, ok := classifyHeader(f); ok {
			known++
		}
	}
	if known >= 1 {
		return true
	}
	// Digits-only first field suggests data, not header.
	if len(fields) > 0 && isAllDigits(fields[0]) {
		return false
	}
	letters := 0
	for _, f := range fields {
		if containsLetter(f) {
			letters++
		}
	}
	return letters >= (len(fields)+1)/2
}

func isAllDigits(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func containsLetter(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > utf8.RuneSelf {
			return true
		}
	}
	return false
}

// parseCSVLine splits a single line with a simple CSV-aware parser (quoted fields).
func parseCSVLine(line string, delim rune) []string {
	var fields []string
	var cur bytes.Buffer
	inQuotes := false
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if inQuotes {
			if r == '"' {
				if i+1 < len(runes) && runes[i+1] == '"' {
					cur.WriteRune('"')
					i++
					continue
				}
				inQuotes = false
				continue
			}
			cur.WriteRune(r)
			continue
		}
		switch r {
		case '"':
			inQuotes = true
		case delim:
			fields = append(fields, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	fields = append(fields, cur.String())
	return fields
}
