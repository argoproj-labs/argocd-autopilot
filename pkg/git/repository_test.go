package git

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/git/gogit"
	"github.com/argoproj/argocd-autopilot/pkg/git/gogit/mocks"
	billy "github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_repo_addRemote(t *testing.T) {
	type args struct {
		name string
		url  string
	}
	tests := map[string]struct {
		args        args
		expectedCfg *config.RemoteConfig
		retErr      error
		wantErr     bool
	}{
		"Basic": {
			args: args{
				name: "test",
				url:  "http://test",
			},
			expectedCfg: &config.RemoteConfig{
				Name: "test",
				URLs: []string{"http://test"},
			},
			wantErr: false,
		},
		"Error": {
			args: args{
				name: "test",
				url:  "http://test",
			},
			expectedCfg: &config.RemoteConfig{
				Name: "test",
				URLs: []string{"http://test"},
			},
			retErr:  fmt.Errorf("error"),
			wantErr: true,
		},
	}

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockRepo := &mocks.Repository{}
			mockRepo.On("CreateRemote", mock.Anything).Return(nil, tt.retErr)

			r := &repo{Repository: mockRepo}
			if err := r.addRemote(tt.args.name, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("repo.addRemote() error = %v, wantErr %v", err, tt.wantErr)
			}

			mockRepo.AssertCalled(t, "CreateRemote", mock.Anything)

			actualCfg := mockRepo.Calls[0].Arguments.Get(0).(*config.RemoteConfig)
			assert.Equal(t, tt.expectedCfg.Name, actualCfg.Name)
		})
	}
}

func Test_getAuth(t *testing.T) {
	tests := map[string]struct {
		auth *Auth
		want transport.AuthMethod
	}{
		"Basic": {
			auth: &Auth{
				Password: "123",
			},
			want: &http.BasicAuth{
				Username: "git",
				Password: "123",
			},
		},
		"Username": {
			auth: &Auth{
				Username: "test",
				Password: "123",
			},
			want: &http.BasicAuth{
				Username: "test",
				Password: "123",
			},
		},
		"nil": {
			auth: nil,
			want: nil,
		},
	}
	t.Parallel()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			if got := getAuth(tt.auth); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_repo_initBranch(t *testing.T) {
	type args struct {
		ctx        context.Context
		branchName string
	}
	tests := map[string]struct {
		args     args
		wantErr  bool
		retErr   error
		assertFn func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree)
	}{
		"Init current branch": {
			args: args{
				ctx:        context.Background(),
				branchName: "",
			},
			assertFn: func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree) {
				r.AssertNotCalled(t, "Worktree")
				wt.AssertCalled(t, "Commit", "initial commit", mock.Anything)
				wt.AssertNotCalled(t, "Checkout")
			},
		},
		"Init and checkout branch": {
			args: args{
				ctx: context.Background(),
			},
			assertFn: func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree) {
				wt.AssertCalled(t, "Commit", "initial commit", mock.Anything)
				b := plumbing.NewBranchReferenceName("test")
				wt.AssertCalled(t, "Checkout", &gg.CheckoutOptions{
					Branch: b,
					Create: true,
				})
			},
		},
		"Error": {
			args: args{
				ctx:        context.Background(),
				branchName: "test",
			},
			wantErr: true,
			retErr:  fmt.Errorf("error"),
			assertFn: func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree) {
				wt.AssertCalled(t, "Commit", "initial commit", mock.Anything)
				wt.AssertNotCalled(t, "Checkout")
			},
		},
	}

	orgWorktree := worktree
	defer func() { worktree = orgWorktree }()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			mockRepo := &mocks.Repository{}
			mockWt := &mocks.Worktree{}
			mockWt.On("Commit", mock.Anything, mock.Anything).Return(nil, tt.retErr)
			mockWt.On("Checkout", mock.Anything).Return(tt.retErr)

			worktree = func(r gogit.Repository) (gogit.Worktree, error) { return mockWt, nil }

			r := &repo{Repository: mockRepo}

			if err := r.initBranch(tt.args.ctx, tt.args.branchName); (err != nil) != tt.wantErr {
				t.Errorf("repo.checkout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_initRepo(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts *CloneOptions
	}
	tests := map[string]struct {
		args     args
		want     Repository
		wantErr  bool
		retErr   error
		assertFn func(t *testing.T, r *mocks.Repository, w *mocks.Worktree)
	}{
		"Basic": {
			args: args{
				ctx: context.Background(),
				opts: &CloneOptions{
					URL:      "http://test",
					Revision: "test",
				},
			},
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree) {
				r.AssertCalled(t, "CreateRemote")
				w.AssertCalled(t, "Commit")
				w.AssertCalled(t, "Commit")
			},
		},
		"Error": {
			args: args{
				ctx: context.Background(),
				opts: &CloneOptions{
					URL:      "http://test",
					Revision: "test",
				},
			},
			retErr:  fmt.Errorf("error"),
			wantErr: true,
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree) {
				r.AssertCalled(t, "CreateRemote", mock.Anything)
				w.AssertNotCalled(t, "Commit")
				w.AssertNotCalled(t, "Commit")
			},
		},
	}

	orgInitRepo := ggInitRepo
	defer func() { ggInitRepo = orgInitRepo }()
	orgWorktree := worktree
	defer func() { worktree = orgWorktree }()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockRepo := &mocks.Repository{}
			mockRepo.On("CreateRemote", mock.Anything).Return(nil, tt.retErr)
			mockWt := &mocks.Worktree{}
			mockWt.On("Commit", mock.Anything, mock.Anything).Return(nil, tt.retErr)
			mockWt.On("Checkout", mock.Anything).Return(tt.retErr)

			ggInitRepo = func(s storage.Storer, worktree billy.Filesystem) (gogit.Repository, error) { return mockRepo, nil }
			worktree = func(r gogit.Repository) (gogit.Worktree, error) { return mockWt, nil }

			got, err := initRepo(tt.args.ctx, tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("initRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_clone(t *testing.T) {
	type args struct {
		ctx  context.Context
		fs   billy.Filesystem
		opts *CloneOptions
	}
	tests := map[string]struct {
		args         args
		retErr       error
		wantErr      bool
		assertFn     func(*testing.T, *repo)
		expectedOpts *gg.CloneOptions
	}{
		"NilOpts": {
			args: args{
				ctx:  context.Background(),
				fs:   nil,
				opts: nil,
			},
			wantErr: true,
			assertFn: func(t *testing.T, r *repo) {
				assert.Nil(t, r)
			},
		},
		"No Auth": {
			args: args{
				ctx: context.Background(),
				fs:  nil,
				opts: &CloneOptions{
					URL: "https://test",
				},
			},
			expectedOpts: &gg.CloneOptions{
				URL:          "https://test",
				Auth:         nil,
				SingleBranch: true,
				Depth:        1,
				Progress:     os.Stderr,
				Tags:         gg.NoTags,
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.NotNil(t, r)
			},
		},
		"With Auth": {
			args: args{
				ctx: context.Background(),
				fs:  nil,
				opts: &CloneOptions{
					URL: "https://test",
					Auth: &Auth{
						Username: "asd",
						Password: "123",
					},
				},
			},
			expectedOpts: &gg.CloneOptions{
				URL: "https://test",
				Auth: &http.BasicAuth{
					Username: "asd",
					Password: "123",
				},
				SingleBranch: true,
				Depth:        1,
				Progress:     os.Stderr,
				Tags:         gg.NoTags,
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.NotNil(t, r)
			},
		},
		"Error": {
			args: args{
				ctx: context.Background(),
				fs:  nil,
				opts: &CloneOptions{
					URL: "https://test",
				},
			},
			expectedOpts: &gg.CloneOptions{
				URL:          "https://test",
				SingleBranch: true,
				Depth:        1,
				Progress:     os.Stderr,
				Tags:         gg.NoTags,
			},
			retErr:  fmt.Errorf("error"),
			wantErr: true,
			assertFn: func(t *testing.T, r *repo) {
				assert.Nil(t, r)
			},
		},
		"With Revision": {
			args: args{
				ctx: context.Background(),
				fs:  nil,
				opts: &CloneOptions{
					URL:      "https://test",
					Revision: "test",
				},
			},
			expectedOpts: &gg.CloneOptions{
				URL:           "https://test",
				SingleBranch:  true,
				Depth:         1,
				Progress:      os.Stderr,
				Tags:          gg.NoTags,
				ReferenceName: plumbing.NewBranchReferenceName("test"),
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.NotNil(t, r)
			},
		},
	}

	orgClone := ggClone
	defer func() { ggClone = orgClone }()

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockRepo := &mocks.Repository{}
			ggClone = func(ctx context.Context, s storage.Storer, worktree billy.Filesystem, o *gg.CloneOptions) (gogit.Repository, error) {
				if tt.expectedOpts != nil {
					assert.True(t, reflect.DeepEqual(o, tt.expectedOpts), "opts not equal")
				}
				if tt.retErr != nil {
					return nil, tt.retErr
				}
				return mockRepo, nil
			}

			got, err := clone(tt.args.ctx, tt.args.fs, tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("clone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.assertFn(t, got)
		})
	}
}

func TestClone(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts *CloneOptions
	}
	tests := map[string]struct {
		args             args
		wantErr          bool
		cloneErr         error
		initErr          error
		expectInitCalled bool
		assertFn         func(*testing.T, Repository)
	}{
		"No error": {
			args: args{
				ctx: context.Background(),
				opts: &CloneOptions{
					URL: "http://test",
				},
			},
			assertFn: func(t *testing.T, r Repository) {
				assert.NotNil(t, r)
			},
			expectInitCalled: false,
		},
		"NilOpts": {
			args: args{
				ctx:  context.Background(),
				opts: nil,
			},
			assertFn: func(t *testing.T, r Repository) {
				assert.Nil(t, r)
			},
			wantErr: true,
		},
		"EmptyRepo": {
			args: args{
				ctx: context.Background(),
				opts: &CloneOptions{
					URL: "http://test",
				},
			},
			assertFn: func(t *testing.T, r Repository) {
				assert.NotNil(t, r)
			},
			cloneErr:         transport.ErrEmptyRemoteRepository,
			wantErr:          false,
			expectInitCalled: true,
		},
		"AnotherErr": {
			args: args{
				ctx: context.Background(),
				opts: &CloneOptions{
					URL: "http://test",
				},
			},
			assertFn: func(t *testing.T, r Repository) {
				assert.Nil(t, r)
			},
			cloneErr:         fmt.Errorf("error"),
			wantErr:          true,
			expectInitCalled: false,
		},
	}

	orgClone := clone
	orgInit := initRepo
	defer func() { clone = orgClone }()
	defer func() { initRepo = orgInit }()

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			r := &repo{}
			clone = func(ctx context.Context, fs billy.Filesystem, opts *CloneOptions) (*repo, error) {
				if tt.cloneErr != nil {
					return nil, tt.cloneErr
				}
				return r, tt.cloneErr
			}
			initRepo = func(ctx context.Context, opts *CloneOptions) (Repository, error) {
				if !tt.expectInitCalled {
					t.Errorf("expectInitCalled = false, but it was called")
				}
				if tt.initErr != nil {
					return nil, tt.initErr
				}
				return r, tt.initErr
			}

			got, err := Clone(tt.args.ctx, tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.assertFn(t, got)
		})
	}
}

func Test_repo_Persist(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts *PushOptions
	}
	tests := map[string]struct {
		args     args
		wantErr  bool
		retErr   error
		assertFn func(t *testing.T, r *mocks.Repository, w *mocks.Worktree)
	}{
		"NilOpts": {
			args: args{
				ctx:  context.Background(),
				opts: nil,
			},
			wantErr: true,
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree) {
				r.AssertNotCalled(t, "PushContext")
			},
		},
		"Default add pattern": {
			args: args{
				ctx: context.Background(),
				opts: &PushOptions{
					AddGlobPattern: "",
					CommitMsg:      "hello",
				},
			},
			wantErr: false,
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree) {
				r.AssertCalled(t, "PushContext", mock.Anything, mock.Anything)
				assert.True(t, reflect.DeepEqual(r.Calls[0].Arguments[1], &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				}))
				w.AssertCalled(t, "AddGlob", ".")
				w.AssertCalled(t, "Commit", "hello", mock.Anything)
			},
		},
		"With add pattern": {
			args: args{
				ctx: context.Background(),
				opts: &PushOptions{
					AddGlobPattern: "test",
					CommitMsg:      "hello",
				},
			},
			wantErr: false,
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree) {
				r.AssertCalled(t, "PushContext", mock.Anything, mock.Anything)
				assert.True(t, reflect.DeepEqual(r.Calls[0].Arguments[1], &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				}))
				w.AssertCalled(t, "AddGlob", "test")
				w.AssertCalled(t, "Commit", "hello", mock.Anything)
			},
		},
	}

	worktreeOrg := worktree
	defer func() { worktree = worktreeOrg }()

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockRepo := &mocks.Repository{}
			mockRepo.On("PushContext", mock.Anything, mock.Anything).Return(tt.retErr)

			mockWt := &mocks.Worktree{}
			mockWt.On("AddGlob", mock.Anything).Return(tt.retErr)
			mockWt.On("Commit", mock.Anything, mock.Anything).Return(nil, tt.retErr)

			r := &repo{Repository: mockRepo}
			worktree = func(r gogit.Repository) (gogit.Worktree, error) {
				return mockWt, tt.retErr
			}

			if err := r.Persist(tt.args.ctx, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("repo.Persist() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.assertFn(t, mockRepo, mockWt)
		})
	}
}
