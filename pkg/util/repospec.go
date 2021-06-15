// The following file was copied from https://github.com/kubernetes-sigs/kustomize/blob/master/api/internal/git/repospec.go
// and modified to expose the ParseGitUrl function

// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"net/url"
	"strings"
	"time"
)

const (
	gitSuffix    = ".git"
	gitDelimiter = "_git/"
)

// From strings like git@github.com:someOrg/someRepo.git or
// https://github.com/someOrg/someRepo?ref=someHash, extract
// the parts.
func ParseGitUrl(n string) (host, orgRepo, path, ref, gitSuff string) {
	if strings.Contains(n, gitDelimiter) {
		index := strings.Index(n, gitDelimiter)
		// Adding _git/ to host
		host = normalizeGitHostSpec(n[:index+len(gitDelimiter)])
		orgRepo = strings.Split(strings.Split(n[index+len(gitDelimiter):], "/")[0], "?")[0]
		path, ref = peelQuery(n[index+len(gitDelimiter)+len(orgRepo):])
		return
	}

	host, n = parseHostSpec(n)
	gitSuff = gitSuffix
	if strings.Contains(n, gitSuffix) {
		index := strings.Index(n, gitSuffix)
		orgRepo = n[0:index]
		n = n[index+len(gitSuffix):]
		path, ref = peelQuery(n)
		return
	}

	i := strings.Index(n, "/")
	if i < 1 {
		path, ref = peelQuery(n)
		return
	}

	j := strings.Index(n[i+1:], "/")
	if j >= 0 {
		j += i + 1
		orgRepo = n[:j]
		path, ref = peelQuery(n[j+1:])
		return
	}

	path = ""
	orgRepo, ref = peelQuery(n)
	return
}

// Clone git submodules by default.
const defaultSubmodules = true

// Arbitrary, but non-infinite, timeout for running commands.
const defaultTimeout = 27 * time.Second

func peelQuery(arg string) (path, ref string) {
	parsed, err := url.Parse(arg)
	if err != nil {
		return path, ""
	}

	values := parsed.Query()
	branch := values.Get("ref")
	tag := values.Get("tag")
	sha := values.Get("sha")
	if sha != "" {
		ref = sha
	} else if tag != "" {
		ref = "refs/tags/" + tag
	} else if branch != "" {
		ref = "refs/heads/" + branch
	}

	return parsed.Path, ref
}

func parseHostSpec(n string) (string, string) {
	var host string
	// Start accumulating the host part.
	for _, p := range [...]string{
		// Order matters here.
		"git::", "gh:", "ssh://", "https://", "http://",
		"git@", "github.com:", "github.com/"} {
		if len(p) < len(n) && strings.ToLower(n[:len(p)]) == p {
			n = n[len(p):]
			host += p
		}
	}
	if host == "git@" {
		i := strings.Index(n, "/")
		if i > -1 {
			host += n[:i+1]
			n = n[i+1:]
		} else {
			i = strings.Index(n, ":")
			if i > -1 {
				host += n[:i+1]
				n = n[i+1:]
			}
		}
		return host, n
	}

	// If host is a http(s) or ssh URL, grab the domain part.
	for _, p := range [...]string{
		"ssh://", "https://", "http://"} {
		if strings.HasSuffix(host, p) {
			i := strings.Index(n, "/")
			if i > -1 {
				host = host + n[0:i+1]
				n = n[i+1:]
			}
			break
		}
	}

	return normalizeGitHostSpec(host), n
}

func normalizeGitHostSpec(host string) string {
	s := strings.ToLower(host)
	if strings.Contains(s, "github.com") {
		if strings.Contains(s, "git@") || strings.Contains(s, "ssh:") {
			host = "git@github.com:"
		} else {
			host = "https://github.com/"
		}
	}
	if strings.HasPrefix(s, "git::") {
		host = strings.TrimPrefix(s, "git::")
	}
	return host
}
