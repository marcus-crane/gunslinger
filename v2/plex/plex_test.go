package plex

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetUserPlaying_Handle500(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.BaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := PlexResponse{}
	got, err := c.getUserPlaying()
	if err == nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestBuildURL(t *testing.T) {
	t.Parallel()
	c := NewClient("abc123")
	c.BaseURL = "https://example.com"
	want := "https://example.com/blah?X-Plex-Token=abc123"
	got := c.buildUrl("/blah")
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}
