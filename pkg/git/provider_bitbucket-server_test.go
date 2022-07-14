package git

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/git/bitbucket-server/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var providerOptions = &ProviderOptions{
	Auth: &Auth{
		Username: "username",
		Password: "password",
	},
}

func createBody(obj interface{}) io.ReadCloser {
	data, _ := json.Marshal(obj)
	return io.NopCloser(strings.NewReader(string(data)))
}

func Test_bitbucketServer_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		orgRepo  string
		want     string
		wantErr  string
		beforeFn func(t *testing.T, c *mocks.MockHttpClient)
	}{
		"Should fail if orgRepo is invalid": {
			orgRepo: "no-scm/project/repo.git",
			wantErr: "invalid Bitbucket url \"no-scm/project/repo.git\" - must be in the form of \"scm/[~]project-or-username/repo-name[.git]\"",
		},
		"Should fail if repos POST fails": {
			orgRepo: "scm/project/repo.git",
			wantErr: "some error",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).Return(nil, errors.New("some error"))
			},
		},
		"Should fail if returned repo doesn't have clone url": {
			orgRepo: "scm/project/repo.git",
			wantErr: "created repo did not contain a valid https clone url",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					repo := &repoResponse{
						Links: Links{
							Clone: []Link{
								{
									Name: "ssh",
									Href: "ssh@some.server/scm/project/repo.git",
								},
							},
						},
					}
					body := createBody(repo)
					res := &http.Response{
						StatusCode: 200,
						Body:       body,
					}
					return res, nil
				})
			},
		},
		"Should create a valid project repo": {
			orgRepo: "scm/project/repo.git",
			want:    "https://some.server/scm/project/repo.git",
			beforeFn: func(t *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "https://some.server/projects/project/repos", req.URL.String())
					repo := &repoResponse{
						Links: Links{
							Clone: []Link{
								{
									Name: "https",
									Href: "https://some.server/scm/project/repo.git",
								},
							},
						},
					}
					body := createBody(repo)
					res := &http.Response{
						StatusCode: 200,
						Body:       body,
					}
					return res, nil
				})
			},
		},
		"Should create a valid user repo": {
			orgRepo: "scm/~user/repo.git",
			want:    "https://some.server/scm/~user/repo.git",
			beforeFn: func(t *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "https://some.server/users/user/repos", req.URL.String())
					repo := &repoResponse{
						Links: Links{
							Clone: []Link{
								{
									Name: "https",
									Href: "https://some.server/scm/~user/repo.git",
								},
							},
						},
					}
					body := createBody(repo)
					res := &http.Response{
						StatusCode: 200,
						Body:       body,
					}
					return res, nil
				})
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockClient := mocks.NewMockHttpClient(ctrl)
			if tt.beforeFn != nil {
				tt.beforeFn(t, mockClient)
			}

			bbs := &bitbucketServer{
				baseURl: "https://some.server",
				c:       mockClient,
				opts:    providerOptions,
			}
			got, err := bbs.CreateRepository(context.Background(), tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_bitbucketServer_GetDefaultBranch(t *testing.T) {
	tests := map[string]struct {
		orgRepo  string
		want     string
		wantErr  string
		beforeFn func(t *testing.T, c *mocks.MockHttpClient)
	}{
		"Should fail if orgRepo is invalid": {
			orgRepo: "no-scm/project/repo.git",
			wantErr: "invalid Bitbucket url \"no-scm/project/repo.git\" - must be in the form of \"scm/[~]project-or-username/repo-name[.git]\"",
		},
		"Should fail if repos GET fails": {
			orgRepo: "scm/project/repo.git",
			wantErr: "some error",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).Return(nil, errors.New("some error"))
			},
		},
		"Should return defaultBranch from project repo": {
			orgRepo: "scm/project/repo.git",
			want:    "some-branch",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "https://some.server/projects/project/repos/repo", req.URL.String())
					repo := &repoResponse{
						DefaultBranch: "some-branch",
					}
					body := createBody(repo)
					res := &http.Response{
						StatusCode: 200,
						Body:       body,
					}
					return res, nil
				})
			},
		},
		"Should return defaultBranch from user repo": {
			orgRepo: "scm/~user/repo.git",
			want:    "some-branch",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "https://some.server/users/user/repos/repo", req.URL.String())
					repo := &repoResponse{
						DefaultBranch: "some-branch",
					}
					body := createBody(repo)
					res := &http.Response{
						StatusCode: 200,
						Body:       body,
					}
					return res, nil
				})
			},
		},
		"Should return 'master' if no defaultBranch is set": {
			orgRepo: "scm/project/repo.git",
			want:    "master",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					repo := &repoResponse{}
					body := createBody(repo)
					res := &http.Response{
						StatusCode: 200,
						Body:       body,
					}
					return res, nil
				})
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockClient := mocks.NewMockHttpClient(ctrl)
			if tt.beforeFn != nil {
				tt.beforeFn(t, mockClient)
			}

			bbs := &bitbucketServer{
				baseURl: "https://some.server",
				c:       mockClient,
				opts:    providerOptions,
			}
			got, err := bbs.GetDefaultBranch(context.Background(), tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)

		})
	}
}

func Test_bitbucketServer_GetAuthor(t *testing.T) {
	tests := map[string]struct {
		wantUsername string
		wantEmail    string
		wantErr      string
		beforeFn     func(t *testing.T, c *mocks.MockHttpClient)
	}{
		"Should fail if user GET fails": {
			wantErr: "some error",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).Return(nil, errors.New("some error"))
			},
		},
		"Should return displayName and emailAddress if available": {
			wantUsername: "displayName",
			wantEmail:    "username@email",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "https://some.server/users/username", req.URL.String())
					user := &userResponse{
						DisplayName:  "displayName",
						EmailAddress: "username@email",
					}
					res := &http.Response{
						StatusCode: 200,
						Body:       createBody(user),
					}
					return res, nil
				})
			},
		},
		"Should return name and slug if no displayName and emailAddress": {
			wantUsername: "name",
			wantEmail:    "slug",
			beforeFn: func(_ *testing.T, c *mocks.MockHttpClient) {
				c.EXPECT().Do(gomock.AssignableToTypeOf(&http.Request{})).Times(1).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "https://some.server/users/username", req.URL.String())
					user := &userResponse{
						Name: "name",
						Slug: "slug",
					}
					res := &http.Response{
						StatusCode: 200,
						Body:       createBody(user),
					}
					return res, nil
				})
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockClient := mocks.NewMockHttpClient(ctrl)
			if tt.beforeFn != nil {
				tt.beforeFn(t, mockClient)
			}

			bbs := &bitbucketServer{
				baseURl: "https://some.server",
				c:       mockClient,
				opts:    providerOptions,
			}
			gotUsername, gotEmail, err := bbs.GetAuthor(context.Background())
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.wantUsername, gotUsername, "username mismatch")
			assert.Equal(t, tt.wantEmail, gotEmail, "email mismatch")
		})
	}
}

func Test_splitOrgRepo(t *testing.T) {
	tests := map[string]struct {
		orgRepo   string
		wantNoun  string
		wantOwner string
		wantName  string
		wantErr   string
	}{
		"Should fail if it doesn't start with 'scm'": {
			orgRepo: "no-scm-start/project/repo.git",
			wantErr: "invalid Bitbucket url \"no-scm-start/project/repo.git\" - must be in the form of \"scm/[~]project-or-username/repo-name[.git]\"",
		},
		"Should fail if it doesn't end with '.git'": {
			orgRepo: "scm/project/repo.git-more-text",
			wantErr: "invalid Bitbucket url \"scm/project/repo.git-more-text\" - must be in the form of \"scm/[~]project-or-username/repo-name[.git]\"",
		},
		"Should fail if it contains more segments": {
			orgRepo: "scm/project/sub/repo.git",
			wantErr: "invalid Bitbucket url \"scm/project/sub/repo.git\" - must be in the form of \"scm/[~]project-or-username/repo-name[.git]\"",
		},
		"Should succeed for a simple orgRepo": {
			orgRepo: "scm/project/repo",
			wantNoun: "projects",
			wantOwner: "project",
			wantName: "repo",
		},
		"Should succeed with '.git'": {
			orgRepo: "scm/project/repo.git",
			wantNoun: "projects",
			wantOwner: "project",
			wantName: "repo",
		},
		"Should identify ~ as users": {
			orgRepo: "scm/~user/repo.git",
			wantNoun: "users",
			wantOwner: "user",
			wantName: "repo",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotNoun, gotOwner, gotName, err := splitOrgRepo(tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.wantNoun, gotNoun)
			assert.Equal(t, tt.wantOwner, gotOwner)
			assert.Equal(t, tt.wantName, gotName)
		})
	}
}
