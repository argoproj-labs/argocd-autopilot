package git

import (
	"context"
	"fmt"
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
		"Should fail if orgRepo is invalid": {
			orgRepo: "invalid",
			wantErr: "failed parsing organization and repo from 'invalid'",
		},
		"Creates repository under user": {
			orgRepo: "username/repoName",
			want:    "main",
			beforeRepoFn: func(c *bbmocks.MockbbRepo) {
				createOpts := bb.RepositoryOptions{
					Owner:     "username",
					RepoSlug:  "repoName",
					Scm:       "git",
					IsPrivate: "true",
				}

				repo := &bb.Repository{
					Name: "userName",
					Mainbranch: bb.RepositoryBranch{
						Name: "main",
					},
				}

				c.EXPECT().Create(&createOpts).
					Times(1).
					Return(repo, nil)
			},
		},
		"Creates repository under user but cloneUrl doesnt exist": {
			orgRepo: "username/repoName",
			wantErr: "failed creating the repository \"repoName\" under \"username\": clone url is empty",
			beforeRepoFn: func(c *bbmocks.MockbbRepo) {
				createOpts := bb.RepositoryOptions{
					Owner:     "username",
					RepoSlug:  "repoName",
					Scm:       "git",
					IsPrivate: "true",
				}

				c.EXPECT().Create(&createOpts).
					Times(1).
					Return(nil, fmt.Errorf("clone url is empty"))
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
			if tt.beforeRepoFn != nil {
				tt.beforeRepoFn(mockRepoClient)
			}

			g := &bitbucket{
				Repository: mockRepoClient,
				User:       mockUserClient,
			}
			got, err := g.CreateRepository(context.Background(), tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equalf(t, tt.want, got, "CreateRepository - %s", name)
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
				Repository: mockRepoClient,
				User:       mockUserClient,
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
		"Should fail if user GET returns 404": {
			wantErr: "404 Not Found",
			beforeUserFn: func(c *bbmocks.MockbbUser) {
				c.EXPECT().Profile().Times(1).
					Return(nil, &bb.UnexpectedResponseStatusError{
						Status: "404 Not Found",
					})
			},
		},
		"Should return name and email (primary and confirmed) if available": {
			wantUsername: "name",
			wantEmail:    "name@email",
			beforeUserFn: func(c *bbmocks.MockbbUser) {
				c.EXPECT().Profile().Times(1).Return(&bb.User{
					Username: "name",
				}, nil)

				c.EXPECT().Emails().Times(1).Return(map[string]interface{}{
					"values": []interface{}{
						map[string]interface{}{
							"email":        "name2@email",
							"is_primary":   false,
							"is_confirmed": true,
						},
						map[string]interface{}{
							"email":        "name@email",
							"is_primary":   true,
							"is_confirmed": true,
						},
						map[string]interface{}{
							"email":        "name3@email",
							"is_primary":   false,
							"is_confirmed": false,
						},
					},
				}, nil)
			},
		},
		"Should return name and confirmed email in case primary not exist": {
			wantUsername: "name",
			wantEmail:    "name@email",
			beforeUserFn: func(c *bbmocks.MockbbUser) {
				c.EXPECT().Profile().Times(1).Return(&bb.User{
					Username: "name",
				}, nil)

				c.EXPECT().Emails().Times(1).Return(map[string]interface{}{
					"values": []interface{}{
						map[string]interface{}{
							"email":        "name2@email",
							"is_primary":   false,
							"is_confirmed": false,
						},
						map[string]interface{}{
							"email":        "name@email",
							"is_primary":   false,
							"is_confirmed": true,
						},
					},
				}, nil)
			},
		},
		"Should return name and name as email in case no primary or confirmed email exist": {
			wantUsername: "name",
			wantEmail:    "name",
			beforeUserFn: func(c *bbmocks.MockbbUser) {
				c.EXPECT().Profile().Times(1).Return(&bb.User{
					Username: "name",
				}, nil)

				c.EXPECT().Emails().Times(1).Return(map[string]interface{}{
					"values": []interface{}{
						map[string]interface{}{
							"email":        "name2@email",
							"is_primary":   false,
							"is_confirmed": false,
						},
						map[string]interface{}{
							"email":        "name@email",
							"is_primary":   false,
							"is_confirmed": false,
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
				Repository: mockRepoClient,
				User:       mockUserClient,
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
