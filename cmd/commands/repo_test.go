package commands

import (
	"context"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/stretchr/testify/assert"
)

func TestRunRepoCreate(t *testing.T) {
	tests := map[string]struct {
		opts     *RepoCreateOptions
		assertFn func(t *testing.T, opts *RepoCreateOptions, ret error)
	}{
		"Invalid provider": {
			opts: &RepoCreateOptions{
				Provider: "foobar",
			},
			assertFn: func(t *testing.T, _ *RepoCreateOptions, ret error) {
				assert.ErrorIs(t, ret, git.ErrProviderNotSupported)
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			tt.assertFn(t, tt.opts, RunRepoCreate(context.Background(), tt.opts))
		})
	}
}
