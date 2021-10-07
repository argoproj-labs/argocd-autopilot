package git

import (
	"context"
	"errors"
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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		"Should use the supplied username": {
			auth: Auth{
				Username: "test",
				Password: "123",
			},
			want: &http.BasicAuth{
				Username: "test",
				Password: "123",
			},
		},
		"Should return nil if no password is supplied": {
			auth: Auth{},
			want: nil,
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

			gitConfig := &config.Config{
				User: struct {
					Name  string
					Email string
				}{
					Name:  "name",
					Email: "email",
				},
			}

			mockRepo.On("ConfigScoped", mock.Anything).Return(gitConfig, nil)
			mockWt.On("AddGlob", mock.Anything).Return(tt.retErr)

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

			cfg := &config.Config{
				User: struct {
					Name  string
					Email string
				}{
					Name:  "name",
					Email: "email",
				},
			}

			mockRepo.On("ConfigScoped", mock.Anything).Return(cfg, nil)
			mockWt.On("AddGlob", mock.Anything).Return(tt.retErr)

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
		expectedOpts *gg.CloneOptions
		checkoutRef  func(t *testing.T, r *repo, ref string) error
		assertFn     func(t *testing.T, r *repo)
	}{
		"Should fail when there are no CloneOptions": {
			wantErr: true,
			assertFn: func(t *testing.T, r *repo) {
				assert.Nil(t, r)
			},
		},
		"Should clone without Auth data": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name.git",
				Auth:     nil,
				Depth:    1,
				Progress: os.Stderr,
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.NotNil(t, r)
			},
		},
		"Should clone with Auth data": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name.git",
				Auth: Auth{
					Username: "asd",
					Password: "123",
				},
			},
			expectedOpts: &gg.CloneOptions{
				URL: "https://github.com/owner/name.git",
				Auth: &http.BasicAuth{
					Username: "asd",
					Password: "123",
				},
				Depth:    1,
				Progress: os.Stderr,
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.NotNil(t, r)
			},
		},
		"Should fail if go-git.Clone fails": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name.git",
				Depth:    1,
				Progress: os.Stderr,
			},
			retErr:  fmt.Errorf("error"),
			wantErr: true,
			assertFn: func(t *testing.T, r *repo) {
				assert.Nil(t, r)
			},
		},
		"Should checkout revision after clone": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name?ref=test",
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name.git",
				Depth:    1,
				Progress: os.Stderr,
			},
			checkoutRef: func(t *testing.T, _ *repo, ref string) error {
				assert.Equal(t, "test", ref)
				return nil
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.NotNil(t, r)
			},
		},
		"Should fail if checkout fails": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name?ref=test",
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name.git",
				Depth:    1,
				Progress: os.Stderr,
			},
			wantErr: true,
			checkoutRef: func(t *testing.T, _ *repo, ref string) error {
				assert.Equal(t, "test", ref)
				return errors.New("some error")
			},
			assertFn: func(t *testing.T, r *repo) {
				assert.Nil(t, r)
			},
		},
	}

	origCheckoutRef, origClone := checkoutRef, ggClone
	defer func() {
		checkoutRef = origCheckoutRef
		ggClone = origClone
	}()

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

			if tt.checkoutRef != nil {
				checkoutRef = func(r *repo, ref string) error {
					return tt.checkoutRef(t, r, ref)
				}
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

func TestGetRepo(t *testing.T) {
	tests := map[string]struct {
		opts         *CloneOptions
		wantErr      string
		cloneFn      func(context.Context, *CloneOptions) (*repo, error)
		createRepoFn func(context.Context, *CloneOptions) (string, error)
		initRepoFn   func(context.Context, *CloneOptions) (*repo, error)
		assertFn     func(*testing.T, Repository, fs.FS, error)
	}{
		"Should get a repo": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
				FS:   fs.Create(memfs.New()),
			},
			cloneFn: func(_ context.Context, opts *CloneOptions) (*repo, error) {
				return &repo{}, nil
			},
			assertFn: func(t *testing.T, r Repository, f fs.FS, e error) {
				assert.NotNil(t, r)
				assert.NotNil(t, f)
				assert.Nil(t, e)
			},
		},
		"Should fail when no CloneOptions": {
			opts:    nil,
			wantErr: ErrNilOpts.Error(),
			assertFn: func(t *testing.T, r Repository, f fs.FS, e error) {
				assert.Nil(t, r)
				assert.Nil(t, f)
				assert.Error(t, ErrNilOpts, e)
			},
		},
		"Should fail when clone fails": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			cloneFn: func(_ context.Context, opts *CloneOptions) (*repo, error) {
				return nil, errors.New("some error")
			},
			assertFn: func(t *testing.T, r Repository, f fs.FS, e error) {
				assert.Nil(t, r)
				assert.Nil(t, f)
				assert.EqualError(t, e, "some error")
			},
		},
		"Should fail when repo not found": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			cloneFn: func(_ context.Context, opts *CloneOptions) (*repo, error) {
				return nil, transport.ErrRepositoryNotFound
			},
			assertFn: func(t *testing.T, r Repository, f fs.FS, e error) {
				assert.Nil(t, r)
				assert.Nil(t, f)
				assert.Error(t, transport.ErrRepositoryNotFound, e)
			},
		},
		"Should fail when createIfNotExist is true and create fails": {
			opts: &CloneOptions{
				Repo:             "https://github.com/owner/name",
				createIfNotExist: true,
			},
			wantErr: "some error",
			cloneFn: func(_ context.Context, opts *CloneOptions) (*repo, error) {
				return nil, transport.ErrRepositoryNotFound
			},
			createRepoFn: func(c context.Context, co *CloneOptions) (string, error) {
				return "", errors.New("some error")
			},
			assertFn: func(t *testing.T, r Repository, f fs.FS, e error) {
				assert.Nil(t, r)
				assert.Nil(t, f)
				assert.EqualError(t, e, "failed to create repository: some error")
			},
		},
		"Should fail when repo is empty and init fails": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			wantErr: "some error",
			cloneFn: func(_ context.Context, opts *CloneOptions) (*repo, error) {
				return nil, transport.ErrEmptyRemoteRepository
			},
			initRepoFn: func(c context.Context, co *CloneOptions) (*repo, error) {
				return nil, errors.New("some error")
			},
			assertFn: func(t *testing.T, r Repository, f fs.FS, e error) {
				assert.Nil(t, r)
				assert.Nil(t, f)
				assert.EqualError(t, e, "failed to initialize repository: some error")
			},
		},
		"Should create and init repo when createIfNotExist is true": {
			opts: &CloneOptions{
				Repo:             "https://github.com/owner/name",
				createIfNotExist: true,
				FS:               fs.Create(memfs.New()),
			},
			wantErr: "some error",
			cloneFn: func(_ context.Context, opts *CloneOptions) (*repo, error) {
				return nil, transport.ErrRepositoryNotFound
			},
			createRepoFn: func(c context.Context, co *CloneOptions) (string, error) {
				return "", nil
			},
			initRepoFn: func(c context.Context, co *CloneOptions) (*repo, error) {
				return &repo{}, nil
			},
			assertFn: func(t *testing.T, r Repository, f fs.FS, e error) {
				assert.NotNil(t, r)
				assert.NotNil(t, f)
				assert.Nil(t, e)
			},
		},
	}
	origClone, origCreateRepo, origInitRepo := clone, createRepo, initRepo
	defer func() {
		clone = origClone
		createRepo = origCreateRepo
		initRepo = origInitRepo
	}()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			clone = tt.cloneFn
			createRepo = tt.createRepoFn
			initRepo = tt.initRepoFn
			if tt.opts != nil {
				tt.opts.Parse()
			}

			r, fs, err := tt.opts.GetRepo(context.Background())
			tt.assertFn(t, r, fs, err)
		})
	}
}

func Test_repo_Persist(t *testing.T) {
	tests := map[string]struct {
		opts        *PushOptions
		retRevision string
		retErr      error
		assertFn    func(t *testing.T, r *mocks.Repository, w *mocks.Worktree, revision string, err error)
	}{
		"NilOpts": {
			opts: nil,
			assertFn: func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree, revision string, err error) {
				assert.ErrorIs(t, err, ErrNilOpts)
				assert.Equal(t, "", revision)
				wt.AssertNotCalled(t, "AddGlob")
				wt.AssertNotCalled(t, "Commit")
				r.AssertNotCalled(t, "PushContext")
			},
		},
		"Default add pattern": {
			opts: &PushOptions{
				AddGlobPattern: "",
				CommitMsg:      "hello",
			},
			retRevision: "0dee45f70b37aeb59e6d2efb29855f97df9bccb2",
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree, revision string, err error) {
				assert.Equal(t, "0dee45f70b37aeb59e6d2efb29855f97df9bccb2", revision)
				assert.Nil(t, err)
				r.AssertCalled(t, "PushContext", mock.Anything, &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				})
				w.AssertCalled(t, "AddGlob", ".")
				w.AssertCalled(t, "Commit", "hello", mock.Anything)
			},
		},
		"With add pattern": {
			opts: &PushOptions{
				AddGlobPattern: "test",
				CommitMsg:      "hello",
			},
			retRevision: "0dee45f70b37aeb59e6d2efb29855f97df9bccb2",
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree, revision string, err error) {
				assert.Equal(t, "0dee45f70b37aeb59e6d2efb29855f97df9bccb2", revision)
				assert.Nil(t, err)
				r.AssertCalled(t, "PushContext", mock.Anything, &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				})
				w.AssertCalled(t, "AddGlob", "test")
				w.AssertCalled(t, "Commit", "hello", mock.Anything)
			},
		},
		"Retry push on 'repo not found err'": {
			opts: &PushOptions{
				AddGlobPattern: "test",
				CommitMsg:      "hello",
			},
			retErr:      transport.ErrRepositoryNotFound,
			retRevision: "0dee45f70b37aeb59e6d2efb29855f97df9bccb2",
			assertFn: func(t *testing.T, r *mocks.Repository, w *mocks.Worktree, revision string, err error) {
				assert.Equal(t, "0dee45f70b37aeb59e6d2efb29855f97df9bccb2", revision)
				assert.Error(t, err, transport.ErrRepositoryNotFound)
				r.AssertCalled(t, "PushContext", mock.Anything, &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				})
				r.AssertNumberOfCalls(t, "PushContext", 3)
				w.AssertCalled(t, "AddGlob", "test")
				w.AssertCalled(t, "Commit", "hello", mock.Anything)
			},
		},
	}

	gitConfig := &config.Config{
		User: struct {
			Name  string
			Email string
		}{
			Name:  "name",
			Email: "email",
		},
	}

	worktreeOrg := worktree
	defer func() { worktree = worktreeOrg }()

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockRepo := &mocks.Repository{}
			mockRepo.On("PushContext", mock.Anything, mock.Anything).Return(tt.retErr)
			mockRepo.On("ConfigScoped", mock.Anything).Return(gitConfig, nil)

			mockWt := &mocks.Worktree{}
			mockWt.On("AddGlob", mock.Anything).Return(nil)
			mockWt.On("Commit", mock.Anything, mock.Anything).Return(plumbing.NewHash(tt.retRevision), nil)

			r := &repo{Repository: mockRepo, progress: os.Stderr}
			worktree = func(r gogit.Repository) (gogit.Worktree, error) {
				return mockWt, nil
			}

			revision, err := r.Persist(context.Background(), tt.opts)
			tt.assertFn(t, mockRepo, mockWt, revision, err)
		})
	}
}

func Test_repo_checkoutRef(t *testing.T) {
	tests := map[string]struct {
		ref      string
		hash     string
		wantErr  string
		beforeFn func() *mocks.Repository
	}{
		"Should checkout a specific hash": {
			ref:  "3992c4",
			hash: "3992c4",
			beforeFn: func() *mocks.Repository {
				r := &mocks.Repository{}
				hash := plumbing.NewHash("3992c4")
				r.On("ResolveRevision", plumbing.Revision("3992c4")).Return(&hash, nil)
				return r
			},
		},
		"Should checkout a tag": {
			ref:  "v1.0.0",
			hash: "3992c4",
			beforeFn: func() *mocks.Repository {
				r := &mocks.Repository{}
				hash := plumbing.NewHash("3992c4")
				r.On("ResolveRevision", plumbing.Revision("v1.0.0")).Return(&hash, nil)
				return r
			},
		},
		"Should checkout a branch": {
			ref:  "CR-1234",
			hash: "3992c4",
			beforeFn: func() *mocks.Repository {
				r := &mocks.Repository{}
				r.On("ResolveRevision", plumbing.Revision("CR-1234")).Return(nil, plumbing.ErrReferenceNotFound)
				r.On("Remotes").Return([]*gg.Remote{
					gg.NewRemote(nil, &config.RemoteConfig{
						Name: "origin",
					}),
				}, nil)
				hash := plumbing.NewHash("3992c4")
				r.On("ResolveRevision", plumbing.Revision("origin/CR-1234")).Return(&hash, nil)
				return r
			},
		},
		"Should fail if ResolveRevision fails": {
			ref:     "CR-1234",
			hash:    "3992c4",
			wantErr: "some error",
			beforeFn: func() *mocks.Repository {
				r := &mocks.Repository{}
				r.On("ResolveRevision", plumbing.Revision("CR-1234")).Return(nil, errors.New("some error"))
				return r
			},
		},
		"Should fail if Remotes fails": {
			ref:     "CR-1234",
			hash:    "3992c4",
			wantErr: "some error",
			beforeFn: func() *mocks.Repository {
				r := &mocks.Repository{}
				r.On("ResolveRevision", plumbing.Revision("CR-1234")).Return(nil, plumbing.ErrReferenceNotFound)
				r.On("Remotes").Return(nil, errors.New("some error"))
				return r
			},
		},
		"Should fail if repo has no remotes": {
			ref:     "CR-1234",
			hash:    "3992c4",
			wantErr: ErrNoRemotes.Error(),
			beforeFn: func() *mocks.Repository {
				r := &mocks.Repository{}
				r.On("ResolveRevision", plumbing.Revision("CR-1234")).Return(nil, plumbing.ErrReferenceNotFound)
				r.On("Remotes").Return([]*gg.Remote{}, nil)
				return r
			},
		},
		"Should fail if branch not found": {
			ref:     "CR-1234",
			hash:    "3992c4",
			wantErr: plumbing.ErrReferenceNotFound.Error(),
			beforeFn: func() *mocks.Repository {
				r := &mocks.Repository{}
				r.On("ResolveRevision", plumbing.Revision("CR-1234")).Return(nil, plumbing.ErrReferenceNotFound)
				r.On("Remotes").Return([]*gg.Remote{
					gg.NewRemote(nil, &config.RemoteConfig{
						Name: "origin",
					}),
				}, nil)
				r.On("ResolveRevision", plumbing.Revision("origin/CR-1234")).Return(nil, plumbing.ErrReferenceNotFound)
				return r
			},
		},
	}
	origWorktree := worktree
	defer func() { worktree = origWorktree }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockwt := &mocks.Worktree{}
			worktree = func(r gogit.Repository) (gogit.Worktree, error) {
				return mockwt, nil
			}
			mockwt.On("Checkout", &gg.CheckoutOptions{
				Hash: plumbing.NewHash(tt.hash),
			}).Return(nil)
			mockrepo := tt.beforeFn()
			r := &repo{Repository: mockrepo}
			if err := r.checkoutRef(tt.ref); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("repo.checkoutRef() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			mockrepo.AssertExpectations(t)
			mockwt.AssertExpectations(t)
		})
	}
}

func TestAddFlags(t *testing.T) {
	type flag struct {
		name      string
		shorthand string
		value     string
		usage     string
		required  bool
	}
	tests := map[string]struct {
		opts        *AddFlagsOptions
		wantedFlags []flag
	}{
		"Should create flags without a prefix": {
			opts: &AddFlagsOptions{},
			wantedFlags: []flag{
				{
					name:      "git-token",
					shorthand: "t",
					usage:     "Your git provider api token [GIT_TOKEN]",
					required:  true,
				},
				{
					name:     "repo",
					usage:    "Repository URL [GIT_REPO]",
					required: true,
				},
			},
		},
		"Should create flags with optional": {
			opts: &AddFlagsOptions{
				Optional: true,
			},
			wantedFlags: []flag{
				{
					name:      "git-token",
					shorthand: "t",
					usage:     "Your git provider api token [GIT_TOKEN]",
				},
				{
					name:  "repo",
					usage: "Repository URL [GIT_REPO]",
				},
			},
		},
		"Should create flags with a prefix": {
			opts: &AddFlagsOptions{
				Prefix: "prefix-",
			},
			wantedFlags: []flag{
				{
					name:     "prefix-git-token",
					usage:    "Your git provider api token [PREFIX_GIT_TOKEN]",
					required: true,
				},
				{
					name:     "prefix-repo",
					usage:    "Repository URL [PREFIX_GIT_REPO]",
					required: true,
				},
			},
		},
		"Should automatically add a dash to prefix": {
			opts: &AddFlagsOptions{
				Prefix: "prefix",
			},
			wantedFlags: []flag{
				{
					name:     "prefix-git-token",
					usage:    "Your git provider api token [PREFIX_GIT_TOKEN]",
					required: true,
				},
				{
					name:     "prefix-repo",
					usage:    "Repository URL [PREFIX_GIT_REPO]",
					required: true,
				},
			},
		},
		"Should add provider flag when needed": {
			opts: &AddFlagsOptions{
				CreateIfNotExist: true,
			},
			wantedFlags: []flag{
				{
					name:  "provider",
					usage: "The git provider, one of: gitea|github",
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			cmd := &cobra.Command{}
			tt.opts.FS = memfs.New()
			_ = AddFlags(cmd, tt.opts)
			fs := cmd.PersistentFlags()
			for _, expected := range tt.wantedFlags {
				actual := fs.Lookup(expected.name)
				assert.NotNil(t, actual, "missing "+expected.name+" flag")
				assert.Equal(t, expected.shorthand, actual.Shorthand, "wrong shorthand for "+expected.name)
				assert.Equal(t, expected.value, actual.DefValue, "wrong default value for "+expected.name)
				assert.Equal(t, expected.usage, actual.Usage, "wrong usage for "+expected.name)
				if expected.required {
					assert.NotEmpty(t, actual.Annotations[cobra.BashCompOneRequiredFlag], expected.name+" was supposed to be required")
					assert.Equal(t, "true", actual.Annotations[cobra.BashCompOneRequiredFlag][0], expected.name+" was supposed to be required")
				} else {
					assert.Nil(t, actual.Annotations[cobra.BashCompOneRequiredFlag], expected.name+" was not supposed to be required")
				}
			}
		})
	}
}

type mockProvider struct {
	createRepository func(opts *CreateRepoOptions) (string, error)
}

func (p *mockProvider) CreateRepository(_ context.Context, opts *CreateRepoOptions) (string, error) {
	return p.createRepository(opts)
}

func Test_createRepo(t *testing.T) {
	tests := map[string]struct {
		opts        *CloneOptions
		want        string
		wantErr     string
		newProvider func(*testing.T, *ProviderOptions) (Provider, error)
	}{
		"Should create new repository": {
			opts: &CloneOptions{
				Repo:     "https://github.com/owner/name.git",
				Provider: "github",
				Auth: Auth{
					Username: "username",
					Password: "password",
				},
			},
			want: "https://github.com/owner/name.git",
			newProvider: func(t *testing.T, opts *ProviderOptions) (Provider, error) {
				assert.Equal(t, "username", opts.Auth.Username)
				assert.Equal(t, "password", opts.Auth.Password)
				assert.Equal(t, "https://github.com/", opts.Host)
				assert.Equal(t, "github", opts.Type)
				return &mockProvider{func(opts *CreateRepoOptions) (string, error) {
					assert.Equal(t, "owner", opts.Owner)
					assert.Equal(t, "name", opts.Name)
					assert.Equal(t, true, opts.Private)
					return "https://github.com/owner/name.git", nil
				}}, nil
			},
		},
		"Should infer correct provider type from repo url": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name.git",
			},
			want: "https://github.com/owner/name.git",
			newProvider: func(t *testing.T, opts *ProviderOptions) (Provider, error) {
				assert.Equal(t, "github", opts.Type)
				return &mockProvider{func(opts *CreateRepoOptions) (string, error) {
					return "https://github.com/owner/name.git", nil
				}}, nil
			},
		},
		"Should fail if provider type is unknown": {
			opts: &CloneOptions{
				Repo: "https://unkown.com/owner/name",
			},
			wantErr: "failed to create the repository, you can try to manually create it before trying again: git provider 'unkown' not supported",
		},
		"Should fail if url doesn't contain orgRepo parts": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner.git",
			},
			wantErr: "failed parsing organization and repo from 'owner'",
		},
		"Should succesfully parse owner and name for long orgRepos": {
			opts: &CloneOptions{
				Repo: "https://github.com/foo22/bar/fizz.git",
			},
			want: "https://github.com/foo22/bar/fizz.git",
			newProvider: func(t *testing.T, opts *ProviderOptions) (Provider, error) {
				assert.Equal(t, "https://github.com/", opts.Host)
				assert.Equal(t, "github", opts.Type)
				return &mockProvider{func(opts *CreateRepoOptions) (string, error) {
					assert.Equal(t, "foo22/bar", opts.Owner)
					assert.Equal(t, "fizz", opts.Name)
					return "https://github.com/foo22/bar/fizz.git", nil
				}}, nil
			},
		},
	}
	origNewProvider := supportedProviders["github"]
	defer func() { supportedProviders["github"] = origNewProvider }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.newProvider != nil {
				supportedProviders["github"] = func(opts *ProviderOptions) (Provider, error) {
					return tt.newProvider(t, opts)
				}
			} else {
				supportedProviders["github"] = origNewProvider
			}

			got, err := createRepo(context.Background(), tt.opts)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("createRepo() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if got != tt.want {
				t.Errorf("createRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_repo_commit(t *testing.T) {
	tests := map[string]struct {
		branchName string
		wantErr    string
		retErr     error
		assertFn   func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree)
		beforeFn   func() *mocks.Repository
	}{
		"Success": {
			branchName: "",
			beforeFn: func() *mocks.Repository {
				mockRepo := &mocks.Repository{}
				mockWt := &mocks.Worktree{}
				hash := plumbing.NewHash("3992c4")
				mockWt.On("Commit", "test", mock.Anything).Return(hash, nil)
				mockWt.On("AddGlob", mock.Anything).Return(nil)
				worktree = func(r gogit.Repository) (gogit.Worktree, error) {
					return mockWt, nil
				}

				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "name",
						Email: "email",
					},
				}

				mockRepo.On("ConfigScoped", mock.Anything).Return(config, nil)

				return mockRepo
			},
			assertFn: func(t *testing.T, r *mocks.Repository, wt *mocks.Worktree) {
				r.AssertCalled(t, "Worktree")
				wt.AssertCalled(t, "Commit", "initial commit", mock.Anything)
			},
		},
		"Error - no gitconfig name and email": {
			branchName: "test",
			beforeFn: func() *mocks.Repository {
				mockRepo := &mocks.Repository{}

				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "",
						Email: "",
					},
				}

				mockRepo.On("ConfigScoped", mock.Anything).Return(config, nil)

				return mockRepo
			},
			wantErr: "failed to commit. Please make sure your gitconfig contains a name and an email",
			assertFn: func(t *testing.T, _ *mocks.Repository, wt *mocks.Worktree) {
				wt.AssertNotCalled(t, "Commit", "initial commit", mock.Anything)
			},
		},

		"Error - ConfigScope fails": {
			branchName: "test",
			beforeFn: func() *mocks.Repository {
				mockRepo := &mocks.Repository{}
				mockRepo.On("ConfigScoped", mock.Anything).Return(nil, fmt.Errorf("test Config error"))

				return mockRepo
			},
			wantErr: "failed to get gitconfig. Error: test Config error",
			assertFn: func(t *testing.T, _ *mocks.Repository, wt *mocks.Worktree) {
				wt.AssertNotCalled(t, "Commit", "initial commit", mock.Anything)
			},
		},

		"Error - AddGlob fails": {
			branchName: "test",
			beforeFn: func() *mocks.Repository {
				mockRepo := &mocks.Repository{}
				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "name",
						Email: "email",
					},
				}

				mockRepo.On("ConfigScoped", mock.Anything).Return(config, nil)
				mockWt := &mocks.Worktree{}
				mockWt.On("AddGlob", mock.Anything).Return(fmt.Errorf("add glob error"))

				worktree = func(r gogit.Repository) (gogit.Worktree, error) {
					return mockWt, nil
				}

				return mockRepo
			},
			wantErr: "add glob error",
			assertFn: func(t *testing.T, _ *mocks.Repository, wt *mocks.Worktree) {
				wt.AssertNotCalled(t, "Commit", "initial commit", mock.Anything)
			},
		},

		"Error - Commit fails": {
			branchName: "test",
			beforeFn: func() *mocks.Repository {
				mockRepo := &mocks.Repository{}
				mockWt := &mocks.Worktree{}
				mockWt.On("AddGlob", mock.Anything).Return(nil)
				worktree = func(r gogit.Repository) (gogit.Worktree, error) {
					return mockWt, nil
				}

				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "name",
						Email: "email",
					},
				}

				mockRepo.On("ConfigScoped", mock.Anything).Return(config, nil)
				mockWt.On("Commit", "test", mock.Anything).Return(nil, fmt.Errorf("test Config error"))

				return mockRepo
			},
			wantErr: "test Config error",
			assertFn: func(t *testing.T, _ *mocks.Repository, wt *mocks.Worktree) {
				wt.AssertNotCalled(t, "Commit", "initial commit", mock.Anything)
			},
		},
	}

	orgWorktree := worktree
	defer func() { worktree = orgWorktree }()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockRepo := tt.beforeFn()
			r := &repo{Repository: mockRepo}

			hash := plumbing.NewHash("3992c4")

			got, err := r.commit(&PushOptions{
				CommitMsg: "test",
			})

			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("r.commit() error = %v", err)
				}

				return
			}

			assert.Equal(t, got, &hash)
		})
	}
}
