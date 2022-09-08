package git

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/git/github/mocks"
	"github.com/golang/mock/gomock"
	gh "github.com/google/go-github/v43/github"
	"github.com/stretchr/testify/assert"
)

func Test_github_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		orgRepo  string
		beforeFn func(*mocks.MockRepositories, *mocks.MockUsers)
		want     string
		wantErr  string
	}{
		"Error getting user": {
			orgRepo: "owner/name",
			beforeFn: func(_ *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(nil, &gh.Response{Response: &http.Response{
						StatusCode: 200,
					}}, errors.New("Some user error"))
			},
			wantErr: "Some user error",
		},
		"Error creating repo": {
			orgRepo: "owner/name",
			beforeFn: func(mr *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(&gh.User{
						Login: gh.String("owner"),
					}, nil, nil)

				mr.EXPECT().Create(context.Background(), "", &gh.Repository{
					Name:    gh.String("name"),
					Private: gh.Bool(true),
				}).
					Times(1).
					Return(nil, &gh.Response{Response: &http.Response{
						StatusCode: 200,
					}}, errors.New("Some repo error"))
			},
			wantErr: "Some repo error",
		},
		"Creates with empty org": {
			orgRepo: "owner/name",
			beforeFn: func(mr *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(&gh.User{
						Login: gh.String("owner"),
					}, nil, nil)

				repo := &gh.Repository{
					DefaultBranch: gh.String("main"),
				}
				res := &gh.Response{Response: &http.Response{
					StatusCode: 200,
				}}
				mr.EXPECT().Create(context.Background(), "", &gh.Repository{
					Name:    gh.String("name"),
					Private: gh.Bool(true),
				}).
					Times(1).
					Return(repo, res, nil)
			},
			want: "main",
		},
		"Creates with org": {
			orgRepo: "org/name",
			beforeFn: func(mr *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(&gh.User{
						Login: gh.String("owner"),
					}, nil, nil)

				repo := &gh.Repository{
					DefaultBranch: gh.String("main"),
				}
				res := &gh.Response{Response: &http.Response{
					StatusCode: 200,
				}}
				mr.EXPECT().Create(context.Background(), "org", &gh.Repository{
					Name:    gh.String("name"),
					Private: gh.Bool(true),
				}).
					Times(1).
					Return(repo, res, nil)
			},
			want: "main",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUsers := mocks.NewMockUsers(ctrl)
			mockRepo := mocks.NewMockRepositories(ctrl)
			ctx := context.Background()

			tt.beforeFn(mockRepo, mockUsers)

			g := &github{
				Repositories: mockRepo,
				Users:        mockUsers,
			}
			got, err := g.CreateRepository(ctx, tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equalf(t, tt.want, got, "CreateRepository - %s", name)
		})
	}
}

func Test_github_GetDefaultBranch(t *testing.T) {
	tests := map[string]struct {
		orgRepo  string
		want     string
		wantErr  string
		beforeFn func(*mocks.MockRepositories)
	}{
		"Should fail if orgRepo is invalid": {
			orgRepo: "invalid",
			wantErr: "failed parsing organization and repo from 'invalid'",
		},
		"Should fail if repo Get fails with 401": {
			orgRepo: "owner/repo",
			wantErr: "some error",
			beforeFn: func(mc *mocks.MockRepositories) {
				res := &gh.Response{
					Response: &http.Response{
						StatusCode: 401,
					},
				}
				mc.EXPECT().Get(context.Background(), "owner", "repo").Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should fail if repo Get fails with 404": {
			orgRepo: "owner/repo",
			wantErr: "owner owner not found: some error",
			beforeFn: func(mc *mocks.MockRepositories) {
				res := &gh.Response{
					Response: &http.Response{
						StatusCode: 404,
					},
				}
				mc.EXPECT().Get(context.Background(), "owner", "repo").Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should succeed with valid default branch": {
			orgRepo: "owner/repo",
			want:    "main",
			beforeFn: func(mr *mocks.MockRepositories) {
				r := &gh.Repository{
					DefaultBranch: gh.String("main"),
				}
				mr.EXPECT().Get(context.Background(), "owner", "repo").Times(1).Return(r, nil, nil)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockRepositories(ctrl)
			if tt.beforeFn != nil {
				tt.beforeFn(mockRepo)
			}

			g := &github{
				Repositories: mockRepo,
			}
			got, err := g.GetDefaultBranch(context.Background(), tt.orgRepo)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_github_GetAuthor(t *testing.T) {
	tests := map[string]struct {
		wantUsername string
		wantEmail    string
		wantErr      string
		beforeFn     func(*mocks.MockUsers)
	}{
		"Should fail if Uset Get fails with 401": {
			wantErr: "authentication failed, make sure credentials are correct: some error",
			beforeFn: func(mu *mocks.MockUsers) {
				res := &gh.Response{
					Response: &http.Response{
						StatusCode: 401,
					},
				}
				mu.EXPECT().Get(gomock.Any(), "").Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should fail if Uset Get fails with 404": {
			wantErr: "some error",
			beforeFn: func(mu *mocks.MockUsers) {
				res := &gh.Response{
					Response: &http.Response{
						StatusCode: 404,
					},
				}
				mu.EXPECT().Get(gomock.Any(), "").Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should fail if ListEmails fail": {
			wantUsername: "name",
			wantEmail:    "login",
			beforeFn: func(mu *mocks.MockUsers) {
				u := &gh.User{
					Name:  gh.String("name"),
					Login: gh.String("login"),
				}
				mu.EXPECT().Get(gomock.Any(), "").Times(1).Return(u, nil, nil)
				mu.EXPECT().ListEmails(gomock.Any(), &gh.ListOptions{
					Page:    0,
					PerPage: 10,
				}).Times(1).Return(nil, nil, errors.New("some error"))
			},
		},
		"Should succeed with user name": {
			wantUsername: "name",
			wantEmail:    "username@email",
			beforeFn: func(mu *mocks.MockUsers) {
				u := &gh.User{
					Name:  gh.String("name"),
					Email: gh.String("username@email"),
				}
				mu.EXPECT().Get(gomock.Any(), "").Times(1).Return(u, nil, nil)
			},
		},
		"Should succeed with user login": {
			wantUsername: "login",
			wantEmail:    "username@email",
			beforeFn: func(mu *mocks.MockUsers) {
				u := &gh.User{
					Login: gh.String("login"),
					Email: gh.String("username@email"),
				}
				mu.EXPECT().Get(gomock.Any(), "").Times(1).Return(u, nil, nil)
			},
		},
		"Should succeed with user primary and verified email": {
			wantUsername: "name",
			wantEmail:    "primary-verified@email",
			beforeFn: func(mu *mocks.MockUsers) {
				u := &gh.User{
					Name: gh.String("name"),
				}
				mu.EXPECT().Get(gomock.Any(), "").Times(1).Return(u, nil, nil)
				emails := []*gh.UserEmail{
					{
						Visibility: gh.String("public"),
						Primary:    gh.Bool(true),
						Verified:   gh.Bool(true),
						Email:      gh.String("primary-verified@email"),
					},
				}
				mu.EXPECT().ListEmails(gomock.Any(), &gh.ListOptions{
					Page:    0,
					PerPage: 10,
				}).Times(1).Return(emails, nil, nil)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUsers := mocks.NewMockUsers(ctrl)
			if tt.beforeFn != nil {
				tt.beforeFn(mockUsers)
			}

			g := &github{
				Users: mockUsers,
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

func Test_github_getEmail(t *testing.T) {
	tests := map[string]struct {
		want     string
		beforeFn func(*mocks.MockUsers)
	}{
		"Should return empty string when ListEmails fails": {
			want: "",
			beforeFn: func(mu *mocks.MockUsers) {
				mu.EXPECT().ListEmails(gomock.Any(), &gh.ListOptions{
					Page:    0,
					PerPage: 10,
				}).Times(1).Return(nil, nil, errors.New("some error"))
			},
		},
		"Should return public primary-verified email": {
			want: "primary-verified@email",
			beforeFn: func(mu *mocks.MockUsers) {
				emails := []*gh.UserEmail{
					{
						Visibility: gh.String("private"),
						Primary:    gh.Bool(true),
						Verified:   gh.Bool(true),
						Email:      gh.String("private@email"),
					},
					{
						Visibility: gh.String("public"),
						Primary:    gh.Bool(true),
						Verified:   gh.Bool(false),
						Email:      gh.String("primary@email"),
					},
					{
						Visibility: gh.String("public"),
						Primary:    gh.Bool(false),
						Verified:   gh.Bool(true),
						Email:      gh.String("verified@email"),
					},
					{
						Visibility: gh.String("public"),
						Primary:    gh.Bool(true),
						Verified:   gh.Bool(true),
						Email:      gh.String("primary-verified@email"),
					},
				}
				mu.EXPECT().ListEmails(gomock.Any(), &gh.ListOptions{
					Page:    0,
					PerPage: 10,
				}).Times(1).Return(emails, nil, nil)
			},
		},
		"Should return public primary email when no primary-verified exist": {
			want: "primary@email",
			beforeFn: func(mu *mocks.MockUsers) {
				emails := []*gh.UserEmail{
					{
						Visibility: gh.String("private"),
						Primary:    gh.Bool(true),
						Verified:   gh.Bool(true),
						Email:      gh.String("private@email"),
					},
					{
						Visibility: gh.String("public"),
						Primary:    gh.Bool(true),
						Verified:   gh.Bool(false),
						Email:      gh.String("primary@email"),
					},
					{
						Visibility: gh.String("public"),
						Primary:    gh.Bool(false),
						Verified:   gh.Bool(true),
						Email:      gh.String("verified@email"),
					},
				}
				mu.EXPECT().ListEmails(gomock.Any(), &gh.ListOptions{
					Page:    0,
					PerPage: 10,
				}).Times(1).Return(emails, nil, nil)
			},
		},
		"Should return public verified email when no primary exist": {
			want: "verified@email",
			beforeFn: func(mu *mocks.MockUsers) {
				emails := []*gh.UserEmail{
					{
						Visibility: gh.String("private"),
						Primary:    gh.Bool(true),
						Verified:   gh.Bool(true),
						Email:      gh.String("private@email"),
					},
					{
						Visibility: gh.String("public"),
						Primary:    gh.Bool(false),
						Verified:   gh.Bool(true),
						Email:      gh.String("verified@email"),
					},
				}
				mu.EXPECT().ListEmails(gomock.Any(), &gh.ListOptions{
					Page:    0,
					PerPage: 10,
				}).Times(1).Return(emails, nil, nil)
			},
		},
		"Should return empty string email when no valid email exist": {
			want: "",
			beforeFn: func(mu *mocks.MockUsers) {
				emails := []*gh.UserEmail{
					{
						Visibility: gh.String("private"),
						Primary:    gh.Bool(true),
						Verified:   gh.Bool(true),
						Email:      gh.String("private@email"),
					},
					{
						Visibility: gh.String("public"),
						Primary:    gh.Bool(false),
						Verified:   gh.Bool(false),
						Email:      gh.String("verified@email"),
					},
				}
				mu.EXPECT().ListEmails(gomock.Any(), &gh.ListOptions{
					Page:    0,
					PerPage: 10,
				}).Times(1).Return(emails, nil, nil)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockUsers := mocks.NewMockUsers(ctrl)
			if tt.beforeFn != nil {
				tt.beforeFn(mockUsers)
			}

			g := &github{
				Users: mockUsers,
			}
			if got := g.getEmail(context.Background()); got != tt.want {
				t.Errorf("github.getEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}
