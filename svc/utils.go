package svc

import (
	"strings"
)

func concat(sep string, elems ...string) string {
	return strings.Join(elems, sep)
}

func RemoveNils[C any | string](s []*C) []*C {
	i := 0 // output index
	for _, x := range s {
		if x != nil {
			// copy and increment index
			s[i] = x
			i++
		}
	}
	// Prevent memory leak by erasing truncated values
	// (not needed if values don't contain pointers, directly or indirectly)
	for j := i; j < len(s); j++ {
		s[j] = nil
	}
	ret := s[:i]
	return ret
}
