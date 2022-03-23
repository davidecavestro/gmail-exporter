package svc

import (
	"strings"
)

func concat(sep string, elems ...string) string {
	return strings.Join(elems, sep)
}
