package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWMFResponseHandling(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		wantError  bool
	}{
		{"identified request", ")]}'\n{\"revision\":\"abcdef0123456789\"}", 200, false},
		{"HTTP error", "<!DOCTYPE html>blocked", 403, true},
		{"empty body", "", 200, true},
		{"missing prefix", "{\"revision\":\"abcdef0123456789\"}", 200, true},
		{"short revision", ")]}'\n{\"revision\":\"abc\"}", 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withMWREL(t, "REL1_43")
			oldClient, oldError := http.DefaultClient, hasError
			t.Cleanup(func() { http.DefaultClient, hasError = oldClient, oldError })
			hasError = false
			http.DefaultClient = &http.Client{Transport: responseTransport(func(r *http.Request) (*http.Response, error) {
				status, body := tc.status, tc.body
				if !strings.Contains(r.UserAgent(), "https://github.com/ShinyColorsWiki/mediawiki-extension-downloader") {
					status, body = 403, "<!DOCTYPE html>blocked"
				}
				return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}
			got := WMFExtensionUrl("AbuseFilter")
			if hasError != tc.wantError {
				t.Fatalf("hasError = %v, want %v; URL: %s", hasError, tc.wantError, got)
			}
			want := "https://extdist.wmflabs.org/dist/extensions/AbuseFilter-REL1_43-abcdef0.tar.gz"
			if tc.wantError {
				want = "https://github.com/wikimedia/mediawiki-extensions-AbuseFilter/archive/REL1_43.tar.gz"
			}
			if got != want {
				t.Fatalf("URL = %s, want %s", got, want)
			}
		})
	}
}
