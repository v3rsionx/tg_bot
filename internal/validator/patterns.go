package validator

import "regexp"

var (
	botTokenPattern = regexp.MustCompile(`^\d{5,20}:[A-Za-z0-9_-]{20,100}$`)
	envKeyPattern   = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
)
