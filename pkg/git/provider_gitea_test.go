package git

import (
	"context"
	"errors"
	"net/http"
	"testing"

	gt "code.gitea.io/sdk/gitea"
	gtmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/gitea/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_gitea_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		orgRepo  string
		beforeFn func(*gtmocks.MockClient)
		want     string
		wantErr  string
	}{
		"Should fail if credentials are wrong": {
			orgRepo: "username/repo",
			beforeFn: func(c *gtmocks.MockClient) {
				res := &gt.Response{
					Response: &http.Response{
						StatusCode: 401,
					},
				}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(nil, res, errors.New("some error"))
			},
			wantErr: "authentication failed, make sure credentials are correct: some error",
		},
		"Should fail if can't get user info": {
			orgRepo: "username/repo",
			beforeFn: func(c *gtmocks.MockClient) {
				res := &gt.Response{
					Response: &http.Response{},
				}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(nil, res, errors.New("some error"))
			},
			wantErr: "some error",
		},
		"Should fail if owner is not found": {
			orgRepo: "org/repo",
			beforeFn: func(c *gtmocks.MockClient) {
				u := &gt.User{UserName: "username"}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(u, nil, nil)
				createOpts := gt.CreateRepoOption{
					Name:    "repo",
					Private: true,
				}
				res := &gt.Response{
					Response: &http.Response{
						StatusCode: 404,
					},
				}
				c.EXPECT().CreateOrgRepo("org", createOpts).
					Times(1).
					Return(nil, res, errors.New("some error"))
			},
			wantErr: "owner org not found: some error",
		},
		"Should fail if repo creation fails": {
			orgRepo: "username/repo",
			beforeFn: func(c *gtmocks.MockClient) {
				u := &gt.User{UserName: "username"}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(u, nil, nil)
				createOpts := gt.CreateRepoOption{
					Name:    "repo",
					Private: true,
				}
				res := &gt.Response{
					Response: &http.Response{},
				}
				c.EXPECT().CreateRepo(createOpts).
					Times(1).
					Return(nil, res, errors.New("some error"))
			},
			wantErr: "some error",
		},
		"Should create a private repository": {
			orgRepo: "username/repo",
			beforeFn: func(c *gtmocks.MockClient) {
				u := &gt.User{UserName: "username"}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(u, nil, nil)
				r := &gt.Repository{
					DefaultBranch: "main",
				}
				createOpts := gt.CreateRepoOption{
					Name:    "repo",
					Private: true,
				}
				c.EXPECT().CreateRepo(createOpts).
					Times(1).
					Return(r, nil, nil)
			},
			want: "main",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := gtmocks.NewMockClient(gomock.NewController(t))
			tt.beforeFn(mockClient)
			g := &gitea{
				client: mockClient,
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

func Test_gitea_GetDefaultBranch(t *testing.T) {
	tests := map[string]struct {
		orgRepo  string
		want     string
		wantErr  string
		beforeFn func(*gtmocks.MockClient)
	}{
		"Should fails if orgRepo is invalid": {
			orgRepo: "invalid",
			wantErr: "failed parsing organization and repo from 'invalid'",
		},
		"Should fail if GetRepo fails with 401": {
			orgRepo: "owner/repo",
			wantErr: "some error",
			beforeFn: func(mc *gtmocks.MockClient) {
				res := &gt.Response{
					Response: &http.Response{
						StatusCode: 401,
					},
				}
				mc.EXPECT().GetRepo("owner", "repo").Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should fail if GetRepo fails with 404": {
			orgRepo: "owner/repo",
			wantErr: "owner owner not found: some error",
			beforeFn: func(mc *gtmocks.MockClient) {
				res := &gt.Response{
					Response: &http.Response{
						StatusCode: 404,
					},
				}
				mc.EXPECT().GetRepo("owner", "repo").Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should succeed with valid default branch": {
			orgRepo: "owner/repo",
			want:    "main",
			beforeFn: func(client *gtmocks.MockClient) {
				r := &gt.Repository{
					DefaultBranch: "main",
				}
				client.EXPECT().GetRepo("owner", "repo").Times(1).Return(r, nil, nil)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := gtmocks.NewMockClient(gomock.NewController(t))
			if tt.beforeFn != nil {
				tt.beforeFn(mockClient)
			}

			g := &gitea{
				client: mockClient,
			}
			got, err := g.GetDefaultBranch(context.Background(), tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("gitea.GetDefaultBranch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_gitea_GetAuthor(t *testing.T) {
	tests := map[string]struct {
		wantUsername string
		wantEmail    string
		wantErr      string
		beforeFn     func(*gtmocks.MockClient)
	}{
		"Should fail if GetMyUserInfo fails with 401": {
			wantErr: "authentication failed, make sure credentials are correct: some error",
			beforeFn: func(mc *gtmocks.MockClient) {
				res := &gt.Response{
					Response: &http.Response{
						StatusCode: 401,
					},
				}
				mc.EXPECT().GetMyUserInfo().Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should fail if GetMyUserInfo fails with 404": {
			wantErr: "some error",
			beforeFn: func(mc *gtmocks.MockClient) {
				res := &gt.Response{
					Response: &http.Response{
						StatusCode: 404,
					},
				}
				mc.EXPECT().GetMyUserInfo().Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should succeed with valid user": {
			wantUsername: "user",
			wantEmail:    "user@email",
			beforeFn: func(mc *gtmocks.MockClient) {
				user := gt.User{
					UserName: "user",
					Email:    "user@email",
				}
				mc.EXPECT().GetMyUserInfo().Times(1).Return(&user, nil, nil)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := gtmocks.NewMockClient(gomock.NewController(t))
			if tt.beforeFn != nil {
				tt.beforeFn(mockClient)
			}

			g := &gitea{
				client: mockClient,
			}
			gotUsername, gotEmail, err := g.GetAuthor(context.Background())
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.wantUsername, gotUsername, "gitea.GetDefaultBranch() username mismatch")
			assert.Equal(t, tt.wantEmail, gotEmail, "gitea.GetDefaultBranch() email mismatch")
		})
	}
}
