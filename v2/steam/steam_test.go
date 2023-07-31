package steam

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	c.APIBaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := ""
	got, err := c.GetUserPlaying()
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
		res, _ := json.Marshal(SteamPlayerSummary{})
		w.Write(res)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.APIBaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := ""
	got, err := c.GetUserPlaying()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetUserPlaying_SuccessUserInactive(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		f, err := os.Open("testdata/profile_inactive.json")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.APIBaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := ""
	got, err := c.GetUserPlaying()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetUserPlaying_SuccessUserActive(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		f, err := os.Open("testdata/profile_active.json")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.APIBaseURL = ts.URL
	c.HTTPClient = ts.Client()
	want := "1191900"
	got, err := c.GetUserPlaying()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestLookupStoreItem_Handle500(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.StoreBaseURL = ts.URL
	c.HTTPClient = ts.Client()
	_, err := c.LookupStoreItem("123456")
	if err == nil {
		t.Fatal(err)
	}
}

func TestLookupStoreItem_HandleMalformedResponse(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		res, _ := json.Marshal(map[string]SteamAppResponse{"abc1234": {}})
		w.Write(res)
	}))
	defer ts.Close()

	c := NewClient("abc123")
	c.StoreBaseURL = ts.URL
	c.HTTPClient = ts.Client()
	_, err := c.LookupStoreItem("abc123")
	if !cmp.Equal(err.Error(), ERR_STORE_RESPONSE_MALFORMED) {
		t.Fatal(err)
	}
}

func TestLookupStoreItem_Success(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		f, err := os.Open("testdata/store_lookup.json")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	defer ts.Close()

	want := SteamAppDetail{
		Type:        "game",
		Name:        "Say No! More",
		HeaderImage: "https://cdn.akamai.steamstatic.com/steam/apps/1191900/header.jpg?t=1688539586",
		Developers:  []string{"Studio Fizbin"},
	}
	c := NewClient("abc123")
	c.StoreBaseURL = ts.URL
	c.HTTPClient = ts.Client()
	got, err := c.LookupStoreItem("1191900")
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}
