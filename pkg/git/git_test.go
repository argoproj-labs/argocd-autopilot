package git

import (
	"context"
	"testing"

	"github.com/codefresh-io/cf-argo/test/utils"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/stretchr/testify/assert"
)

func Test_NewProvider(t *testing.T) {
	tests := map[string]struct {
		opts             *Options
		expectedProvider Provider
		expectedError    string
	}{
		"Github": {
			&Options{
				Type: "github",
			},
			&github{},
			"",
		},
		"No Type": {
			&Options{},
			nil,
			ErrProviderNotSupported.Error(),
		},
		"Bad Type": {
			&Options{Type: "foo"},
			nil,
			ErrProviderNotSupported.Error(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, err := NewProvider(test.opts)
			if test.expectedError != "" {
				assert.EqualError(t, err, test.expectedError)
				return
			}
			assert.IsType(t, test.expectedProvider, p)
		})
	}
}

func Test_Clone(t *testing.T) {
	tests := map[string]struct {
		opts             *CloneOptions
		expectedPath     string
		expectedURL      string
		expectedPassword string
		expectedRefName  plumbing.ReferenceName
	}{
		"Simple": {
			opts: &CloneOptions{
				Path: "/foo/bar",
				URL:  "https://github.com/foo/bar",
				Auth: nil,
			},
			expectedPath:    "/foo/bar",
			expectedURL:     "https://github.com/foo/bar",
			expectedRefName: plumbing.HEAD,
		},
		"With tag": {
			opts: &CloneOptions{
				Path: "/foo/bar",
				URL:  "https://github.com/foo/bar@tag",
				Auth: nil,
			},
			expectedPath:    "/foo/bar",
			expectedURL:     "https://github.com/foo/bar",
			expectedRefName: plumbing.NewTagReferenceName("tag"),
		},
		"With branch": {
			opts: &CloneOptions{
				Path: "/foo/bar",
				URL:  "https://github.com/foo/bar#branch",
				Auth: nil,
			},
			expectedPath:    "/foo/bar",
			expectedURL:     "https://github.com/foo/bar",
			expectedRefName: plumbing.NewBranchReferenceName("branch"),
		},
		"With token": {
			opts: &CloneOptions{
				Path: "/foo/bar",
				URL:  "https://github.com/foo/bar",
				Auth: &Auth{
					Password: "password",
				},
			},
			expectedPath:     "/foo/bar",
			expectedURL:      "https://github.com/foo/bar",
			expectedPassword: "password",
			expectedRefName:  plumbing.HEAD,
		},
	}

	orig := plainClone
	defer func() { plainClone = orig }()

	for name, test := range tests {
		plainClone = func(ctx context.Context, path string, isBare bool, o *gg.CloneOptions) (*gg.Repository, error) {
			assert.Equal(t, test.expectedPath, path)
			assert.Equal(t, test.expectedURL, o.URL)
			assert.Equal(t, test.expectedRefName, o.ReferenceName)
			assert.Equal(t, 1, o.Depth)
			assert.False(t, isBare)
			if o.Auth != nil {
				bauth, _ := o.Auth.(*http.BasicAuth)
				assert.Equal(t, test.expectedPassword, bauth.Password)
			}

			return nil, nil
		}
		t.Run(name, func(t *testing.T) {
			_, _ = Clone(utils.MockLoggerContext(), test.opts)
		})
	}
}
