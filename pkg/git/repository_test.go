package git

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git/gogit"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git/gogit/mocks"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
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
		auth Auth
		want transport.AuthMethod
	}{
		"Basic": {
			auth: Auth{
				Password: "123",
			},
			want: &http.BasicAuth{
				Username: "git",
				Password: "123",
			},
		},
		"Username": {
			auth: Auth{
				Username: "test",
				Password: "123",
			},
			want: &http.BasicAuth{
				Username: "test",
				Password: "123",
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			if got := getAuth(tt.auth); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_repo_initBranch(t *testing.T) {
	tests := map[string]struct {
		branchName string
		wantErr    bool
		retErr     error
		assertFn   func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree)
	}{
		"Init current branch": {
			branchName: "",
			assertFn: func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree) {
				r.AssertNotCalled(t, "Worktree")
				wt.AssertCalled(t, "Commit", "initial commit", mock.Anything)
				wt.AssertNotCalled(t, "Checkout")
			},
		},
		"Init and checkout branch": {
			assertFn: func(t *testing.T, _ *mocks.Repository, wt *mocks.Worktree) {
				wt.AssertCalled(t, "Commit", "initial commit", mock.Anything)
				b := plumbing.NewBranchReferenceName("test")
				wt.AssertCalled(t, "Checkout", &gg.CheckoutOptions{
					Branch: b,
					Create: true,
				})
			},
		},
		"Error": {
			branchName: "test",
			wantErr:    true,
			retErr:     fmt.Errorf("error"),
			assertFn: func(t *testing.T, _ *mocks.Repository, wt *mocks.Worktree) {
				wt.AssertCalled(t, "Commit", "initial commit", mock.Anything)
				wt.AssertNotCalled(t, "Checkout")
			},
		},
	}

	orgWorktree := worktree
	defer func() { worktree = orgWorktree }()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockRepo := &mocks.Repository{}
			mockWt := &mocks.Worktree{}
			mockWt.On("Commit", mock.Anything, mock.Anything).Return(nil, tt.retErr)
			mockWt.On("Checkout", mock.Anything).Return(tt.retErr)

			worktree = func(r gogit.Repository) (gogit.Worktree, error) { return mockWt, nil }

			r := &repo{Repository: mockRepo}

			if err := r.initBranch(context.Background(), tt.branchName); (err != nil) != tt.wantErr {
				t.Errorf("repo.checkout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_initRepo(t *testing.T) {
	tests := map[string]struct {
		opts     *CloneOptions
		want     Repository
		wantErr  bool
		retErr   error
		assertFn func(t *testing.T, r *mocks.Repository, w *mocks.Worktree)
	}{
		"Basic": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name?ref=test",
			},
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree) {
				r.AssertCalled(t, "CreateRemote")
				w.AssertCalled(t, "Commit")
				w.AssertCalled(t, "Commit")
			},
		},
		"Error": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name?ref=test",
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

			tt.opts.Parse()
			got, err := initRepo(context.Background(), tt.opts)
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
	tests := map[string]struct {
		opts         *CloneOptions
		retErr       error
		wantErr      bool
		assertFn     func(*testing.T, *repo)
		expectedOpts *gg.CloneOptions
	}{
		"NilOpts": {
			wantErr: true,
			assertFn: func(t *testing.T, r *repo) {
				assert.Nil(t, r)
			},
		},
		"No Auth": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name",
				Auth:     nil,
				Depth:    1,
				Progress: os.Stderr,
				Tags:     gg.NoTags,
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.NotNil(t, r)
			},
		},
		"With Auth": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
				Auth: Auth{
					Username: "asd",
					Password: "123",
				},
			},
			expectedOpts: &gg.CloneOptions{
				URL: "https://github.com/owner/name",
				Auth: &http.BasicAuth{
					Username: "asd",
					Password: "123",
				},
				Depth:    1,
				Progress: os.Stderr,
				Tags:     gg.NoTags,
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.NotNil(t, r)
			},
		},
		"Error": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name",
				Depth:    1,
				Progress: os.Stderr,
				Tags:     gg.NoTags,
			},
			retErr:  fmt.Errorf("error"),
			wantErr: true,
			assertFn: func(t *testing.T, r *repo) {
				assert.Nil(t, r)
			},
		},
		"With Revision": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name?ref=test",
			},
			expectedOpts: &gg.CloneOptions{
				URL:           "https://github.com/owner/name",
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
			ggClone = func(_ context.Context, _ storage.Storer, _ billy.Filesystem, o *gg.CloneOptions) (gogit.Repository, error) {
				if tt.expectedOpts != nil {
					assert.True(t, reflect.DeepEqual(o, tt.expectedOpts), "opts not equal")
				}

				if tt.retErr != nil {
					return nil, tt.retErr
				}

				return mockRepo, nil
			}

			if tt.opts != nil {
				tt.opts.Parse()
			}

			got, err := clone(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("clone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			tt.assertFn(t, got)
		})
	}
}

func TestClone(t *testing.T) {
	tests := map[string]struct {
		opts             *CloneOptions
		wantErr          bool
		cloneErr         error
		initErr          error
		expectInitCalled bool
		assertFn         func(*testing.T, Repository, fs.FS)
	}{
		"No error": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			assertFn: func(t *testing.T, r Repository, _ fs.FS) {
				assert.NotNil(t, r)
			},
			expectInitCalled: false,
		},
		"NilOpts": {
			opts: nil,
			assertFn: func(t *testing.T, r Repository, repofs fs.FS) {
				assert.Nil(t, r)
				assert.Nil(t, repofs)
			},
			wantErr: true,
		},
		"EmptyRepo": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			assertFn: func(t *testing.T, r Repository, repofs fs.FS) {
				assert.NotNil(t, r)
				assert.NotNil(t, repofs)
			},
			cloneErr:         transport.ErrEmptyRemoteRepository,
			wantErr:          false,
			expectInitCalled: true,
		},
		"AnotherErr": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			assertFn: func(t *testing.T, r Repository, repofs fs.FS) {
				assert.Nil(t, r)
				assert.Nil(t, repofs)
			},
			cloneErr:         fmt.Errorf("error"),
			wantErr:          true,
			expectInitCalled: false,
		},
		"Use chroot": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name/some/folder",
			},
			assertFn: func(t *testing.T, _ Repository, repofs fs.FS) {
				assert.Equal(t, "/some/folder", repofs.Root())
			},
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
			clone = func(_ context.Context, _ *CloneOptions) (*repo, error) {
				if tt.cloneErr != nil {
					return nil, tt.cloneErr
				}
				return r, nil
			}
			initRepo = func(_ context.Context, _ *CloneOptions) (*repo, error) {
				if !tt.expectInitCalled {
					t.Errorf("expectInitCalled = false, but it was called")
				}
				if tt.initErr != nil {
					return nil, tt.initErr
				}
				return r, nil
			}

			if tt.opts != nil {
				tt.opts.Parse()
				tt.opts.FS = memfs.New()
			}

			gotRepo, gotFS, err := tt.opts.Clone(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			tt.assertFn(t, gotRepo, gotFS)
		})
	}
}

func Test_repo_Persist(t *testing.T) {
	tests := map[string]struct {
		opts     *PushOptions
		wantErr  bool
		retErr   error
		assertFn func(t *testing.T, r *mocks.Repository, w *mocks.Worktree)
	}{
		"NilOpts": {
			opts:    nil,
			wantErr: true,
			assertFn: func(t *testing.T, r *mocks.Repository, _ *mocks.Worktree) {
				r.AssertNotCalled(t, "PushContext")
			},
		},
		"Default add pattern": {
			opts: &PushOptions{
				AddGlobPattern: "",
				CommitMsg:      "hello",
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
			opts: &PushOptions{
				AddGlobPattern: "test",
				CommitMsg:      "hello",
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

			if err := r.Persist(context.Background(), tt.opts); (err != nil) != tt.wantErr {
				t.Errorf("repo.Persist() error = %v, wantErr %v", err, tt.wantErr)
			}

			tt.assertFn(t, mockRepo, mockWt)
		})
	}
}
