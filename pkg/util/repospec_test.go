// The following code was copied from https://github.com/kubernetes-sigs/kustomize/blob/master/api/internal/git/repospec_test.go
// and modified to test the copied repospec.go

// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

const refQuery = "?ref="

var orgRepos = []string{"someOrg/someRepo", "kubernetes/website"}

var suffixes = []string{"", ".git"}

var pathNames = []string{"README.md", "foo/krusty.txt", ""}

var hrefArgs = []string{"someBranch", "master", "v0.1.0", ""}

var hostNamesRawAndNormalized = [][]string{
	{"gh:", "gh:"},
	{"GH:", "gh:"},
	{"gitHub.com/", "https://github.com/"},
	{"github.com:", "https://github.com/"},
	{"http://github.com/", "https://github.com/"},
	{"https://github.com/", "https://github.com/"},
	{"hTTps://github.com/", "https://github.com/"},
	{"ssh://git.example.com:7999/", "ssh://git.example.com:7999/"},
	{"git::https://gitlab.com/", "https://gitlab.com/"},
	{"git::http://git.example.com/", "http://git.example.com/"},
	{"git::https://git.example.com/", "https://git.example.com/"},
	{"git@github.com:", "git@github.com:"},
	{"git@github.com/", "git@github.com:"},
	{"git@gitlab2.sqtools.ru:10022/", "git@gitlab2.sqtools.ru:10022/"},
}

func makeUrl(hostFmt, orgRepo, suffix, path, href string) string {
	orgRepo += suffix
	if len(path) > 0 {
		orgRepo = filepath.Join(orgRepo, path)
	}
	url := hostFmt + orgRepo
	if href != "" {
		url += refQuery + href
	}
	return url
}

func TestNewRepoSpecFromUrl(t *testing.T) {
	var bad [][]string
	for _, tuple := range hostNamesRawAndNormalized {
		hostRaw := tuple[0]
		hostSpec := tuple[1]
		for _, orgRepo := range orgRepos {
			for _, suffix := range suffixes {
				for _, pathName := range pathNames {
					for _, hrefArg := range hrefArgs {
						uri := makeUrl(hostRaw, orgRepo, suffix, pathName, hrefArg)
						host, org, path, ref, _, _, _ := ParseGitUrl(uri)
						if host != hostSpec {
							bad = append(bad, []string{"host", uri, host, hostSpec})
						}

						if org != orgRepo {
							bad = append(bad, []string{"orgRepo", uri, org, orgRepo})
						}

						if path != pathName {
							bad = append(bad, []string{"path", uri, path, pathName})
						}

						if ref != hrefArg {
							bad = append(bad, []string{"ref", uri, ref, hrefArg})
						}
					}
				}
			}
		}
	}
	if len(bad) > 0 {
		for _, tuple := range bad {
			fmt.Printf("\n"+
				"     from uri: %s\n"+
				"  actual %4s: %s\n"+
				"expected %4s: %s\n",
				tuple[1], tuple[0], tuple[2], tuple[0], tuple[3])
		}
		t.Fail()
	}
}

func TestNewRepoSpecFromUrl_CloneSpecs(t *testing.T) {
	testcases := []struct {
		input     string
		cloneSpec string
		absPath   string
		ref       string
	}{
		{
			input:     "http://github.com/someorg/somerepo/somedir",
			cloneSpec: "https://github.com/someorg/somerepo.git",
			absPath:   "somedir",
			ref:       "",
		},
		{
			input:     "git@github.com:someorg/somerepo/somedir",
			cloneSpec: "git@github.com:someorg/somerepo.git",
			absPath:   "somedir",
			ref:       "",
		},
		{
			input:     "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git?ref=v0.1.0",
			cloneSpec: "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git",
			absPath:   "",
			ref:       "v0.1.0",
		},
		{
			input:     "https://itfs.mycompany.com/collection/project/_git/somerepos",
			cloneSpec: "https://itfs.mycompany.com/collection/project/_git/somerepos",
			absPath:   "",
			ref:       "",
		},
		{
			input:     "https://itfs.mycompany.com/collection/project/_git/somerepos?version=v1.0.0",
			cloneSpec: "https://itfs.mycompany.com/collection/project/_git/somerepos",
			absPath:   "",
			ref:       "v1.0.0",
		},
		{
			input:     "git::https://itfs.mycompany.com/collection/project/_git/somerepos",
			cloneSpec: "https://itfs.mycompany.com/collection/project/_git/somerepos",
			absPath:   "",
			ref:       "",
		},
		{
			input:     "https://gitlab-onprem.devops.cf-cd.com/root/gitlab-demo_git-source/resources_gitlab-demo",
			cloneSpec: "https://gitlab-onprem.devops.cf-cd.com/root/gitlab-demo_git-source.git",
			absPath:   "resources_gitlab-demo",
			ref:       "",
		},
	}
	for _, testcase := range testcases {
		host, orgRepo, path, ref, _, suffix, _ := ParseGitUrl(testcase.input)
		cloneSpec := host + orgRepo + suffix
		if cloneSpec != testcase.cloneSpec {
			t.Errorf("CloneSpec expected to be %v, but got %v on %s",
				testcase.cloneSpec, cloneSpec, testcase.input)
		}
		if path != testcase.absPath {
			t.Errorf("AbsPath expected to be %v, but got %v on %s",
				testcase.absPath, path, testcase.input)
		}
		if ref != testcase.ref {
			t.Errorf("ref expected to be %v, but got %v on %s",
				testcase.ref, ref, testcase.input)
		}
	}
}

func TestPeelQuery(t *testing.T) {
	testcases := []struct {
		input string

		path       string
		ref        string
		submodules bool
		timeout    time.Duration
	}{
		{
			// All empty.
			input:      "somerepos",
			path:       "somerepos",
			ref:        "",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			input:      "somerepos?ref=v1.0.0",
			path:       "somerepos",
			ref:        "v1.0.0",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			input:      "somerepos?version=master",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			// A ref value takes precedence over a version value.
			input:      "somerepos?version=master&ref=v1.0.0",
			path:       "somerepos",
			ref:        "v1.0.0",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			// Empty submodules value uses default.
			input:      "somerepos?version=master&submodules=",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			// Malformed submodules value uses default.
			input:      "somerepos?version=master&submodules=maybe",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			input:      "somerepos?version=master&submodules=true",
			path:       "somerepos",
			ref:        "master",
			submodules: true,
			timeout:    defaultTimeout,
		},
		{
			input:      "somerepos?version=master&submodules=false",
			path:       "somerepos",
			ref:        "master",
			submodules: false,
			timeout:    defaultTimeout,
		},
		{
			// Empty timeout value uses default.
			input:      "somerepos?version=master&timeout=",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			// Malformed timeout value uses default.
			input:      "somerepos?version=master&timeout=jiffy",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			// Zero timeout value uses default.
			input:      "somerepos?version=master&timeout=0",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			input:      "somerepos?version=master&timeout=0s",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    defaultTimeout,
		},
		{
			input:      "somerepos?version=master&timeout=61",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    61 * time.Second,
		},
		{
			input:      "somerepos?version=master&timeout=1m1s",
			path:       "somerepos",
			ref:        "master",
			submodules: defaultSubmodules,
			timeout:    61 * time.Second,
		},
		{
			input:      "somerepos?version=master&submodules=false&timeout=1m1s",
			path:       "somerepos",
			ref:        "master",
			submodules: false,
			timeout:    61 * time.Second,
		},
	}

	for _, testcase := range testcases {
		path, ref, timeout, submodules := peelQuery(testcase.input)
		if path != testcase.path || ref != testcase.ref || timeout != testcase.timeout || submodules != testcase.submodules {
			t.Errorf("peelQuery: expected (%s, %s, %v, %v) got (%s, %s, %v, %v) on %s",
				testcase.path, testcase.ref, testcase.timeout, testcase.submodules,
				path, ref, timeout, submodules,
				testcase.input)
		}
	}
}
