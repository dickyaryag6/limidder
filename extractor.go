package limidder

import (
	"fmt"
	"net/http"
	"strings"
)

type Extractor interface {
	ExtractKey(r *http.Request, applyUserRateLimitToAllPaths bool) (string, error)
}

type httpHeaderExtractor struct {
	headers []string
	fn      func([]string) (string, error)
}

func NewHTTPHeadersExtractor(fn func([]string) (string, error), headers ...string) Extractor {
	return &httpHeaderExtractor{
		headers: headers,
		fn:      fn,
	}
}

func (h *httpHeaderExtractor) ExtractKey(r *http.Request, applyUserRateLimitToAllPaths bool) (string, error) {

	values := make([]string, 0, len(h.headers))

	// if we can't find a value for the headers, give up and return an error.
	var path string
	if !applyUserRateLimitToAllPaths {
		path = fmt.Sprintf(":%s:%s", r.Method, r.URL.String())
	}

	if h.fn != nil {
		keyString, err := h.fn(h.headers)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s", keyString, path), nil
	}

	for _, key := range h.headers {
		if value := strings.TrimSpace(r.Header.Get(key)); value == "" {
			return "", fmt.Errorf("the header %v must have a value set", key)
		} else {
			values = append(values, value)
		}
	}

	return fmt.Sprintf("%s%s", strings.Join(values, "-"), path), nil
}
