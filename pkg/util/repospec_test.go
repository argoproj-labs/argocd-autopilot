// The following code was copied from https://github.com/kubernetes-sigs/kustomize/blob/master/api/internal/git/repospec_test.go
// and modified to test the copied repospec.go

// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"path/filepath"
	"testing"
)

const refQuery = "?ref="

var orgRepos = []string{"someOrg/someRepo", "kubernetes/website"}

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

func makeUrl(hostFmt, orgRepo, path, href string) string {
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
			for _, pathName := range pathNames {
				for _, hrefArg := range hrefArgs {
					uri := makeUrl(hostRaw, orgRepo, pathName, hrefArg)
					host, org, path, ref, _ := ParseGitUrl(uri)
					if host != hostSpec {
						bad = append(bad, []string{"host", uri, host, hostSpec})
					}
					if org != orgRepo {
						bad = append(bad, []string{"orgRepo", uri, orgRepo, orgRepo})
					}
					if path != pathName {
						bad = append(bad, []string{"path", uri, path, pathName})
					}
					if hrefArg != "" && ref != "refs/heads/"+hrefArg {
						bad = append(bad, []string{"ref", uri, ref, hrefArg})
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
		tag       string
		sha       string
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
			input:     "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git?ref=branch",
			cloneSpec: "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git",
			absPath:   "",
			ref:       "refs/heads/branch",
		},
		{
			input:     "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git?tag=v0.1.0",
			cloneSpec: "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git",
			absPath:   "",
			ref:       "refs/tags/v0.1.0",
		},
		{
			input:     "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git?sha=some_sha",
			cloneSpec: "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git",
			absPath:   "",
			ref:       "some_sha",
		},
		{
			input:     "https://itfs.mycompany.com/collection/project/_git/somerepos",
			cloneSpec: "https://itfs.mycompany.com/collection/project/_git/somerepos",
			absPath:   "",
			ref:       "",
		},
		{
			input:     "git::https://itfs.mycompany.com/collection/project/_git/somerepos",
			cloneSpec: "https://itfs.mycompany.com/collection/project/_git/somerepos",
			absPath:   "",
			ref:       "",
		},
	}
	for _, testcase := range testcases {
		host, orgRepo, path, ref, suffix := ParseGitUrl(testcase.input)
		cloneSpec := host + orgRepo + suffix
		if cloneSpec != testcase.cloneSpec {
			t.Errorf("CloneSpec expected to be %v, but got %v on %s", testcase.cloneSpec, cloneSpec, testcase.input)
		}

		if path != testcase.absPath {
			t.Errorf("AbsPath expected to be %v, but got %v on %s", testcase.absPath, path, testcase.input)
		}

		if ref != testcase.ref {
			t.Errorf("ref expected to be %v, but got %v on %s", testcase.ref, ref, testcase.input)
		}
	}
}

func TestPeelQuery(t *testing.T) {
	testcases := []struct {
		input string

		path string
		ref  string
	}{
		{
			// All empty.
			input: "somerepos",
			path:  "somerepos",
		},
		{
			input: "somerepos?ref=branch",
			path:  "somerepos",
			ref:   "refs/heads/branch",
		},
		{
			input: "somerepos?tag=v1.0.0",
			path:  "somerepos",
			ref:   "refs/tags/v1.0.0",
		},
		{
			input: "somerepos?sha=some_sha",
			path:  "somerepos",
			ref:   "some_sha",
		},
		{
			input: "somerepos?ref=branch&tag=v1.0.0",
			path:  "somerepos",
			ref:   "refs/tags/v1.0.0",
		},
		{
			input: "somerepos?ref=branch&sha=some_sha",
			path:  "somerepos",
			ref:   "some_sha",
		},
		{
			input: "somerepos?sha=some_sha&tag=v1.0.0",
			path:  "somerepos",
			ref:   "some_sha",
		},
	}

	for _, testcase := range testcases {
		path, ref := peelQuery(testcase.input)
		if path != testcase.path || ref != testcase.ref {
			t.Errorf("peelQuery: expected (%s, %s) got (%s, %s) on %s",
				testcase.path, testcase.ref,
				path, ref,
				testcase.input)
		}
	}
}
