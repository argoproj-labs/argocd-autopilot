package git

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/git/github/mocks"

	gh "github.com/google/go-github/v35/github"
	"github.com/stretchr/testify/assert"
)

func Test_github_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		user    *gh.User
		userErr error
		repo    *gh.Repository
		repoErr error
		opts    *CreateRepoOptions
		org     string
		want    string
		wantErr string
	}{
		"Error getting user": {
			userErr: errors.New("Some user error"),
			opts: &CreateRepoOptions{
				Owner:   "owner",
				Name:    "name",
				Private: false,
			},
			wantErr: "Some user error",
		},
		"Error creating repo": {
			user: &gh.User{
				Login: gh.String("owner"),
			},
			opts: &CreateRepoOptions{
				Owner:   "owner",
				Name:    "name",
				Private: false,
			},
			repoErr: errors.New("Some repo error"),
			wantErr: "Some repo error",
		},
		"Creates with empty org": {
			user: &gh.User{
				Login: gh.String("owner"),
			},
			opts: &CreateRepoOptions{
				Owner:   "owner",
				Name:    "name",
				Private: false,
			},
			repo: &gh.Repository{
				CloneURL: gh.String("https://github.com/owner/repo"),
			},
			want: "https://github.com/owner/repo",
		},
		"Creates with org": {
			user: &gh.User{
				Login: gh.String("owner"),
			},
			opts: &CreateRepoOptions{
				Owner:   "org",
				Name:    "name",
				Private: false,
			},
			org: "org",
			repo: &gh.Repository{
				CloneURL: gh.String("https://github.com/org/repo"),
			},
			want: "https://github.com/org/repo",
		},
		"Error when no cloneURL": {
			user: &gh.User{
				Login: gh.String("owner"),
			},
			opts: &CreateRepoOptions{
				Owner:   "org",
				Name:    "name",
				Private: false,
			},
			org:     "org",
			repo:    &gh.Repository{},
			wantErr: "repo clone url is nil",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockUsers := new(mocks.Users)
			mockRepo := new(mocks.Repositories)
			ctx := context.Background()

			mockUsers.On("Get", ctx, "").Return(tt.user, &gh.Response{Response: &http.Response{
				StatusCode: 200,
			}}, tt.userErr)

			mockRepo.On("Create", ctx, tt.org, &gh.Repository{
				Name:    gh.String(tt.opts.Name),
				Private: gh.Bool(tt.opts.Private),
			}).Return(tt.repo, &gh.Response{Response: &http.Response{
				StatusCode: 200,
			}}, tt.repoErr)

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
