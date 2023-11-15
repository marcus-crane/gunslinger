package plex

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/marcus-crane/gunslinger/shared"
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

func TestGetUserPlaying_HandleMalformedResponse(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		res, _ := json.Marshal(PlexResponse{})
		w.Write(res)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.BaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := PlexResponse{}
	got, err := c.getUserPlaying()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetUserPlaying_SuccessMultiplePlaying(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fixture, err := filepath.Abs("testdata/status_multiple.json")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(fixture)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.BaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := PlexResponse{
		MediaContainer: MediaContainer{
			Size: 2,
			Metadata: []Metadata{
				{
					Duration:         2640264,
					GrandparentTitle: "House",
					Thumb:            "/library/metadata/35056/thumb/1689405799",
					ParentThumb:      "/library/metadata/35045/thumb/1689405787",
					Index:            11,
					ParentIndex:      7,
					ParentTitle:      "Season 7",
					Title:            "Family Practice",
					Type:             shared.CATEGORY_EPISODE,
					ViewOffset:       1654361,
					Director:         []Director{{Name: "Miguel Sapochnik"}},
					Player:           Player{State: "playing"},
				},
				{
					Attribution:      "com.tidal",
					Duration:         185706,
					GrandparentTitle: "Broadcast",
					Thumb:            "/library/metadata/38878/thumb/1698216009",
					ParentThumb:      "/library/metadata/38878/thumb/1698216009",
					Index:            1,
					ParentIndex:      1,
					ParentTitle:      "BBC Maida Vale Sessions",
					Title:            "The Note [Message From Home]",
					Type:             "track",
					Player:           Player{State: "playing"},
				},
			},
		},
	}
	got, err := c.getUserPlaying()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetUserPlaying_SuccessMusicPlaying(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fixture, err := filepath.Abs("testdata/status_music.json")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(fixture)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.BaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := PlexResponse{
		MediaContainer: MediaContainer{
			Size: 1,
			Metadata: []Metadata{
				{
					Duration:         110837,
					GrandparentTitle: "天気予報",
					Thumb:            "/library/metadata/38306/thumb/1697708433",
					ParentThumb:      "/library/metadata/38306/thumb/1697708433",
					Index:            9,
					ParentIndex:      1,
					ParentTitle:      "データベースコンパイラ (Database Compiler)",
					Title:            "あなたの夢を運転する",
					Type:             "track",
					ViewOffset:       15704,
					Player:           Player{State: "playing"},
				},
			},
		},
	}
	got, err := c.getUserPlaying()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetUserPlaying_SuccessTVPlaying(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fixture, err := filepath.Abs("testdata/status_tv.json")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(fixture)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.BaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := PlexResponse{
		MediaContainer: MediaContainer{
			Size: 1,
			Metadata: []Metadata{
				{
					Duration:         2640264,
					GrandparentTitle: "House",
					Thumb:            "/library/metadata/35056/thumb/1689405799",
					ParentThumb:      "/library/metadata/35045/thumb/1689405787",
					Index:            11,
					ParentIndex:      7,
					ParentTitle:      "Season 7",
					Title:            "Family Practice",
					Type:             shared.CATEGORY_EPISODE,
					ViewOffset:       255922,
					Director:         []Director{{Name: "Miguel Sapochnik"}},
					Player:           Player{State: "playing"},
				},
			},
		},
	}
	got, err := c.getUserPlaying()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestQueryMediaState_SuccessMultiplePlaying(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fixture, err := filepath.Abs("testdata/status_multiple.json")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(fixture)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.BaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := []shared.DBMediaItem{
		{
			Title:    "07x11 Family Practice",
			Subtitle: "House",
			Category: shared.CATEGORY_EPISODE,
			IsActive: true,
			Elapsed:  1654361,
			Duration: 2640264,
			Source:   shared.SOURCE_PLEX,
		},
		{
			Title:    "The Note [Message From Home]",
			Subtitle: "BBC Maida Vale Sessions",
			Author:   "Broadcast",
			Category: "track",
			IsActive: true,
			Duration: 185706,
			Source:   "plex",
		},
	}
	got, err := c.QueryMediaState()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestQueryMediaState_SuccessMusicPlaying(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fixture, err := filepath.Abs("testdata/status_music.json")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(fixture)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.BaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := []shared.DBMediaItem{
		{
			Title:    "あなたの夢を運転する",
			Subtitle: "データベースコンパイラ (Database Compiler)",
			Author:   "天気予報",
			Category: shared.CATEGORY_TRACK,
			IsActive: true,
			Elapsed:  15704,
			Duration: 110837,
			Source:   shared.SOURCE_PLEX,
		},
	}
	got, err := c.QueryMediaState()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestQueryMediaState_SuccessTVPlaying(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fixture, err := filepath.Abs("testdata/status_tv.json")
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(fixture)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.BaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := []shared.DBMediaItem{
		{
			Title:    "07x11 Family Practice",
			Subtitle: "House",
			Category: shared.CATEGORY_EPISODE,
			IsActive: true,
			Elapsed:  255922,
			Duration: 2640264,
			Source:   shared.SOURCE_PLEX,
		},
	}
	got, err := c.QueryMediaState()
	if err != nil {
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
