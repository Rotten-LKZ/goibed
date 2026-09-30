package utils

import (
	"fmt"
	"net/http"
	"net/url"
)

func GetResourceURL(r *http.Request, customPath string) (string, error) {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}

	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	basePath := r.URL.Path

	baseURLString := fmt.Sprintf("%s://%s%s", scheme, host, basePath)
	baseURL, err := url.Parse(baseURLString)
	if err != nil {
		return "", fmt.Errorf("failed to parse base url: %w", err)
	}

	customURL, err := url.Parse(customPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse custom path: %w", err)
	}

	finalURL := baseURL.ResolveReference(customURL)

	return finalURL.String(), nil
}
