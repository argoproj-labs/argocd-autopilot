package git

import (
	"context"
	"testing"

	bbmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/bitbucket/mocks"
	"github.com/golang/mock/gomock"
	bb "github.com/ktrysmt/go-bitbucket"
	"github.com/stretchr/testify/assert"
)

func Test_bitbucket_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		orgRepo      string
		want         string
		wantErr      string
		beforeRepoFn func(*bbmocks.MockbbRepo)
	}{
		"Creates repository under user": {
			orgRepo: "username/repoName",
			want:    "https://username@bitbucket.org/username/repoName.git",
			beforeRepoFn: func(c *bbmocks.MockbbRepo) {
				createOpts := bb.RepositoryOptions{
					Owner:     "username",
					RepoSlug:  "repoName",
					Scm:       "git",
					IsPrivate: "true",
				}

				links := map[string]interface{}{
					"self": map[string]string{
						"href": "https://api.bitbucket.org/2.0/repositories/userName/repoName",
					},
					"clone": []interface{}{
						map[string]interface{}{
							"name": "https",
							"href": "https://username@bitbucket.org/username/repoName.git",
						},
					},
				}

				repo := &bb.Repository{
					Name:  "userName",
					Links: links,
				}

				c.EXPECT().Create(&createOpts).
					Times(1).
					Return(repo, nil)
			},
		},
		"Fails if token missing required permissions scopes": {
			orgRepo: "username/repoName",
			wantErr: "failed creating the repository \"repoName\" under \"username\": 403 Forbidden",
			beforeRepoFn: func(c *bbmocks.MockbbRepo) {
				createOpts := bb.RepositoryOptions{
					Owner:     "username",
					RepoSlug:  "repoName",
					Scm:       "git",
					IsPrivate: "true",
				}

				err := &bb.UnexpectedResponseStatusError{
					Status: "403 Forbidden",
				}

				c.EXPECT().Create(&createOpts).
					Times(1).
					Return(nil, err)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepoClient := bbmocks.NewMockbbRepo(gomock.NewController(t))
			mockUserClient := bbmocks.NewMockbbUser(gomock.NewController(t))

			tt.beforeRepoFn(mockRepoClient)

			g := &bitbucket{
				client: &bbClientImpl{
					Repository: mockRepoClient,
					User:       mockUserClient,
				},
			}
			got, err := g.CreateRepository(context.Background(), tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_bitbucket_GetDefaultBranch(t *testing.T) {
	tests := map[string]struct {
		orgRepo      string
		want         string
		wantErr      string
		beforeRepoFn func(*bbmocks.MockbbRepo)
	}{
		"Should fail if orgRepo is invalid": {
			orgRepo: "invalid",
			wantErr: "failed parsing organization and repo from 'invalid'",
		},
		"Should fail if repo Get fails with 403": {
			orgRepo: "owner/repo",
			wantErr: "403 Forbidden",
			beforeRepoFn: func(c *bbmocks.MockbbRepo) {
				err := &bb.UnexpectedResponseStatusError{
					Status: "403 Forbidden",
				}
				getOpts := &bb.RepositoryOptions{
					Owner:     "owner",
					RepoSlug:  "repo",
					Scm:       "git",
					IsPrivate: "true",
				}
				c.EXPECT().Get(getOpts).
					Times(1).
					Return(nil, err)
			},
		},
		"Should fail if repo Get fails with 404 - not found ": {
			orgRepo: "owner/repo",
			wantErr: "404 Not Found",
			beforeRepoFn: func(c *bbmocks.MockbbRepo) {
				err := &bb.UnexpectedResponseStatusError{
					Status: "404 Not Found",
				}
				getOpts := &bb.RepositoryOptions{
					Owner:     "owner",
					RepoSlug:  "repo",
					Scm:       "git",
					IsPrivate: "true",
				}
				c.EXPECT().Get(getOpts).
					Times(1).
					Return(nil, err)
			},
		},
		"Should succeed with valid default branch": {
			orgRepo: "owner/repo",
			want:    "master",
			beforeRepoFn: func(c *bbmocks.MockbbRepo) {
				res := &bb.Repository{
					Mainbranch: bb.RepositoryBranch{
						Type: "branch",
						Name: "master",
					},
				}
				getOpts := &bb.RepositoryOptions{
					Owner:     "owner",
					RepoSlug:  "repo",
					Scm:       "git",
					IsPrivate: "true",
				}
				c.EXPECT().Get(getOpts).
					Times(1).
					Return(res, nil)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepoClient := bbmocks.NewMockbbRepo(gomock.NewController(t))
			mockUserClient := bbmocks.NewMockbbUser(gomock.NewController(t))
			if tt.beforeRepoFn != nil {
				tt.beforeRepoFn(mockRepoClient)
			}

			g := &bitbucket{
				client: &bbClientImpl{
					Repository: mockRepoClient,
					User:       mockUserClient,
				},
			}
			got, err := g.GetDefaultBranch(context.Background(), tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_bitbucket_GetAuthor(t *testing.T) {
	tests := map[string]struct {
		wantUsername string
		wantEmail    string
		wantErr      string
		beforeUserFn func(*bbmocks.MockbbUser)
	}{
		"Should fail with auth failed if user GET returns 403": {
			wantErr: "403 Forbidden",
			beforeUserFn: func(c *bbmocks.MockbbUser) {
				c.EXPECT().Profile().Times(1).
					Return(nil, &bb.UnexpectedResponseStatusError{
						Status: "403 Forbidden",
					})
			},
		},
		"Should fail if user GET returns 404": {
			wantErr: "404 Not Found",
			beforeUserFn: func(c *bbmocks.MockbbUser) {
				c.EXPECT().Profile().Times(1).
					Return(nil, &bb.UnexpectedResponseStatusError{
						Status: "404 Not Found",
					})
			},
		},
		"Should return name and email if available": {
			wantUsername: "name",
			wantEmail:    "name@email",
			beforeUserFn: func(c *bbmocks.MockbbUser) {
				c.EXPECT().Profile().Times(1).Return(&bb.User{
					Username: "name",
				}, nil)

				c.EXPECT().Emails().Times(1).Return(map[string]interface{}{
					"values": []interface{}{
						map[string]interface{}{
							"email": "name@email",
						},
					},
				}, nil)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepoClient := bbmocks.NewMockbbRepo(gomock.NewController(t))
			mockUserClient := bbmocks.NewMockbbUser(gomock.NewController(t))
			if tt.beforeUserFn != nil {
				tt.beforeUserFn(mockUserClient)
			}

			g := &bitbucket{
				client: &bbClientImpl{
					Repository: mockRepoClient,
					User:       mockUserClient,
				},
			}
			gotUsername, gotEmail, err := g.GetAuthor(context.Background())
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.wantUsername, gotUsername, "username mismatch")
			assert.Equal(t, tt.wantEmail, gotEmail, "email mismatch")
		})
	}
}
