package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostNtfy(t *testing.T) {
	var gotPath, gotTitle, gotClick, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTitle = r.Header.Get("Title")
		gotClick = r.Header.Get("Click")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer srv.Close()
	old := ntfyBaseURL
	ntfyBaseURL = srv.URL
	defer func() { ntfyBaseURL = old }()

	if err := postNtfy("topic-x", "hhmi needs you", "merge #1?", "https://github.com/o/r/pull/1"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/topic-x" || gotTitle != "hhmi needs you" || gotBody != "merge #1?" || gotClick != "https://github.com/o/r/pull/1" {
		t.Errorf("got path=%q title=%q body=%q click=%q", gotPath, gotTitle, gotBody, gotClick)
	}
}

func TestPostNtfyErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	old := ntfyBaseURL
	ntfyBaseURL = srv.URL
	defer func() { ntfyBaseURL = old }()

	if err := postNtfy("t", "x", "y", ""); err == nil {
		t.Error("want error on 429")
	}
}
