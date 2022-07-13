package application

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	fsmocks "github.com/argoproj-labs/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/kube"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	kusttypes "sigs.k8s.io/kustomize/api/types"
)

func bootstrapMockFS() fs.FS {
	repofs := fs.Create(memfs.New())
	clusterResConf := &ClusterResConfig{Name: store.Default.ClusterContextName, Server: store.Default.DestServer}
	clusterResPath := repofs.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, store.Default.ClusterContextName+".json")
	_ = repofs.WriteYamls(clusterResPath, clusterResConf)
	return repofs
}

func Test_newKustApp(t *testing.T) {
	orgGenerateManifests := generateManifests
	defer func() { generateManifests = orgGenerateManifests }()
	generateManifests = func(k *kusttypes.Kustomization) ([]byte, error) {
		return []byte("foo"), nil
	}

	tests := map[string]struct {
		opts              *CreateOptions
		srcRepoURL        string
		srcTargetRevision string
		srcRepoRoot       string
		projectName       string
		wantErr           string
		assertFn          func(*testing.T, *kustApp)
	}{
		"Should fail when there is no app specifier": {
			opts: &CreateOptions{
				AppName: "name",
			},
			wantErr: ErrEmptyAppSpecifier.Error(),
		},
		"Should fail when there is no app name": {
			opts: &CreateOptions{
				AppSpecifier: "app",
			},
			wantErr: ErrEmptyAppName.Error(),
		},
		"Should fail when there is no project name": {
			opts: &CreateOptions{
				AppSpecifier: "app",
				AppName:      "name",
			},
			wantErr: ErrEmptyProjectName.Error(),
		},
		"Should fail when there is an invalid installation mode": {
			opts: &CreateOptions{
				AppSpecifier:     "app",
				AppName:          "name",
				InstallationMode: "foo",
			},
			projectName: "project",
			wantErr:     "unknown installation mode: foo",
		},
		"Should create a correct base kustomization and config.json": {
			opts: &CreateOptions{
				AppSpecifier: "app",
				AppName:      "name",
			},
			srcRepoURL:        "github.com/owner/repo",
			srcTargetRevision: "branch",
			projectName:       "project",
			assertFn: func(t *testing.T, a *kustApp) {
				assert.Equal(t, "app", a.base.Resources[0])
				assert.Equal(t, "../../base", a.overlay.Resources[0])
				assert.Nil(t, a.namespace)
				assert.Nil(t, a.manifests)
				assert.True(t, reflect.DeepEqual(&Config{
					AppName:           "name",
					UserGivenName:     "name",
					SrcPath:           filepath.Join(store.Default.AppsDir, "name", store.Default.OverlaysDir, "project"),
					SrcRepoURL:        "github.com/owner/repo",
					SrcTargetRevision: "branch",
				}, a.config))
			},
		},
		"Should create a flat install.yaml when InstallationModeFlat is set": {
			opts: &CreateOptions{
				AppSpecifier:     "app",
				AppName:          "name",
				InstallationMode: InstallationModeFlat,
				DestNamespace:    "namespace",
			},
			srcRepoURL:        "github.com/owner/repo",
			srcTargetRevision: "branch",
			projectName:       "project",
			assertFn: func(t *testing.T, a *kustApp) {
				assert.Equal(t, "install.yaml", a.base.Resources[0])
				assert.Equal(t, []byte("foo"), a.manifests)
				assert.Equal(t, 1, len(a.overlay.Resources))
				assert.Equal(t, "../../base", a.overlay.Resources[0])
				assert.Equal(t, "namespace", a.namespace.ObjectMeta.Name)
				assert.True(t, reflect.DeepEqual(&Config{
					AppName:           "name",
					UserGivenName:     "name",
					DestNamespace:     "namespace",
					SrcPath:           filepath.Join(store.Default.AppsDir, "name", store.Default.OverlaysDir, "project"),
					SrcRepoURL:        "github.com/owner/repo",
					SrcTargetRevision: "branch",
				}, a.config))
			},
		},
		"Should have labels in the resulting config.json": {
			opts: &CreateOptions{
				AppSpecifier: "app",
				AppName:      "name",
				Labels: map[string]string{
					"key": "value",
				},
			},
			srcRepoURL:        "github.com/owner/repo",
			srcTargetRevision: "branch",
			projectName:       "project",
			assertFn: func(t *testing.T, a *kustApp) {
				assert.True(t, reflect.DeepEqual(&Config{
					AppName:           "name",
					UserGivenName:     "name",
					SrcPath:           filepath.Join(store.Default.AppsDir, "name", store.Default.OverlaysDir, "project"),
					SrcRepoURL:        "github.com/owner/repo",
					SrcTargetRevision: "branch",
					Labels: map[string]string{
						"key": "value",
					},
				}, a.config))
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			app, err := newKustApp(tt.opts, tt.projectName, tt.srcRepoURL, tt.srcTargetRevision, tt.srcRepoRoot)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}

				return
			}

			tt.assertFn(t, app)
		})
	}
}

func Test_writeFile(t *testing.T) {
	type args struct {
		root string
		path string
		name string
		data []byte
	}
	tests := map[string]struct {
		args     args
		assertFn func(t *testing.T, repofs fs.FS, exists bool, err error)
		beforeFn func(t *testing.T, repofs fs.FS) fs.FS
	}{
		"On Root": {
			args: args{
				path: "foo/bar",
				name: "test",
				data: []byte("data"),
			},
			assertFn: func(t *testing.T, repofs fs.FS, exists bool, ret error) {
				assert.NoError(t, ret)

				f, err := repofs.Open("/foo/bar")
				assert.NoError(t, err)
				d, err := ioutil.ReadAll(f)
				assert.NoError(t, err)

				assert.Equal(t, d, []byte("data"))
				assert.False(t, exists)
			},
		},
		"With Chroot": {
			args: args{
				root: "someroot",
				path: "foo/bar",
				name: "test",
				data: []byte("data2"),
			},
			assertFn: func(t *testing.T, repofs fs.FS, exists bool, ret error) {
				assert.NoError(t, ret)

				assert.Equal(t, "/someroot", repofs.Root())
				f, err := repofs.Open("/foo/bar")
				assert.NoError(t, err)
				d, err := ioutil.ReadAll(f)
				assert.NoError(t, err)

				assert.Equal(t, d, []byte("data2"))
				assert.False(t, exists)
			},
		},
		"File exists": {
			args: args{
				path: "foo/bar",
				name: "test",
				data: []byte("data2"),
			},
			beforeFn: func(_ *testing.T, repofs fs.FS) fs.FS {
				_ = billyUtils.WriteFile(repofs, "/foo/bar", []byte("data"), 0666)
				return repofs
			},
			assertFn: func(t *testing.T, _ fs.FS, exists bool, ret error) {
				assert.NoError(t, ret)
				assert.True(t, exists)
			},
		},
		"Write error": {
			args: args{
				path: "foo/bar",
				name: "test",
				data: []byte("data2"),
			},
			beforeFn: func(t *testing.T, _ fs.FS) fs.FS {
				mfs := fsmocks.NewMockFS(gomock.NewController(t))
				mfs.EXPECT().CheckExistsOrWrite(gomock.Any(), gomock.Any()).
					Times(1).
					Return(false, errors.New("error"))
				mfs.EXPECT().Root().
					Times(1).
					Return("/")
				mfs.EXPECT().Join(gomock.Any(), gomock.Any()).
					Times(1).
					Return("/foo/bar")
				return mfs
			},
			assertFn: func(t *testing.T, _ fs.FS, _ bool, ret error) {
				assert.Error(t, ret)
				assert.EqualError(t, ret, "failed to create 'test' file at '/foo/bar': error")
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			repofs := fs.Create(memfs.New())
			if tt.args.root != "" {
				bfs, _ := repofs.Chroot(tt.args.root)
				repofs = fs.Create(bfs)
			}

			if tt.beforeFn != nil {
				repofs = tt.beforeFn(t, repofs)
			}

			got, err := writeFile(repofs, tt.args.path, tt.args.name, tt.args.data)
			tt.assertFn(t, repofs, got, err)
		})
	}
}

func Test_kustCreateFiles(t *testing.T) {
	tests := map[string]struct {
		beforeFn func() (app *kustApp, repofs fs.FS, appsfs fs.FS, projectName string)
		assertFn func(t *testing.T, repofs fs.FS, appsfs fs.FS, err error)
	}{
		"Should create all files for a simple application": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName:    "app",
							DestServer: store.Default.DestServer,
						},
					},
				}
				repofs := bootstrapMockFS()
				return app, repofs, repofs, "project"
			},
			assertFn: func(t *testing.T, repofs fs.FS, _ fs.FS, err error) {
				assert.NoError(t, err)

				assert.True(t, repofs.ExistsOrDie(repofs.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project", "config.json")), "overlay config should exist")
				assert.True(t, repofs.ExistsOrDie(repofs.Join(store.Default.AppsDir, "app", store.Default.BaseDir, "kustomization.yaml")), "base kustomization should exist")
				assert.False(t, repofs.ExistsOrDie(repofs.Join(store.Default.AppsDir, "app", store.Default.BaseDir, "install.yaml")), "install file should not exist")
				assert.True(t, repofs.ExistsOrDie(repofs.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project", "kustomization.yaml")), "overlay kustomization should exist")
				assert.False(t, repofs.ExistsOrDie(repofs.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project", "namespace.yaml")), "overlay namespace should not exist")
			},
		},
		"Should create all files in two separate filesystems": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName:    "app",
							DestServer: store.Default.DestServer,
						},
					},
				}
				repofs := bootstrapMockFS()
				appsfs := fs.Create(memfs.New())
				return app, repofs, appsfs, "project"
			},
			assertFn: func(t *testing.T, repofs fs.FS, appsfs fs.FS, err error) {
				assert.NoError(t, err)

				// in repofs - only config.json
				assert.True(t, repofs.ExistsOrDie(repofs.Join(store.Default.AppsDir, "app", "project", "config.json")), "app config should exist in repofs")
				assert.False(t, repofs.ExistsOrDie(repofs.Join(store.Default.AppsDir, "app", store.Default.BaseDir)), "base directory should not exist in repofs")
				assert.False(t, repofs.ExistsOrDie(repofs.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)), "overlay directory should not exist in repofs")

				// in appsfs - only base and overlay kustomization
				assert.True(t, appsfs.ExistsOrDie(appsfs.Join(store.Default.AppsDir, "app", store.Default.BaseDir, "kustomization.yaml")), "base kustomization should exist in appsfs")
				assert.False(t, appsfs.ExistsOrDie(appsfs.Join(store.Default.AppsDir, "app", store.Default.BaseDir, "install.yaml")), "install file should not exist in appsfs")
				assert.True(t, appsfs.ExistsOrDie(appsfs.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project", "kustomization.yaml")), "overlay kustomization should exist in appsfs")
				assert.False(t, appsfs.ExistsOrDie(appsfs.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project", "namespace.yaml")), "overlay namespace should not exist in appsfs")
			},
		},
		"Should create install.yaml when manifests exist": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName:    "app",
							DestServer: store.Default.DestServer,
						},
					},
					manifests: []byte("some manifests"),
				}
				repofs := bootstrapMockFS()
				return app, repofs, repofs, "project"
			},
			assertFn: func(t *testing.T, repofs fs.FS, _ fs.FS, err error) {
				assert.NoError(t, err)
				installFile := repofs.Join(store.Default.AppsDir, "app", store.Default.BaseDir, "install.yaml")
				assert.True(t, repofs.ExistsOrDie(installFile), "install file should exist")
				data, _ := repofs.ReadFile(installFile)
				assert.Equal(t, "some manifests", string(data))
			},
		},
		"Should create namespace.yaml on the correct cluster resources directory when needed": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName:    "app",
							DestServer: store.Default.DestServer,
						},
					},
					namespace: kube.GenerateNamespace("foo", nil),
				}
				repofs := bootstrapMockFS()
				return app, repofs, repofs, "project"
			},
			assertFn: func(t *testing.T, repofs fs.FS, _ fs.FS, err error) {
				assert.NoError(t, err)
				path := repofs.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, store.Default.ClusterContextName, "foo-ns.yaml")
				ns, err := repofs.ReadFile(path)
				assert.NoError(t, err, "namespace file should exist in cluster-resources dir")
				namespace := &v1.Namespace{}
				assert.NoError(t, yaml.Unmarshal(ns, namespace))
				assert.Equal(t, "foo", namespace.Name)
			},
		},
		"Should fail if trying to install an application with destServer that is not configured yet": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName:    "app",
							DestServer: "foo",
						},
					},
				}
				repofs := bootstrapMockFS()
				return app, repofs, repofs, "project"
			},
			assertFn: func(t *testing.T, _ fs.FS, _ fs.FS, err error) {
				assert.Error(t, err, "cluster 'foo' is not configured yet, you need to create a project that uses this cluster first")
			},
		},
		"Should fail when base kustomization is different from kustRes": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName:    "app",
							DestServer: store.Default.DestServer,
						},
					},
					base: &kusttypes.Kustomization{
						TypeMeta: kusttypes.TypeMeta{
							APIVersion: kusttypes.KustomizationVersion,
							Kind:       kusttypes.KustomizationKind,
						},
						Resources: []string{"github.com/owner/repo?ref=v1.2.3"},
					},
				}
				origBase := &kusttypes.Kustomization{
					TypeMeta: kusttypes.TypeMeta{
						APIVersion: kusttypes.KustomizationVersion,
						Kind:       kusttypes.KustomizationKind,
					},
					Resources: []string{"github.com/owner/different_repo?ref=v1.2.3"},
				}
				repofs := bootstrapMockFS()
				_ = repofs.WriteYamls(repofs.Join(store.Default.AppsDir, "app", store.Default.BaseDir, "kustomization.yaml"), origBase)
				return app, repofs, repofs, "project"
			},
			assertFn: func(t *testing.T, _ fs.FS, _ fs.FS, err error) {
				assert.ErrorIs(t, err, ErrAppCollisionWithExistingBase)
			},
		},
		"Should fail when overlay already exists": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				origBase := &kusttypes.Kustomization{
					TypeMeta: kusttypes.TypeMeta{
						APIVersion: kusttypes.KustomizationVersion,
						Kind:       kusttypes.KustomizationKind,
					},
					Resources: []string{"github.com/owner/different_repo?ref=v1.2.3"},
				}
				overlay := &kusttypes.Kustomization{
					TypeMeta: kusttypes.TypeMeta{
						APIVersion: kusttypes.KustomizationVersion,
						Kind:       kusttypes.KustomizationKind,
					},
					Resources: []string{"../../base"},
				}
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName:    "app",
							DestServer: store.Default.DestServer,
						},
					},
					base: origBase,
				}
				repofs := bootstrapMockFS()
				_ = repofs.WriteYamls(repofs.Join(store.Default.AppsDir, "app", store.Default.BaseDir, "kustomization.yaml"), origBase)
				_ = repofs.WriteYamls(repofs.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project", "kustomization.yaml"), overlay)
				return app, repofs, repofs, "project"
			},
			assertFn: func(t *testing.T, _ fs.FS, _ fs.FS, err error) {
				assert.ErrorIs(t, err, ErrAppAlreadyInstalledOnProject)
			},
		},
		"Should fail when failing to find original appRepo": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName: "app",
						},
					},
				}
				repofs := bootstrapMockFS()
				appsfs := fs.Create(memfs.New())
				_ = repofs.MkdirAll(repofs.Join(store.Default.AppsDir, "app"), 0666)
				return app, repofs, appsfs, "project"
			},
			assertFn: func(t *testing.T, _ fs.FS, _ fs.FS, err error) {
				assert.EqualError(t, err, "Failed getting app repo: Application 'app' has no overlays")
			},
		},
		"Should fail when app exists on another repo": {
			beforeFn: func() (*kustApp, fs.FS, fs.FS, string) {
				app := &kustApp{
					baseApp: baseApp{
						opts: &CreateOptions{
							AppName: "app",
						},
					},
				}
				config := &Config{
					AppName:    "app",
					SrcRepoURL: "https://github.com/owner/other_name.git",
				}
				repofs := bootstrapMockFS()
				appsfs := fs.Create(memfs.New())
				_ = repofs.WriteJson(repofs.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project", "config.json"), config)
				return app, repofs, appsfs, "project"
			},
			assertFn: func(t *testing.T, _ fs.FS, _ fs.FS, err error) {
				assert.EqualError(t, err, "an application with the same name already exists in 'https://github.com/owner/other_name.git', consider choosing a different name")
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			app, repofs, appsfs, projectName := tt.beforeFn()
			err := app.CreateFiles(repofs, appsfs, projectName)
			tt.assertFn(t, repofs, appsfs, err)
		})
	}
}

func TestInferAppType(t *testing.T) {
	tests := map[string]struct {
		want     string
		beforeFn func() fs.FS
	}{
		"Should return ksonnet if required files are present": {
			want: "ksonnet",
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "app.yaml", []byte{}, 0666)
				_ = billyUtils.WriteFile(memfs, "components/params.libsonnet", []byte{}, 0666)
				return fs.Create(memfs)
			},
		},
		"Should not return ksonnet if 'app.yaml' is missing": {
			want: AppTypeDirectory,
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "components/params.libsonnet", []byte{}, 0666)
				return fs.Create(memfs)
			},
		},
		"Should not return ksonnet if 'components/params.libsonnet' is missing": {
			want: AppTypeDirectory,
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "app.yaml", []byte{}, 0666)
				return fs.Create(memfs)
			},
		},
		"Should return ksonnet as the highest priority": {
			want: "ksonnet",
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "app.yaml", []byte{}, 0666)
				_ = billyUtils.WriteFile(memfs, "components/params.libsonnet", []byte{}, 0666)
				_ = billyUtils.WriteFile(memfs, "Chart.yaml", []byte{}, 0666)
				_ = billyUtils.WriteFile(memfs, "kustomization.yaml", []byte{}, 0666)
				return fs.Create(memfs)
			},
		},
		"Should return helm if 'Chart.yaml' is present": {
			want: "helm",
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "Chart.yaml", []byte{}, 0666)
				return fs.Create(memfs)
			},
		},
		"Should return helm as a higher priority than kustomize": {
			want: "helm",
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "Chart.yaml", []byte{}, 0666)
				_ = billyUtils.WriteFile(memfs, "kustomization.yaml", []byte{}, 0666)
				return fs.Create(memfs)
			},
		},
		"Should return kustomize if 'kustomization.yaml' file is present": {
			want: AppTypeKustomize,
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "kustomization.yaml", []byte{}, 0666)
				return fs.Create(memfs)
			},
		},
		"Should return kustomize if 'kustomization.yml' file is present": {
			want: AppTypeKustomize,
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "kustomization.yml", []byte{}, 0666)
				return fs.Create(memfs)
			},
		},
		"Should return kustomize if 'Kustomization' folder is present": {
			want: AppTypeKustomize,
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = memfs.MkdirAll("Kustomization", 0666)
				return fs.Create(memfs)
			},
		},
		"Should return dir if no other match": {
			want: AppTypeDirectory,
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				return fs.Create(memfs)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repofs := tt.beforeFn()
			if got := InferAppType(repofs); got != tt.want {
				t.Errorf("InferAppType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteFromProject(t *testing.T) {
	tests := map[string]struct {
		wantErr  string
		beforeFn func() fs.FS
		assertFn func(*testing.T, fs.FS)
	}{
		"Should remove entire app folder, if it contains only one overlay": {
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				return fs.Create(memfs)
			},
			assertFn: func(t *testing.T, repofs fs.FS) {
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)))
			},
		},
		"Should delete just the overlay, if there are more": {
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project2"), 0666)
				return fs.Create(memfs)
			},
			assertFn: func(t *testing.T, repofs fs.FS) {
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project")))
			},
		},
		"Should remove directory apps": {
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", "project"), 0666)
				return fs.Create(memfs)
			},
			assertFn: func(t *testing.T, repofs fs.FS) {
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
		"Should not delete anything, if kust app is not in project": {
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project2"), 0666)
				return fs.Create(memfs)
			},
			assertFn: func(t *testing.T, repofs fs.FS) {
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
		"Should not delete anything, if dir app is not in project": {
			beforeFn: func() fs.FS {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", "project2"), 0666)
				return fs.Create(memfs)
			},
			assertFn: func(t *testing.T, repofs fs.FS) {
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repofs := tt.beforeFn()
			if err := DeleteFromProject(repofs, "app", "project"); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("DeleteFromProject() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, repofs)
			}
		})
	}
}

func Test_newDirApp(t *testing.T) {
	tests := map[string]struct {
		opts *CreateOptions
		want *dirApp
	}{
		"Should create a simple app with requested fields": {
			opts: &CreateOptions{
				AppName:       "fooapp",
				AppSpecifier:  "github.com/foo/bar/somepath/in/repo?ref=v0.1.2",
				DestNamespace: "fizz",
				DestServer:    "buzz",
			},
			want: &dirApp{

				dirConfig: &dirConfig{
					Config: Config{
						AppName:           "fooapp",
						UserGivenName:     "fooapp",
						DestNamespace:     "fizz",
						DestServer:        "buzz",
						SrcRepoURL:        "https://github.com/foo/bar.git",
						SrcTargetRevision: "v0.1.2",
						SrcPath:           "somepath/in/repo",
					},
				},
			},
		},
		"Should use the correct path, when no path is supplied": {
			opts: &CreateOptions{
				AppName:      "fooapp",
				AppSpecifier: "github.com/foo/bar",
			},
			want: &dirApp{
				dirConfig: &dirConfig{
					Config: Config{
						AppName:           "fooapp",
						UserGivenName:     "fooapp",
						DestNamespace:     "",
						DestServer:        "",
						SrcRepoURL:        "https://github.com/foo/bar.git",
						SrcTargetRevision: "",
						SrcPath:           ".",
					},
				},
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			if got := newDirApp(tt.opts); !reflect.DeepEqual(got.dirConfig, tt.want.dirConfig) {
				t.Errorf("newDirApp() = %+v, want %+v", got.dirConfig, tt.want.dirConfig)
			}
		})
	}
}

func Test_dirApp_CreateFiles(t *testing.T) {
	tests := map[string]struct {
		projectName string
		app         *dirApp
		beforeFn    func() fs.FS
		assertFn    func(*testing.T, fs.FS, error)
	}{
		"Should fail if an app with the same name already exist": {
			projectName: "project",
			app: &dirApp{
				baseApp: baseApp{
					opts: &CreateOptions{
						AppName:       "foo",
						AppSpecifier:  "github.com/foo/bar/path",
						DestNamespace: "default",
						DestServer:    store.Default.DestServer,
					},
				},
			},
			beforeFn: func() fs.FS {
				fs := bootstrapMockFS()
				appPath := fs.Join(store.Default.AppsDir, "foo", "project", "DUMMY")
				_ = billyUtils.WriteFile(fs, appPath, []byte{}, 0666)
				return fs
			},
			assertFn: func(t *testing.T, _ fs.FS, err error) {
				assert.Error(t, err, ErrAppAlreadyInstalledOnProject)
			},
		},
		"Should not create namespace if app namespace is 'default'": {
			app: &dirApp{
				baseApp: baseApp{
					opts: &CreateOptions{
						AppName:       "foo",
						AppSpecifier:  "github.com/foo/bar/path",
						DestNamespace: "default",
						DestServer:    store.Default.DestServer,
					},
				},
			},
			beforeFn: bootstrapMockFS,
			assertFn: func(t *testing.T, repofs fs.FS, err error) {
				assert.NoError(t, err)
				exists, err := repofs.Exists(repofs.Join(
					store.Default.BootsrtrapDir,
					store.Default.ClusterResourcesDir,
					store.Default.ClusterContextName,
					"default-ns.yaml",
				))
				assert.NoError(t, err)
				assert.False(t, exists)
			},
		},
		"Should fail with destServer that is not configured yet": {
			app: &dirApp{
				baseApp: baseApp{
					opts: &CreateOptions{
						AppName:       "foo",
						AppSpecifier:  "github.com/foo/bar/path",
						DestNamespace: "default",
						DestServer:    "some.new.server",
					},
				},
			},
			beforeFn: bootstrapMockFS,
			assertFn: func(t *testing.T, _ fs.FS, err error) {
				assert.Error(t, err, "cluster 'some.new.server' is not configured yet, you need to create a project that uses this cluster first")
			},
		},
		"Should create namespace in correct cluster resources dir": {
			app: &dirApp{
				baseApp: baseApp{
					opts: &CreateOptions{
						AppName:       "foo",
						AppSpecifier:  "github.com/foo/bar/path",
						DestNamespace: "buzz",
						DestServer:    store.Default.DestServer,
					},
				},
			},
			beforeFn: bootstrapMockFS,
			assertFn: func(t *testing.T, repofs fs.FS, _ error) {
				exists, err := repofs.Exists(repofs.Join(
					store.Default.BootsrtrapDir,
					store.Default.ClusterResourcesDir,
					store.Default.ClusterContextName,
					"buzz-ns.yaml",
				))
				assert.NoError(t, err)
				assert.True(t, exists)
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			repofs := tt.beforeFn()
			tt.assertFn(t, repofs, tt.app.CreateFiles(repofs, repofs, tt.projectName))
		})
	}
}
