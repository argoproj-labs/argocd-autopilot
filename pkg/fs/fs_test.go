package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	SampleJson struct {
		Name string `json:"name"`
	}
)

func TestCreate(t *testing.T) {
	tests := map[string]struct {
		bfs      billy.Filesystem
		beforeFn func(fs billy.Filesystem) FS
	}{
		"should create FS": {
			bfs: memfs.New(),
			beforeFn: func(bfs billy.Filesystem) FS {
				return &fsimpl{bfs}
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			want := tt.beforeFn(tt.bfs)
			if got := Create(tt.bfs); !reflect.DeepEqual(got, want) {
				t.Errorf("Create() = %v, want %v", got, want)
			}
		})
	}
}

func Test_fsimpl_Exists(t *testing.T) {
	tests := map[string]struct {
		beforeFn func(*testing.T) billy.Filesystem
		path     string
		want     bool
		wantErr  bool
	}{
		"Should exist": {
			path: "/foo/noam/bar",
			beforeFn: func(_ *testing.T) billy.Filesystem {
				fs := memfs.New()
				f, err := fs.Create("/foo/noam/bar")
				util.Die(err)
				defer f.Close()
				return fs
			},
			want:    true,
			wantErr: false,
		},
		"Should not exist": {
			path: "/foo/noam/bar",
			beforeFn: func(_ *testing.T) billy.Filesystem {
				return memfs.New()
			},
			want:    false,
			wantErr: false,
		},
		"Should throw error": {
			path: "invalid file path",
			beforeFn: func(t *testing.T) billy.Filesystem {
				m := mocks.NewMockFS(gomock.NewController(t))
				m.EXPECT().Stat("invalid file path").
					Times(1).
					Return(nil, fmt.Errorf("error"))
				return m
			},
			want:    false,
			wantErr: true,
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			fs := &fsimpl{Filesystem: tt.beforeFn(t)}
			got, err := fs.Exists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("fs.Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fs.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fsimpl_CheckExistsOrWrite(t *testing.T) {
	type args struct {
		path string
		data []byte
	}
	tests := map[string]struct {
		args     args
		want     bool
		wantErr  string
		beforeFn func(m *mocks.MockFS)
	}{
		"should exists": {
			args: args{path: "/usr/bar", data: []byte{}},
			want: true,
			beforeFn: func(m *mocks.MockFS) {
				m.EXPECT().Stat("/usr/bar").
					Times(1).
					Return(nil, nil)
			},
		},
		"should error on fail check": {
			args:    args{path: "/usr/bar", data: []byte{}},
			wantErr: "failed to check if file exists on repo at '/usr/bar': some error",
			beforeFn: func(m *mocks.MockFS) {
				m.EXPECT().Stat("/usr/bar").
					Times(1).
					Return(nil, fmt.Errorf("some error"))
			},
		},
		"should fail if WriteFile failed": {
			args:    args{path: "/usr/bar", data: []byte{}},
			want:    false,
			wantErr: "failed to create file at '/usr/bar': " + os.ErrPermission.Error(),
			beforeFn: func(m *mocks.MockFS) {
				m.EXPECT().Stat("/usr/bar").
					Times(1).
					Return(nil, os.ErrNotExist)
				m.EXPECT().OpenFile("/usr/bar", gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, os.ErrPermission)
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockedFS := mocks.NewMockFS(gomock.NewController(t))
			tt.beforeFn(mockedFS)
			fs := Create(mockedFS)
			got, err := fs.CheckExistsOrWrite(tt.args.path, tt.args.data)

			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}

				return
			}

			if got != tt.want {
				t.Errorf("fsimpl.CheckExistsOrWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fsimpl_ExistsOrDie(t *testing.T) {
	tests := map[string]struct {
		path     string
		want     bool
		wantErr  bool
		beforeFn func(m *mocks.MockFS)
	}{
		"should exists if path exists": {
			path:    "/usr/bar",
			want:    true,
			wantErr: false,
			beforeFn: func(m *mocks.MockFS) {
				m.EXPECT().Stat("/usr/bar").
					Times(1).
					Return(nil, nil)
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mock := mocks.NewMockFS(gomock.NewController(t))
			fs := Create(mock)
			tt.beforeFn(mock)
			if got := fs.ExistsOrDie(tt.path); got != tt.want {
				t.Errorf("fsimpl.ExistsOrDie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fsimpl_ReadFile(t *testing.T) {
	tests := map[string]struct {
		filename string
		want     []byte
		wantErr  string
		beforeFn func() FS
	}{
		"Should read file data": {
			filename: "file",
			want:     []byte("some data"),
			beforeFn: func() FS {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "file", []byte("some data"), 0666)
				return Create(memfs)
			},
		},
		"Should fail if file does not exist": {
			filename: "file",
			wantErr:  os.ErrNotExist.Error(),
			beforeFn: func() FS {
				memfs := memfs.New()
				return Create(memfs)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fs := tt.beforeFn()
			got, err := fs.ReadFile(tt.filename)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}

				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_fsimpl_ReadYamls(t *testing.T) {
	tests := map[string]struct {
		o        []interface{}
		wantErr  string
		beforeFn func() FS
		assertFn func(*testing.T, ...interface{})
	}{
		"Should read a simple yaml file": {
			o: []interface{}{
				&corev1.Namespace{},
			},
			beforeFn: func() FS {
				ns := &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace",
					},
				}
				y, _ := yaml.Marshal(ns)
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "filename", y, 0666)
				return Create(memfs)
			},
			assertFn: func(t *testing.T, o ...interface{}) {
				ns := o[0].(*corev1.Namespace)
				assert.Equal(t, "namespace", ns.Name)
			},
		},
		"Should return two manifests when requested": {
			o: []interface{}{
				&corev1.Namespace{},
				&corev1.Namespace{},
			},
			beforeFn: func() FS {
				ns1 := &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace1",
					},
				}
				ns2 := &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace2",
					},
				}
				y1, _ := yaml.Marshal(ns1)
				y2, _ := yaml.Marshal(ns2)
				data := util.JoinManifests(y1, y2)
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "filename", data, 0666)
				return Create(memfs)
			},
			assertFn: func(t *testing.T, o ...interface{}) {
				ns1 := o[0].(*corev1.Namespace)
				ns2 := o[1].(*corev1.Namespace)
				assert.Equal(t, "namespace1", ns1.Name)
				assert.Equal(t, "namespace2", ns2.Name)
			},
		},
		"Should return only the 2nd manifest when requested": {
			o: []interface{}{
				nil,
				&corev1.Namespace{},
			},
			beforeFn: func() FS {
				ns1 := &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace1",
					},
				}
				ns2 := &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace2",
					},
				}
				y1, _ := yaml.Marshal(ns1)
				y2, _ := yaml.Marshal(ns2)
				data := util.JoinManifests(y1, y2)
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "filename", data, 0666)
				return Create(memfs)
			},
			assertFn: func(t *testing.T, o ...interface{}) {
				assert.Nil(t, o[0])
				ns2 := o[1].(*corev1.Namespace)
				assert.Equal(t, "namespace2", ns2.Name)
			},
		},
		"Should fail if file does not exist": {
			o: []interface{}{
				&corev1.Namespace{},
			},
			wantErr: os.ErrNotExist.Error(),
			beforeFn: func() FS {
				memfs := memfs.New()
				return Create(memfs)
			},
		},
		"Should fail if file contains less manifests than expected": {
			o: []interface{}{
				&corev1.Namespace{},
				&corev1.Namespace{},
			},
			wantErr: "expected at least 2 manifests when reading 'filename'",
			beforeFn: func() FS {
				ns := &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace",
					},
				}
				memfs := memfs.New()
				y, _ := yaml.Marshal(ns)
				_ = billyUtils.WriteFile(memfs, "filename", y, 0666)
				return Create(memfs)
			},
		},
		"Should fail if second manifest is corrupted": {
			o: []interface{}{
				&corev1.Namespace{},
				&corev1.Namespace{},
			},
			wantErr: "error unmarshaling JSON: json: cannot unmarshal string into Go value of type v1.Namespace",
			beforeFn: func() FS {
				ns := &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace",
					},
				}
				y, _ := yaml.Marshal(ns)
				memfs := memfs.New()
				data := util.JoinManifests(y, []byte("some data"))
				_ = billyUtils.WriteFile(memfs, "filename", data, 0666)
				return Create(memfs)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fs := tt.beforeFn()
			if err := fs.ReadYamls("filename", tt.o...); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("ReadYamls() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, tt.o...)
			}
		})
	}
}

func Test_fsimpl_WriteYamls(t *testing.T) {
	tests := map[string]struct {
		o        []interface{}
		wantErr  string
		assertFn func(*testing.T, FS)
	}{
		"Should write a simple manifest": {
			o: []interface{}{
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace",
					},
				},
			},
			assertFn: func(t *testing.T, fs FS) {
				data, err := fs.ReadFile("filename")
				assert.NoError(t, err)
				ns := &corev1.Namespace{}
				err = yaml.Unmarshal(data, ns)
				assert.NoError(t, err)
				assert.Equal(t, "namespace", ns.Name)
			},
		},
		"Should write two manifests": {
			o: []interface{}{
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace1",
					},
				},
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace2",
					},
				},
			},
			assertFn: func(t *testing.T, fs FS) {
				data, err := fs.ReadFile("filename")
				assert.NoError(t, err)
				manifests := util.SplitManifests(data)
				ns1 := &corev1.Namespace{}
				ns2 := &corev1.Namespace{}
				err = yaml.Unmarshal(manifests[0], ns1)
				assert.NoError(t, err)
				err = yaml.Unmarshal(manifests[1], ns2)
				assert.NoError(t, err)
				assert.Equal(t, "namespace1", ns1.Name)
				assert.Equal(t, "namespace2", ns2.Name)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fs := Create(memfs.New())
			if err := fs.WriteYamls("filename", tt.o...); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("WriteYamls() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, fs)
			}
		})
	}
}

func Test_fsimpl_ReadJson(t *testing.T) {
	tests := map[string]struct {
		o        interface{}
		wantErr  string
		beforeFn func() FS
		assertFn func(*testing.T, interface{})
	}{
		"Should read a simple json file": {
			o: &SampleJson{},
			beforeFn: func() FS {
				j, _ := json.Marshal(&SampleJson{
					Name: "name",
				})
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "filename", j, 0666)
				return Create(memfs)
			},
			assertFn: func(t *testing.T, o interface{}) {
				j := o.(*SampleJson)
				assert.Equal(t, "name", j.Name)
			},
		},
		"Should fail if file does not exist": {
			o:       &SampleJson{},
			wantErr: os.ErrNotExist.Error(),
			beforeFn: func() FS {
				memfs := memfs.New()
				return Create(memfs)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fs := tt.beforeFn()
			if err := fs.ReadJson("filename", tt.o); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("ReadYamls() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, tt.o)
			}
		})
	}
}

func Test_fsimpl_WriteJson(t *testing.T) {
	tests := map[string]struct {
		o        interface{}
		wantErr  string
		assertFn func(*testing.T, FS)
	}{
		"Should write a simple file": {
			o: &SampleJson{
				Name: "name",
			},
			assertFn: func(t *testing.T, fs FS) {
				data, err := fs.ReadFile("filename")
				assert.NoError(t, err)
				j := &SampleJson{}
				err = yaml.Unmarshal(data, j)
				assert.NoError(t, err)
				assert.Equal(t, "name", j.Name)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fs := Create(memfs.New())
			if err := fs.WriteJson("filename", tt.o); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("WriteYamls() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, fs)
			}
		})
	}
}
