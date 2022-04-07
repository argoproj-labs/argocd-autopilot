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
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	createRepository func(orgRepo string) (string, error)

	getAuthor func() (string, string, error)
}

func (p *mockProvider) CreateRepository(_ context.Context, orgRepo string) (string, error) {
	return p.createRepository(orgRepo)
}

func (p *mockProvider) GetAuthor(ctx context.Context) (username, email string, err error) {
	if p.getAuthor != nil {
		return p.getAuthor()
	}

	return "username", "user@email.com", nil
}

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
			mockRepo := mocks.NewMockRepository(gomock.NewController(t))
			mockRepo.EXPECT().CreateRemote(&config.RemoteConfig{
				Name: tt.args.name,
				URLs: []string{tt.args.url},
			}).
				Times(1).
				Return(nil, tt.retErr)

			r := &repo{Repository: mockRepo}
			if err := r.addRemote(tt.args.name, tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("repo.addRemote() error = %v, wantErr %v", err, tt.wantErr)
			}
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
		assertFn   func(r *mocks.MockRepository, wt *mocks.MockWorktree)
	}{
		"Init current branch": {
			branchName: "",
			assertFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree) {
				r.EXPECT().Worktree().Times(0)
				wt.EXPECT().Commit("initial commit", gomock.Any()).Times(1)
				wt.EXPECT().Checkout(gomock.Any()).Times(0)
			},
		},
		"Init and checkout branch": {
			assertFn: func(_ *mocks.MockRepository, wt *mocks.MockWorktree) {
				b := plumbing.NewBranchReferenceName("test")
				wt.EXPECT().Commit("initial commit", gomock.Any()).
					Times(1)
				wt.EXPECT().Checkout(&gg.CheckoutOptions{Branch: b, Create: true}).
					Times(1)
			},
		},
		"Error": {
			branchName: "test",
			wantErr:    true,
			retErr:     fmt.Errorf("error"),
			assertFn: func(_ *mocks.MockRepository, wt *mocks.MockWorktree) {
				wt.EXPECT().Commit("initial commit", gomock.Any()).
					Times(0)
				wt.EXPECT().Checkout(gomock.Any()).
					Times(0)
			},
		},
	}

	orgWorktree := worktree
	defer func() { worktree = orgWorktree }()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockRepository(ctrl)
			mockWt := mocks.NewMockWorktree(ctrl)
			mockProvider := &mockProvider{}

			mockWt.EXPECT().Commit(gomock.Any(), gomock.Any()).Return(plumbing.Hash{}, tt.retErr).AnyTimes()
			mockWt.EXPECT().Checkout(gomock.Any()).Return(tt.retErr).AnyTimes()
			mockWt.EXPECT().AddGlob(gomock.Any()).Return(tt.retErr).AnyTimes()

			gitConfig := &config.Config{
				User: struct {
					Name  string
					Email string
				}{
					Name:  "name",
					Email: "email",
				},
			}

			mockRepo.EXPECT().ConfigScoped(gomock.Any()).Return(gitConfig, nil).AnyTimes()

			worktree = func(r gogit.Repository) (gogit.Worktree, error) { return mockWt, nil }

			r := &repo{Repository: mockRepo, provider: mockProvider}

			if err := r.initBranch(context.Background(), tt.branchName); (err != nil) != tt.wantErr {
				t.Errorf("repo.checkout() error = %v, wantErr %v", err, tt.wantErr)
			}

			// tt.assertFn(mockRepo, mockWt)
		})
	}
}

func Test_initRepo(t *testing.T) {
	tests := map[string]struct {
		opts     *CloneOptions
		want     Repository
		wantErr  bool
		retErr   error
		assertFn func(r *mocks.MockRepository, w *mocks.MockWorktree)
	}{
		"Basic": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name?ref=test",
			},
			assertFn: func(r *mocks.MockRepository, w *mocks.MockWorktree) {
				r.EXPECT().CreateRemote(gomock.Any()).Times(1)
				w.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(2)
			},
		},
		"Error": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name?ref=test",
			},
			retErr:  fmt.Errorf("error"),
			wantErr: true,
			assertFn: func(r *mocks.MockRepository, w *mocks.MockWorktree) {
				r.EXPECT().CreateRemote(gomock.Any()).Times(1)
				w.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(0)
			},
		},
	}

	orgInitRepo := ggInitRepo
	defer func() { ggInitRepo = orgInitRepo }()
	orgWorktree := worktree
	defer func() { worktree = orgWorktree }()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockRepository(ctrl)
			mockWt := mocks.NewMockWorktree(ctrl)
			mockProvider := &mockProvider{}

			mockRepo.EXPECT().CreateRemote(gomock.Any()).Return(nil, tt.retErr).AnyTimes()
			mockWt.EXPECT().Commit(gomock.Any(), gomock.Any()).Return(plumbing.Hash{}, tt.retErr).AnyTimes()
			mockWt.EXPECT().Checkout(gomock.Any()).Return(tt.retErr).AnyTimes()

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

			mockRepo.EXPECT().ConfigScoped(gomock.Any()).Return(cfg, nil).AnyTimes()
			mockWt.EXPECT().AddGlob(gomock.Any()).Return(tt.retErr).AnyTimes()

			tt.opts.Parse()
			tt.opts.provider = mockProvider
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
		opts           *CloneOptions
		retErr         error
		wantErr        bool
		expectedOpts   *gg.CloneOptions
		checkoutRef    func(t *testing.T, r *repo, ref string) error
		checkoutBranch func(t *testing.T, r *repo, branch string, upsert bool) error
		assertFn       func(t *testing.T, r *repo, cloneCalls int)
	}{
		"Should fail when there are no CloneOptions": {
			wantErr: true,
			assertFn: func(t *testing.T, r *repo, _ int) {
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
			assertFn: func(t *testing.T, r *repo, _ int) {
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
			assertFn: func(t *testing.T, r *repo, _ int) {
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
			assertFn: func(t *testing.T, r *repo, _ int) {
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
			assertFn: func(t *testing.T, r *repo, _ int) {
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
			assertFn: func(t *testing.T, r *repo, _ int) {
				assert.Nil(t, r)
			},
		},
		"Should try to upsert branch if upsert branch and cloneForWrite are set": {
			opts: &CloneOptions{
				Repo:          "https://github.com/owner/name?ref=test",
				UpsertBranch:  true,
				CloneForWrite: true,
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name.git",
				Depth:    1,
				Progress: os.Stderr,
			},
			checkoutRef: func(t *testing.T, _ *repo, _ string) error {
				// should not call this function
				assert.Equal(t, true, false)
				return nil
			},
			checkoutBranch: func(t *testing.T, _ *repo, branch string, upsert bool) error {
				assert.Equal(t, branch, "test")
				assert.Equal(t, upsert, true)
				return nil
			},
			assertFn: func(t *testing.T, r *repo, _ int) {
				assert.NotNil(t, r)
			},
		},
		"Should retry if fails with 'repo not found' error": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name",
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name.git",
				Auth:     nil,
				Depth:    1,
				Progress: os.Stderr,
			},
			assertFn: func(t *testing.T, r *repo, cloneCalls int) {
				assert.Nil(t, r)
				assert.Equal(t, cloneCalls, 3)
			},
			retErr:  transport.ErrRepositoryNotFound,
			wantErr: true,
		},
		"Should not retry if CreateIfNotExist is true": {
			opts: &CloneOptions{
				Repo:             "https://github.com/owner/name",
				CreateIfNotExist: true,
			},
			expectedOpts: &gg.CloneOptions{
				URL:      "https://github.com/owner/name.git",
				Auth:     nil,
				Depth:    1,
				Progress: os.Stderr,
			},
			assertFn: func(t *testing.T, r *repo, cloneCalls int) {
				assert.Nil(t, r)
				assert.Equal(t, cloneCalls, 1)
			},
			retErr:  transport.ErrRepositoryNotFound,
			wantErr: true,
		},
	}

	origCheckoutRef, origClone := checkoutRef, ggClone
	defer func() {
		checkoutRef = origCheckoutRef
		ggClone = origClone
	}()

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockRepo := mocks.NewMockRepository(gomock.NewController(t))
			cloneCalls := 0
			ggClone = func(_ context.Context, _ storage.Storer, _ billy.Filesystem, o *gg.CloneOptions) (gogit.Repository, error) {
				cloneCalls++
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

			if tt.checkoutBranch != nil {
				checkoutBranch = func(r *repo, branch string, upsertBranch bool) error {
					return tt.checkoutBranch(t, r, branch, upsertBranch)
				}
			}

			got, err := clone(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("clone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			tt.assertFn(t, got, cloneCalls)
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
		"Should fail when CreateIfNotExist is true and create fails": {
			opts: &CloneOptions{
				Repo:             "https://github.com/owner/name",
				CreateIfNotExist: true,
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
		"Should create and init repo when CreateIfNotExist is true": {
			opts: &CloneOptions{
				Repo:             "https://github.com/owner/name",
				CreateIfNotExist: true,
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
		wantErr     bool
		beforeFn    func(r *mocks.MockRepository, w *mocks.MockWorktree)
	}{
		"NilOpts": {
			opts:    nil,
			wantErr: true,
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree) {
				wt.EXPECT().AddGlob(gomock.Any()).Times(0)
				wt.EXPECT().Commit(gomock.Any(), gomock.Any()).Times(0)
				r.EXPECT().PushContext(gomock.Any(), gomock.Any()).Times(0)
			},
		},
		"Default add pattern": {
			opts: &PushOptions{
				AddGlobPattern: "",
				CommitMsg:      "hello",
			},
			retRevision: "0dee45f70b37aeb59e6d2efb29855f97df9bccb2",
			beforeFn: func(r *mocks.MockRepository, w *mocks.MockWorktree) {
				r.EXPECT().PushContext(gomock.Any(), &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				}).
					Times(1).
					Return(nil)
				w.EXPECT().AddGlob(".").
					Times(1).
					Return(nil)
				w.EXPECT().Commit("hello", gomock.Any()).
					Times(1).
					Return(plumbing.NewHash("0dee45f70b37aeb59e6d2efb29855f97df9bccb2"), nil)
			},
		},
		"With add pattern": {
			opts: &PushOptions{
				AddGlobPattern: "test",
				CommitMsg:      "hello",
			},
			retRevision: "0dee45f70b37aeb59e6d2efb29855f97df9bccb2",
			beforeFn: func(r *mocks.MockRepository, w *mocks.MockWorktree) {
				r.EXPECT().PushContext(gomock.Any(), &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				}).
					Times(1).
					Return(nil)
				w.EXPECT().AddGlob("test").
					Times(1).
					Return(nil)
				w.EXPECT().Commit("hello", gomock.Any()).
					Times(1).
					Return(plumbing.NewHash("0dee45f70b37aeb59e6d2efb29855f97df9bccb2"), nil)
			},
		},
		"Retry push on 'repo not found err'": {
			opts: &PushOptions{
				AddGlobPattern: "test",
				CommitMsg:      "hello",
			},
			retRevision: "0dee45f70b37aeb59e6d2efb29855f97df9bccb2",
			beforeFn: func(r *mocks.MockRepository, w *mocks.MockWorktree) {
				r.EXPECT().PushContext(gomock.Any(), &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				}).
					Times(1).
					Return(transport.ErrRepositoryNotFound).
					Times(1).
					Return(nil)
				w.EXPECT().AddGlob("test").
					Times(1).
					Return(nil)
				w.EXPECT().Commit("hello", gomock.Any()).
					Times(1).
					Return(plumbing.NewHash("0dee45f70b37aeb59e6d2efb29855f97df9bccb2"), nil)
			},
		},
		"Fail after 3 retries with 'repo not found err'": {
			opts: &PushOptions{
				AddGlobPattern: "test",
				CommitMsg:      "hello",
			},
			wantErr: true,
			beforeFn: func(r *mocks.MockRepository, w *mocks.MockWorktree) {
				r.EXPECT().PushContext(gomock.Any(), &gg.PushOptions{
					Auth:     nil,
					Progress: os.Stderr,
				}).
					Times(3).
					Return(transport.ErrRepositoryNotFound)
				w.EXPECT().AddGlob("test").
					Times(1).
					Return(nil)
				w.EXPECT().Commit("hello", gomock.Any()).
					Times(1).
					Return(plumbing.NewHash("0dee45f70b37aeb59e6d2efb29855f97df9bccb2"), nil)
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
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockRepository(ctrl)
			mockWt := mocks.NewMockWorktree(ctrl)
			mockProvider := &mockProvider{}

			mockRepo.EXPECT().ConfigScoped(gomock.Any()).Return(gitConfig, nil).AnyTimes()

			r := &repo{Repository: mockRepo, progress: os.Stderr, provider: mockProvider}
			worktree = func(r gogit.Repository) (gogit.Worktree, error) {
				return mockWt, nil
			}

			tt.beforeFn(mockRepo, mockWt)

			revision, err := r.Persist(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Equal(t, tt.retRevision, revision)
			}
		})
	}
}

func Test_repo_checkoutRef(t *testing.T) {
	tests := map[string]struct {
		ref      string
		hash     string
		wantErr  string
		beforeFn func(*mocks.MockRepository)
	}{
		"Should checkout a specific hash": {
			ref:  "3992c4",
			hash: "3992c4",
			beforeFn: func(r *mocks.MockRepository) {
				hash := plumbing.NewHash("3992c4")
				r.EXPECT().ResolveRevision(plumbing.Revision("3992c4")).
					Times(1).
					Return(&hash, nil)
			},
		},
		"Should checkout a tag": {
			ref:  "v1.0.0",
			hash: "3992c4",
			beforeFn: func(r *mocks.MockRepository) {
				hash := plumbing.NewHash("3992c4")
				r.EXPECT().ResolveRevision(plumbing.Revision("v1.0.0")).
					Times(1).
					Return(&hash, nil)
			},
		},
		"Should checkout a branch": {
			ref:  "CR-1234",
			hash: "3992c4",
			beforeFn: func(r *mocks.MockRepository) {
				r.EXPECT().ResolveRevision(plumbing.Revision("CR-1234")).
					Times(1).
					Return(nil, plumbing.ErrReferenceNotFound)
				r.EXPECT().Remotes().
					Times(1).
					Return([]*gg.Remote{
						gg.NewRemote(nil, &config.RemoteConfig{
							Name: "origin",
						}),
					}, nil)
				hash := plumbing.NewHash("3992c4")
				r.EXPECT().ResolveRevision(plumbing.Revision("origin/CR-1234")).
					Times(1).
					Return(&hash, nil)
			},
		},
		"Should fail if ResolveRevision fails": {
			ref:     "CR-1234",
			hash:    "3992c4",
			wantErr: "some error",
			beforeFn: func(r *mocks.MockRepository) {
				r.EXPECT().ResolveRevision(plumbing.Revision("CR-1234")).
					Times(1).
					Return(nil, errors.New("some error"))
			},
		},
		"Should fail if Remotes fails": {
			ref:     "CR-1234",
			hash:    "3992c4",
			wantErr: "some error",
			beforeFn: func(r *mocks.MockRepository) {
				r.EXPECT().ResolveRevision(plumbing.Revision("CR-1234")).
					Times(1).
					Return(nil, plumbing.ErrReferenceNotFound)
				r.EXPECT().Remotes().
					Times(1).
					Return(nil, errors.New("some error"))
			},
		},
		"Should fail if repo has no remotes": {
			ref:     "CR-1234",
			hash:    "3992c4",
			wantErr: ErrNoRemotes.Error(),
			beforeFn: func(r *mocks.MockRepository) {
				r.EXPECT().ResolveRevision(plumbing.Revision("CR-1234")).
					Times(1).
					Return(nil, plumbing.ErrReferenceNotFound)
				r.EXPECT().Remotes().
					Times(1).
					Return([]*gg.Remote{}, nil)
			},
		},
		"Should fail if branch not found": {
			ref:     "CR-1234",
			hash:    "3992c4",
			wantErr: plumbing.ErrReferenceNotFound.Error(),
			beforeFn: func(r *mocks.MockRepository) {
				r.EXPECT().ResolveRevision(plumbing.Revision("CR-1234")).
					Times(1).
					Return(nil, plumbing.ErrReferenceNotFound)
				r.EXPECT().Remotes().
					Times(1).
					Return([]*gg.Remote{
						gg.NewRemote(nil, &config.RemoteConfig{
							Name: "origin",
						}),
					}, nil)
				r.EXPECT().ResolveRevision(plumbing.Revision("origin/CR-1234")).
					Times(1).
					Return(nil, plumbing.ErrReferenceNotFound)
			},
		},
	}
	origWorktree := worktree
	defer func() { worktree = origWorktree }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockwt := mocks.NewMockWorktree(ctrl)
			mockRepo := mocks.NewMockRepository(ctrl)
			worktree = func(r gogit.Repository) (gogit.Worktree, error) {
				return mockwt, nil
			}
			mockwt.EXPECT().Checkout(&gg.CheckoutOptions{Hash: plumbing.NewHash(tt.hash)}).
				Return(nil).
				AnyTimes()

			tt.beforeFn(mockRepo)
			r := &repo{Repository: mockRepo}

			if err := r.checkoutRef(tt.ref); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("repo.checkoutRef() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}
		})
	}
}

func Test_repo_checkoutBranch(t *testing.T) {
	tests := map[string]struct {
		ref               string
		createIfNotExists bool
		wantErr           string
		beforeFn          func(*mocks.MockRepository, *mocks.MockWorktree)
	}{
		"Should checkout a specific branch": {
			ref: "test",
			beforeFn: func(_ *mocks.MockRepository, wt *mocks.MockWorktree) {
				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("test"),
				}).
					Times(1).
					Return(nil)
			},
		},
		"Should fail if Checkout fails without create": {
			ref:     "CR-1234",
			wantErr: plumbing.ErrReferenceNotFound.Error(),
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree) {
				r.EXPECT().Remotes().
					Times(1).
					Return([]*gg.Remote{
						gg.NewRemote(nil, &config.RemoteConfig{
							Name: "origin",
						}),
					}, nil)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("CR-1234"),
				}).
					Times(1).
					Return(plumbing.ErrReferenceNotFound)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewRemoteReferenceName("origin", "CR-1234"),
				}).
					Times(1).
					Return(plumbing.ErrReferenceNotFound)
			},
		},
		"Should fail if Remotes fails": {
			ref:     "CR-1234",
			wantErr: "some error",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree) {
				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("CR-1234"),
				}).
					Times(1).
					Return(plumbing.ErrReferenceNotFound)

				r.EXPECT().Remotes().
					Times(1).
					Return(nil, fmt.Errorf("some error"))
			},
		},
		"Should fail if repo has no remotes": {
			ref:     "CR-1234",
			wantErr: ErrNoRemotes.Error(),
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree) {
				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("CR-1234"),
				}).
					Times(1).
					Return(plumbing.ErrReferenceNotFound)

				r.EXPECT().Remotes().
					Times(1).
					Return([]*gg.Remote{}, nil)
			},
		},
		"Should create local branch if succeeded to checkout remote branch": {
			ref: "CR-1234",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree) {
				r.EXPECT().Remotes().
					Times(1).
					Return([]*gg.Remote{
						gg.NewRemote(nil, &config.RemoteConfig{
							Name: "origin",
						}),
					}, nil)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("CR-1234"),
				}).
					Times(1).
					Return(plumbing.ErrReferenceNotFound)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewRemoteReferenceName("origin", "CR-1234"),
				}).
					Times(1).
					Return(nil)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("CR-1234"),
					Create: true,
				}).
					Times(1).
					Return(nil)
			},
		},
		"Should create local branch if remote branch is not found and create is true": {
			ref:               "CR-1234",
			createIfNotExists: true,
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree) {
				r.EXPECT().Remotes().
					Times(1).
					Return([]*gg.Remote{
						gg.NewRemote(nil, &config.RemoteConfig{
							Name: "origin",
						}),
					}, nil)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("CR-1234"),
				}).
					Times(1).
					Return(plumbing.ErrReferenceNotFound)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewRemoteReferenceName("origin", "CR-1234"),
				}).
					Times(1).
					Return(plumbing.ErrReferenceNotFound)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("CR-1234"),
					Create: true,
				}).
					Times(1).
					Return(nil)
			},
		},
		"Should fail if cannot checkout remote branch for some reason": {
			ref:     "CR-1234",
			wantErr: "some error",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree) {
				r.EXPECT().Remotes().
					Times(1).
					Return([]*gg.Remote{
						gg.NewRemote(nil, &config.RemoteConfig{
							Name: "origin",
						}),
					}, nil)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName("CR-1234"),
				}).
					Times(1).
					Return(plumbing.ErrReferenceNotFound)

				wt.EXPECT().Checkout(&gg.CheckoutOptions{
					Branch: plumbing.NewRemoteReferenceName("origin", "CR-1234"),
				}).
					Times(1).
					Return(fmt.Errorf("some error"))
			},
		},
	}
	origWorktree := worktree
	defer func() { worktree = origWorktree }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockRepository(ctrl)
			mockWT := mocks.NewMockWorktree(ctrl)
			tt.beforeFn(mockRepo, mockWT)

			r := &repo{Repository: mockRepo}
			worktree = func(r gogit.Repository) (gogit.Worktree, error) {
				return mockWT, nil
			}

			if err := r.checkoutBranch(tt.ref, tt.createIfNotExists); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("repo.checkoutBranch() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}
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
					usage: "The git provider, one of: azure|gitea|github|gitlab",
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

func Test_createRepo(t *testing.T) {
	tests := map[string]struct {
		opts    *CloneOptions
		want    string
		wantErr string
	}{
		"Should create new repository": {
			opts: &CloneOptions{
				Repo:     "https://github.com/owner/name.git",
				Provider: "github",
				Auth: Auth{
					Username: "username",
					Password: "password",
				},
				provider: &mockProvider{func(orgRepo string) (string, error) {
					assert.Equal(t, "owner/name", orgRepo)
					return "https://github.com/owner/name.git", nil
				}, nil},
			},
			want: "https://github.com/owner/name.git",
		},
		"Should infer correct provider type from repo url": {
			opts: &CloneOptions{
				Repo: "https://github.com/owner/name.git",
				provider: &mockProvider{func(orgRepo string) (string, error) {
					return "https://github.com/owner/name.git", nil
				}, nil},
			},
			want: "https://github.com/owner/name.git",
		},
		"Should succesfully parse owner and name for long orgRepos": {
			opts: &CloneOptions{
				Repo: "https://github.com/foo22/bar/fizz.git",
				provider: &mockProvider{func(orgRepo string) (string, error) {
					assert.Equal(t, "foo22/bar/fizz", orgRepo)
					return "https://github.com/foo22/bar/fizz.git", nil
				}, nil},
			},
			want: "https://github.com/foo22/bar/fizz.git",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
		beforeFn   func(r *mocks.MockRepository, wt *mocks.MockWorktree, p *mockProvider)
	}{
		"Success": {
			branchName: "",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree, _ *mockProvider) {
				hash := plumbing.NewHash("3992c4")
				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "user",
						Email: "email",
					},
				}

				r.EXPECT().ConfigScoped(gomock.Any()).
					Times(1).
					Return(config, nil)
				wt.EXPECT().Commit("test", gomock.Any()).
					Times(1).
					Return(hash, nil)
				wt.EXPECT().AddGlob(gomock.Any()).
					Times(1).
					Return(nil)
			},
		},
		"Success - author info from provider": {
			branchName: "",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree, _ *mockProvider) {
				hash := plumbing.NewHash("3992c4")
				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "",
						Email: "",
					},
				}

				r.EXPECT().ConfigScoped(gomock.Any()).
					Times(1).
					Return(config, nil)
				wt.EXPECT().Commit("test", gomock.Any()).
					Times(1).
					Return(hash, nil)
				wt.EXPECT().AddGlob(gomock.Any()).
					Times(1).
					Return(nil)
			},
		},
		"Error - getAuthor fails": {
			branchName: "test",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree, p *mockProvider) {
				p.getAuthor = func() (string, string, error) {
					return "", "", fmt.Errorf("some error")
				}
				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "",
						Email: "",
					},
				}

				r.EXPECT().ConfigScoped(gomock.Any()).
					Times(1).
					Return(config, nil)
				wt.EXPECT().Commit(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantErr: "failed to get author information: some error",
		},
		"Error - no name and email": {
			branchName: "test",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree, p *mockProvider) {
				p.getAuthor = func() (string, string, error) {
					return "", "", nil
				}
				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "",
						Email: "",
					},
				}

				r.EXPECT().ConfigScoped(gomock.Any()).
					Times(1).
					Return(config, nil)
				wt.EXPECT().Commit(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantErr: "missing required author information in git config, make sure your git config contains a 'user.name' and 'user.email'",
		},
		"Error - ConfigScope fails": {
			branchName: "test",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree, _ *mockProvider) {
				r.EXPECT().ConfigScoped(gomock.Any()).
					Times(1).
					Return(nil, fmt.Errorf("test Config error"))
				wt.EXPECT().Commit(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantErr: "failed to get gitconfig: test Config error",
		},
		"Error - AddGlob fails": {
			branchName: "test",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree, _ *mockProvider) {
				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "name",
						Email: "email",
					},
				}

				r.EXPECT().ConfigScoped(gomock.Any()).
					Times(1).
					Return(config, nil)
				wt.EXPECT().AddGlob(gomock.Any()).
					Times(1).
					Return(fmt.Errorf("add glob error"))
				wt.EXPECT().Commit(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantErr: "add glob error",
		},
		"Error - Commit fails": {
			branchName: "test",
			beforeFn: func(r *mocks.MockRepository, wt *mocks.MockWorktree, _ *mockProvider) {
				config := &config.Config{
					User: struct {
						Name  string
						Email string
					}{
						Name:  "name",
						Email: "email",
					},
				}

				r.EXPECT().ConfigScoped(gomock.Any()).
					Times(1).
					Return(config, nil)
				wt.EXPECT().AddGlob(gomock.Any()).
					Times(1).
					Return(nil)
				wt.EXPECT().Commit("test", gomock.Any()).
					Times(1).
					Return(plumbing.Hash{}, fmt.Errorf("test Config error"))
			},
			wantErr: "test Config error",
		},
	}

	orgWorktree := worktree
	defer func() { worktree = orgWorktree }()
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mocks.NewMockRepository(ctrl)
			mockWt := mocks.NewMockWorktree(ctrl)
			mockProvider := &mockProvider{}

			r := &repo{Repository: mockRepo, provider: mockProvider}
			worktree = func(r gogit.Repository) (gogit.Worktree, error) {
				return mockWt, nil
			}

			tt.beforeFn(mockRepo, mockWt, mockProvider)

			got, err := r.commit(context.Background(), &PushOptions{CommitMsg: "test"})

			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("r.commit() error = %v", err)
				}

				return
			}

			hash := plumbing.NewHash("3992c4")
			assert.Equal(t, got, &hash)
		})
	}
}
