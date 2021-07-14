package git

import (
	"context"
	"testing"

	gt "code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
)

func Test_gitea_CreateRepository(t *testing.T) {
	type fields struct {
		opts   *ProviderOptions
		client *gt.Client
	}
	tests := map[string]struct {
		fields  fields
		opts    *CreateRepoOptions
		want    string
		wantErr string
	}{
		// TODO: Add test cases.
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			g := &gitea{
				opts:   tt.fields.opts,
				client: tt.fields.client,
			}
			got, err := g.CreateRepository(context.Background(), tt.opts)

			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("gitea.CreateRepository() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if got != tt.want {
				t.Errorf("gitea.CreateRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}
