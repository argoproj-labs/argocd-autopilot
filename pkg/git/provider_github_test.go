package git

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/git/mocks"
	gh "github.com/google/go-github/v34/github"
	"github.com/stretchr/testify/assert"
)

func Test_github_GetRepository(t *testing.T) {
	tests := map[string]struct {
		repo    *gh.Repository
		res     *gh.Response
		err     error
		opts    *GetRepoOptions
		want    string
		wantErr string
	}{
		"No repo": {
			res: &gh.Response{
				Response: &http.Response{
					StatusCode: http.StatusNotFound,
				},
			},
			opts: &GetRepoOptions{
				Owner: "owner",
				Name:  "repo",
			},
			want:    "",
			wantErr: "git repository not found",
		},
		"Has repo": {
			repo: &gh.Repository{
				CloneURL: gh.String("https://github.com/owner/repo"),
			},
			opts: &GetRepoOptions{
				Owner: "owner",
				Name:  "repo",
			},
			want:    "https://github.com/owner/repo",
			wantErr: "",
		},
		"Error getting repo": {
			err: errors.New("Some error"),
			opts: &GetRepoOptions{
				Owner: "owner",
				Name:  "repo",
			},
			want:    "",
			wantErr: "Some error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := new(mocks.Repositories)
			ctx := context.Background()
			mockRepo.On("Get", ctx, tt.opts.Owner, tt.opts.Name).Return(tt.repo, tt.res, tt.err)
			g := &github{
				Repositories: mockRepo,
			}
			got, err := g.GetRepository(ctx, tt.opts)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("github.GetRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
			mockUsers.On("Get", ctx, "").Return(tt.user, nil, tt.userErr)
			mockRepo.On("Create", ctx, tt.org, &gh.Repository{
				Name:    gh.String(tt.opts.Name),
				Private: gh.Bool(tt.opts.Private),
			}).Return(tt.repo, nil, tt.repoErr)
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
