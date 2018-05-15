package xgraph

import (
	"strings"
)

// MultiError is a type containing multiple errors
type MultiError []error

func (me MultiError) Error() string {
	strs := make([]string, len(me))
	for i, err := range me {
		strs[i] = err.Error()
	}
	return strings.Join(strs, "\n")
}

// dedup takes a sorted string slice with duplicates and returns a slice without duplicates
func dedup(strs []string) []string {
	for i := 1; i < len(strs); i++ {
		if strs[i-1] == strs[i] {
			strs = append(strs[:i-1], strs[i:]...)
			i--
		}
	}
	return strs
}
