//go:build go1.14
// +build go1.14

package gear

import "net/http"

func getHeaderValues(h http.Header, key string) []string {
	return h.Values(key)
}
