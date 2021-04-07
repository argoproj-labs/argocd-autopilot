package git

// import (
// 	"context"
// 	"errors"
// 	"testing"

// 	billy "github.com/go-git/go-billy/v5"
// 	gg "github.com/go-git/go-git/v5"
// 	"github.com/go-git/go-git/v5/plumbing"
// 	"github.com/go-git/go-git/v5/plumbing/transport/http"
// 	"github.com/go-git/go-git/v5/storage"
// 	"github.com/stretchr/testify/assert"
// )

// // func Test_Clone(t *testing.T) {
// // 	tests := map[string]struct {
// // 		opts             *CloneOptions
// // 		mockError        error
// // 		expectedURL      string
// // 		expectedPassword string
// // 		expectedRefName  plumbing.ReferenceName
// // 		expectedErr      string
// // 	}{
// // 		"Simple": {
// // 			opts: &CloneOptions{
// // 				URL: "https://github.com/foo/bar",
// // 			},
// // 			expectedURL:     "https://github.com/foo/bar",
// // 			expectedRefName: plumbing.HEAD,
// // 		},
// // 		"With branch": {
// // 			opts: &CloneOptions{
// // 				URL:      "https://github.com/foo/bar",
// // 				Revision: "branch",
// // 			},
// // 			expectedURL:     "https://github.com/foo/bar",
// // 			expectedRefName: plumbing.NewBranchReferenceName("branch"),
// // 		},
// // 		"With token": {
// // 			opts: &CloneOptions{
// // 				URL: "https://github.com/foo/bar",
// // 				Auth: &Auth{
// // 					Password: "password",
// // 				},
// // 			},
// // 			expectedURL:      "https://github.com/foo/bar",
// // 			expectedPassword: "password",
// // 			expectedRefName:  plumbing.HEAD,
// // 		},
// // 		"Empty URL": {
// // 			opts: &CloneOptions{
// // 				URL: "",
// // 			},
// // 			expectedErr: "URL field is required",
// // 		},
// // 		"No Options": {
// // 			expectedErr: "options cannot be nil",
// // 		},
// // 		"Clone error": {
// // 			opts: &CloneOptions{
// // 				URL: "https://github.com/foo/bar",
// // 			},
// // 			mockError:       errors.New("some error"),
// // 			expectedURL:     "https://github.com/foo/bar",
// // 			expectedRefName: plumbing.HEAD,
// // 			expectedErr:     "some error",
// // 		},
// // 	}

// // 	orig := clone

// // 	defer func() { clone = orig }()

// // 	for name, test := range tests {
// // 		clone = func(ctx context.Context, s storage.Storer, worktree billy.Filesystem, o *gg.CloneOptions) (*gg.Repository, error) {
// // 			assert.Equal(t, test.expectedURL, o.URL)
// // 			assert.Equal(t, test.expectedRefName, o.ReferenceName)
// // 			assert.True(t, o.SingleBranch)
// // 			assert.Equal(t, 1, o.Depth)
// // 			assert.Equal(t, gg.NoTags, o.Tags)

// // 			if o.Auth != nil {
// // 				bauth, _ := o.Auth.(*http.BasicAuth)
// // 				assert.Equal(t, test.expectedPassword, bauth.Password)
// // 			}

// // 			return nil, test.mockError
// // 		}

// // 		t.Run(name, func(t *testing.T) {
// // 			_, err := Clone(context.Background(), nil, test.opts)
// // 			if test.expectedErr != "" {
// // 				assert.EqualError(t, err, test.expectedErr)
// // 			}
// // 		})
// // 	}
// // }

// // func Test_repo_Add(t *testing.T) {
// // 	type fields struct {
// // 		r *gg.Repository
// // 	}

// // 	type args struct {
// // 		ctx     context.Context
// // 		pattern string
// // 	}

// // 	tests := []struct {
// // 		name    string
// // 		fields  fields
// // 		args    args
// // 		wantErr bool
// // 	}{}

// // 	for _, tt := range tests {
// // 		t.Run(tt.name, func(t *testing.T) {
// // 			r := &repo{
// // 				r: tt.fields.r,
// // 			}

// // 			if err := r.Add(tt.args.ctx, tt.args.pattern); (err != nil) != tt.wantErr {
// // 				t.Errorf("repo.Add() error = %v, wantErr %v", err, tt.wantErr)
// // 			}
// // 		})
// // 	}
// // }
