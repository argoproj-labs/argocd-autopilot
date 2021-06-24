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
				tt.opts.FS = fs.Create(memfs.New())
			}

			gotRepo, gotFS, err := tt.opts.GetRepo(context.Background())
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
		prefix      string
		wantedFlags []flag
	}{
		"Should create flags without a prefix": {
			wantedFlags: []flag{
				{
					name:      "git-token",
					shorthand: "t",
					usage:     "Your git provider api token [GIT_TOKEN]",
					required:  true,
				},
				{
					name:  "provider",
					usage: "The git provider, one of: github|github.com",
				},
				{
					name:     "repo",
					usage:    "Repository URL [GIT_REPO]",
					required: true,
				},
			},
		},
		"Should create flags with a prefix": {
			prefix: "prefix-",
			wantedFlags: []flag{
				{
					name:  "prefix-git-token",
					usage: "Your git provider api token [PREFIX_GIT_TOKEN]",
				},
				{
					name:  "prefix-provider",
					usage: "The git provider, one of: github|github.com",
				},
				{
					name:  "prefix-repo",
					usage: "Repository URL [PREFIX_GIT_REPO]",
				},
			},
		},
		"Should automatically add a dash to prefix": {
			prefix: "prefix",
			wantedFlags: []flag{
				{
					name:  "prefix-git-token",
					usage: "Your git provider api token [PREFIX_GIT_TOKEN]",
				},
				{
					name:  "prefix-provider",
					usage: "The git provider, one of: github|github.com",
				},
				{
					name:  "prefix-repo",
					usage: "Repository URL [PREFIX_GIT_REPO]",
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			cmd := &cobra.Command{}
			_ = AddFlags(cmd, nil, tt.prefix)
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
