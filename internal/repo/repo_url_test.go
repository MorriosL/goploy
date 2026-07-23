// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package repo

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhenorzz/goploy/internal/model"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantSch  string
		wantHost string
		wantHTTP bool
		wantErr  bool
	}{
		{name: "https", raw: "https://github.com/org/repo.git", wantSch: "https", wantHost: "github.com", wantHTTP: true},
		{name: "http with port", raw: "http://127.0.0.1:8929/g/repo.git", wantSch: "http", wantHost: "127.0.0.1", wantHTTP: true},
		{name: "ssh scheme", raw: "ssh://git@github.com:22/org/repo.git", wantSch: "ssh", wantHost: "github.com"},
		{name: "git scheme", raw: "git://github.com/org/repo.git", wantSch: "git", wantHost: "github.com"},
		{name: "scp style", raw: "git@github.com:org/repo.git", wantSch: "ssh", wantHost: "github.com"},
		{name: "scp style custom user", raw: "deploy@gitlab.local:group/repo.git", wantSch: "ssh", wantHost: "gitlab.local"},
		{name: "uppercase scheme normalized", raw: "HTTPS://Github.com/o/r.git", wantSch: "https", wantHost: "Github.com", wantHTTP: true},
		{name: "file scheme empty host", raw: "file:///etc/passwd", wantErr: true},
		{name: "empty", raw: "", wantErr: true},
		{name: "no scheme no colon", raw: "just-a-path", wantErr: true},
		{name: "scheme without host", raw: "http://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sch, host, isHTTP, err := parseRepoURL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got scheme=%q host=%q", sch, host)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sch != tt.wantSch {
				t.Errorf("scheme = %q, want %q", sch, tt.wantSch)
			}
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
			if isHTTP != tt.wantHTTP {
				t.Errorf("isHTTP = %v, want %v", isHTTP, tt.wantHTTP)
			}
		})
	}
}

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"loopback v4", "127.0.0.1", false},
		{"loopback v4 high", "127.255.255.255", false},
		{"private 10", "10.0.0.1", false},
		{"private 172", "172.16.0.1", false},
		{"private 192", "192.168.1.1", false},
		{"loopback v6", "::1", false},
		{"public", "8.8.8.8", false},
		{"cloud metadata", "169.254.169.254", true},
		{"link-local v4", "169.254.0.1", true},
		{"link-local v6", "fe80::1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"unspecified v6", "::", true},
		{"multicast v4", "224.0.0.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBlockedIP(net.ParseIP(tt.ip)); got != tt.want {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestValidateRepoURL(t *testing.T) {
	// Genuine smart-HTTP advertisement: first pkt-line "# service=git-upload-pack\n"
	// (total length 30 = 0x1e) followed by a flush packet "0000".
	const advert = "001e# service=git-upload-pack\n0000"

	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("service") != "git-upload-pack" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		_, _ = w.Write([]byte(advert))
	}))
	defer good.Close()

	// 200 to everything with a non-git content type; mimics a random open port
	// or fake server that dumb-HTTP fallback would accept as a valid repo.
	dumb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	}))
	defer dumb.Close()

	// Mimics a Docker-style daemon answering HTTP with JSON.
	jsonSvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer jsonSvc.Close()

	tests := []struct {
		name     string
		repoType string
		url      string
		wantErr  bool
	}{
		{"smart http git passes", model.RepoGit, good.URL + "/g/repo.git", false},
		{"dumb http rejected", model.RepoGit, dumb.URL + "/g/repo.git", true},
		{"json service rejected", model.RepoGit, jsonSvc.URL + "/g/repo.git", true},
		{"cloud metadata blocked", model.RepoGit, "http://169.254.169.254/g/repo.git", true},
		{"file scheme rejected", model.RepoGit, "file://localhost/etc/passwd", true},
		{"gopher scheme rejected", model.RepoGit, "gopher://127.0.0.1:6379/x", true},
		{"ssh loopback allowed", model.RepoGit, "ssh://127.0.0.1/repo.git", false},
		{"scp loopback allowed", model.RepoGit, "git@127.0.0.1:repo.git", false},
		{"unknown repo type", "bogus", "https://github.com/o/r.git", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepoURL(tt.repoType, tt.url)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
