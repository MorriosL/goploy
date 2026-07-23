// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package repo

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zhenorzz/goploy/internal/model"
)

// ErrRepoUnreachable is the client-safe error for an unreachable repository;
// handlers log the wrapped detail and return this fixed message (no
// port/protocol leakage).
var ErrRepoUnreachable = errors.New("repository is not accessible")

// ErrInvalidRepoURL is returned for malformed URLs or disallowed schemes.
var ErrInvalidRepoURL = errors.New("invalid repository URL")

// repoProbeTimeout bounds the git ls-remote / svn ls probes so a blackholing
// remote cannot block the request until the HTTP client times out.
const repoProbeTimeout = 30 * time.Second

// repoSchemes is the per-type allowlist of accepted URL schemes.
var repoSchemes = map[string]map[string]bool{
	model.RepoGit:  {"http": true, "https": true, "ssh": true, "git": true},
	model.RepoSVN:  {"http": true, "https": true, "svn": true, "svn+ssh": true},
	model.RepoFTP:  {"ftp": true, "ftps": true},
	model.RepoSFTP: {"sftp": true, "ssh": true},
}

// ValidateRepoURL enforces a per-type scheme allowlist, blocks never-legit
// addresses (cloud metadata, link-local, multicast, unspecified), and requires
// a genuine smart-HTTP advertisement for HTTP(S) Git URLs. Loopback and private
// ranges stay allowed for same-box/internal hosting.
func ValidateRepoURL(repoType, rawURL string) error {
	allowed, ok := repoSchemes[repoType]
	if !ok {
		return ErrInvalidRepoURL
	}

	scheme, host, isHTTP, err := parseRepoURL(rawURL)
	if err != nil {
		return err
	}
	if !allowed[scheme] {
		return ErrInvalidRepoURL
	}

	if isHTTP {
		return smartHTTPPrecheck(rawURL)
	}
	// Subprocess-based clients (git/svn) can't be IP-pinned, so best-effort
	// resolve-and-check before handing the URL off.
	return checkHostAddress(host)
}

// parseRepoURL splits a repository URL into its lower-cased scheme and host,
// handling scheme:// and SCP-like "[user@]host:path" forms.
func parseRepoURL(raw string) (scheme, host string, isHTTP bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false, ErrInvalidRepoURL
	}

	if strings.Contains(raw, "://") {
		u, perr := url.Parse(raw)
		if perr != nil || u.Hostname() == "" {
			return "", "", false, ErrInvalidRepoURL
		}
		s := strings.ToLower(u.Scheme)
		return s, u.Hostname(), s == "http" || s == "https", nil
	}

	// SCP-like: "[user@]host:path" -- a ":" with no "/" before it and no "://".
	if idx := strings.IndexByte(raw, ':'); idx > 0 && strings.IndexByte(raw[:idx], '/') < 0 {
		userHost := raw[:idx]
		if at := strings.LastIndexByte(userHost, '@'); at >= 0 {
			userHost = userHost[at+1:]
		}
		if userHost == "" {
			return "", "", false, ErrInvalidRepoURL
		}
		return "ssh", userHost, false, nil
	}

	return "", "", false, ErrInvalidRepoURL
}

// checkHostAddress rejects host if any resolved address is blocked
// (conservative: any blocked -> reject).
func checkHostAddress(host string) error {
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("%w: resolve %q", ErrRepoUnreachable, host)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: blocked address for %q", ErrRepoUnreachable, host)
		}
	}
	return nil
}

// isBlockedIP blocks only never-legit ranges (link-local incl. cloud metadata,
// multicast, unspecified); loopback/private stay allowed.
func isBlockedIP(ip net.IP) bool {
	return ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// smartHTTPPrecheck requires a genuine smart-HTTP advertisement (status 200,
// application/x-git-upload-pack-advertisement, "# service=git-upload-pack"
// pkt-line) so a plain 200 from any open port can't be misreported as a
// reachable repo. TLS uses the system root store to match git.
func smartHTTPPrecheck(rawURL string) error {
	endpoint := strings.TrimRight(rawURL, "/") + "/info/refs?service=git-upload-pack"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("%w: build request", ErrRepoUnreachable)
	}
	req.Header.Set("User-Agent", "git/goploy-smart-http-precheck")

	client := &http.Client{
		Timeout: 10 * time.Second,
		// Do not follow redirects: a 3xx could point at an unvalidated internal target.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialChecked(ctx, network, addr)
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRepoUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrRepoUnreachable, resp.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-git-upload-pack-advertisement" {
		return fmt.Errorf("%w: not a smart-HTTP git service", ErrRepoUnreachable)
	}

	// Bound the read so a hostile server cannot stream an unbounded body.
	br := bufio.NewReader(io.LimitReader(resp.Body, 512))
	line, err := readFirstPktLine(br)
	if err != nil || !strings.HasPrefix(line, "# service=git-upload-pack") {
		return fmt.Errorf("%w: malformed advertisement", ErrRepoUnreachable)
	}
	return nil
}

// dialChecked skips blocked IPs and dials a validated one directly, pinning
// the connection so resolution can't redirect to a blocked address.
func dialChecked(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	var lastErr error
	for _, ip := range ips {
		if isBlockedIP(ip.IP) {
			continue
		}
		conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if derr == nil {
			return conn, nil
		}
		lastErr = derr
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("%w: blocked address", ErrRepoUnreachable)
}

// readFirstPktLine reads one git pkt-line: 4 hex length digits then payload;
// "0000" flush yields "".
func readFirstPktLine(r *bufio.Reader) (string, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return "", err
	}
	n, err := strconv.ParseInt(string(lenBuf[:]), 16, 64)
	if err != nil {
		return "", fmt.Errorf("parse pkt-line length: %w", err)
	}
	if n == 0 {
		return "", nil
	}
	if n < 4 {
		return "", fmt.Errorf("invalid pkt-line length %d", n)
	}
	payload := make([]byte, n-4)
	if _, err := io.ReadFull(r, payload); err != nil {
		return "", err
	}
	return string(payload), nil
}
