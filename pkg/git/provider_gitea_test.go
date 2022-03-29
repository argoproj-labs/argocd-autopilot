package git

import (
	"context"
	"errors"
	"net/http"
	"testing"

	gtmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/gitea/mocks"
	"github.com/golang/mock/gomock"

	gt "code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
)

func Test_gitea_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		opts     *CreateRepoOptions
		beforeFn func(*gtmocks.MockClient)
		want     string
		wantErr  string
	}{
		"Should fail if credentials are wrong": {
			opts: &CreateRepoOptions{
				Name:  "repo",
				Owner: "username",
			},
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
			opts: &CreateRepoOptions{
				Name:  "repo",
				Owner: "username",
			},
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
			opts: &CreateRepoOptions{
				Name:  "repo",
				Owner: "org",
			},
			beforeFn: func(c *gtmocks.MockClient) {
				u := &gt.User{UserName: "username"}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(u, nil, nil)
				createOpts := gt.CreateRepoOption{
					Name:    "repo",
					Private: false,
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
		"Should fail repo creation fails": {
			opts: &CreateRepoOptions{
				Name:  "repo",
				Owner: "username",
			},
			beforeFn: func(c *gtmocks.MockClient) {
				u := &gt.User{UserName: "username"}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(u, nil, nil)
				createOpts := gt.CreateRepoOption{
					Name:    "repo",
					Private: false,
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
		"Should create a simple org/repo repository": {
			opts: &CreateRepoOptions{
				Name:  "repo",
				Owner: "org",
			},
			beforeFn: func(c *gtmocks.MockClient) {
				u := &gt.User{UserName: "username"}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(u, nil, nil)
				r := &gt.Repository{
					CloneURL: "http://gitea.com/org/repo",
				}
				createOpts := gt.CreateRepoOption{
					Name:    "repo",
					Private: false,
				}
				c.EXPECT().CreateOrgRepo("org", createOpts).
					Times(1).
					Return(r, nil, nil)
			},
			want: "http://gitea.com/org/repo",
		},
		"Should create a simple username/repo repository": {
			opts: &CreateRepoOptions{
				Name:  "repo",
				Owner: "username",
			},
			beforeFn: func(c *gtmocks.MockClient) {
				u := &gt.User{UserName: "username"}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(u, nil, nil)
				r := &gt.Repository{
					CloneURL: "http://gitea.com/username/repo",
				}
				createOpts := gt.CreateRepoOption{
					Name:    "repo",
					Private: false,
				}
				c.EXPECT().CreateRepo(createOpts).
					Times(1).
					Return(r, nil, nil)
			},
			want: "http://gitea.com/username/repo",
		},
		"Should create a private repository": {
			opts: &CreateRepoOptions{
				Name:    "repo",
				Owner:   "username",
				Private: true,
			},
			beforeFn: func(c *gtmocks.MockClient) {
				u := &gt.User{UserName: "username"}
				c.EXPECT().GetMyUserInfo().
					Times(1).
					Return(u, nil, nil)
				r := &gt.Repository{
					CloneURL: "http://gitea.com/username/repo",
				}
				createOpts := gt.CreateRepoOption{
					Name:    "repo",
					Private: true,
				}
				c.EXPECT().CreateRepo(createOpts).
					Times(1).
					Return(r, nil, nil)
			},
			want: "http://gitea.com/username/repo",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := gtmocks.NewMockClient(gomock.NewController(t))
			tt.beforeFn(mockClient)
			g := &gitea{
				client: mockClient,
			}
			got, err := g.CreateRepository(context.Background(), tt.opts)

			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("gitea.CreateRepository() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if got != tt.want {
				t.Errorf("gitea.CreateRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}
