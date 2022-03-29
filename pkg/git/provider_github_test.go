package git

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/git/github/mocks"
	"github.com/golang/mock/gomock"

	gh "github.com/google/go-github/v35/github"
	"github.com/stretchr/testify/assert"
)

func Test_github_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		opts     *CreateRepoOptions
		beforeFn func(*mocks.MockRepositories, *mocks.MockUsers)
		want     string
		wantErr  string
	}{
		"Error getting user": {
			opts: &CreateRepoOptions{
				Owner:   "owner",
				Name:    "name",
				Private: false,
			},
			beforeFn: func(mr *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(nil, &gh.Response{Response: &http.Response{
						StatusCode: 200,
					}}, errors.New("Some user error"))
			},
			wantErr: "Some user error",
		},
		"Error creating repo": {
			opts: &CreateRepoOptions{
				Owner:   "owner",
				Name:    "name",
				Private: false,
			},
			beforeFn: func(mr *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(&gh.User{
						Login: gh.String("owner"),
					}, nil, nil)

				mr.EXPECT().Create(context.Background(), "", &gh.Repository{
					Name:    gh.String("name"),
					Private: gh.Bool(false),
				}).
					Times(1).
					Return(nil, &gh.Response{Response: &http.Response{
						StatusCode: 200,
					}}, errors.New("Some repo error"))
			},
			wantErr: "Some repo error",
		},
		"Creates with empty org": {
			opts: &CreateRepoOptions{
				Owner:   "owner",
				Name:    "name",
				Private: false,
			},
			beforeFn: func(mr *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(&gh.User{
						Login: gh.String("owner"),
					}, nil, nil)

				mr.EXPECT().Create(context.Background(), "", &gh.Repository{
					Name:    gh.String("name"),
					Private: gh.Bool(false),
				}).
					Times(1).
					Return(&gh.Repository{
						CloneURL: gh.String("https://github.com/owner/repo"),
					}, &gh.Response{Response: &http.Response{
						StatusCode: 200,
					}}, nil)
			},
			want: "https://github.com/owner/repo",
		},
		"Creates with org": {
			opts: &CreateRepoOptions{
				Owner:   "org",
				Name:    "name",
				Private: false,
			},
			beforeFn: func(mr *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(&gh.User{
						Login: gh.String("owner"),
					}, nil, nil)

				mr.EXPECT().Create(context.Background(), "org", &gh.Repository{
					Name:    gh.String("name"),
					Private: gh.Bool(false),
				}).
					Times(1).
					Return(&gh.Repository{
						CloneURL: gh.String("https://github.com/org/repo"),
					}, &gh.Response{Response: &http.Response{
						StatusCode: 200,
					}}, nil)
			},
			want: "https://github.com/org/repo",
		},
		"Error when no cloneURL": {
			opts: &CreateRepoOptions{
				Owner:   "org",
				Name:    "name",
				Private: false,
			},
			beforeFn: func(mr *mocks.MockRepositories, mu *mocks.MockUsers) {
				mu.EXPECT().Get(context.Background(), "").
					Times(1).
					Return(&gh.User{
						Login: gh.String("owner"),
					}, nil, nil)

				mr.EXPECT().Create(context.Background(), "org", &gh.Repository{
					Name:    gh.String("name"),
					Private: gh.Bool(false),
				}).
					Times(1).
					Return(&gh.Repository{}, &gh.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			wantErr: "repo clone url is nil",
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
			got, err := g.CreateRepository(ctx, tt.opts)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("github.CreateRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}
