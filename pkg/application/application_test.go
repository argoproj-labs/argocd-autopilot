package application

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	kusttypes "sigs.k8s.io/kustomize/api/types"
)

func Test_parseApplication(t *testing.T) {
	orgGenerateManifests := generateManifests
	defer func() { generateManifests = orgGenerateManifests }()
	generateManifests = func(k *kusttypes.Kustomization) ([]byte, error) {
		return []byte("foo"), nil
	}

	tests := map[string]struct {
		opts     *CreateOptions
		assertFn func(*testing.T, *application, error)
	}{
		"No app specifier": {
			opts: &CreateOptions{
				AppName: "foo",
			},
			assertFn: func(t *testing.T, a *application, ret error) {
				assert.ErrorIs(t, ret, ErrEmptyAppSpecifier)
			},
		},
		"No app name": {
			opts: &CreateOptions{
				AppSpecifier: "foo",
			},
			assertFn: func(t *testing.T, a *application, ret error) {
				assert.ErrorIs(t, ret, ErrEmptyAppName)
			},
		},
		"Invalid installation mode": {
			opts: &CreateOptions{
				AppSpecifier:     "foo",
				AppName:          "foo",
				InstallationMode: "foo",
			},
			assertFn: func(t *testing.T, a *application, ret error) {
				assert.EqualError(t, ret, "unknown installation mode: foo")
			},
		},
		"Normal installation mode": {
			opts: &CreateOptions{
				AppSpecifier: "foo",
				AppName:      "foo",
			},
			assertFn: func(t *testing.T, a *application, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, "foo", a.Base().Resources[0])
				assert.Equal(t, "../../base", a.Overlay().Resources[0])
				assert.Nil(t, a.Namespace())
				assert.Nil(t, a.Manifests())
				assert.True(t, reflect.DeepEqual(&Config{
					AppName:       "foo",
					UserGivenName: "foo",
				}, a.Config()))
			},
		},
		"Flat installation mode with namespace": {
			opts: &CreateOptions{
				AppSpecifier:     "foo",
				AppName:          "foo",
				InstallationMode: InstallationModeFlat,
				DestNamespace:    "foo",
			},
			assertFn: func(t *testing.T, a *application, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, "install.yaml", a.Base().Resources[0])
				assert.Equal(t, []byte("foo"), a.Manifests())
				assert.Equal(t, "../../base", a.Overlay().Resources[0])
				assert.Equal(t, "namespace.yaml", a.Overlay().Resources[1])
				assert.Equal(t, "foo", a.Namespace().ObjectMeta.Name)
				assert.True(t, reflect.DeepEqual(&Config{
					AppName:       "foo",
					UserGivenName: "foo",
					DestNamespace: "foo",
				}, a.Config()))
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			app, err := parseApplication(tt.opts)
			tt.assertFn(t, app, err)
		})
	}
}
