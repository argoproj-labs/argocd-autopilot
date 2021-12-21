package git

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_adoGit_CreateRepository(t *testing.T) {
	type fields struct {
		adoClient AdoClient
	}
	type args struct {
		ctx  context.Context
		opts *CreateRepoOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &adoGit{
				adoClient: tt.fields.adoClient,
			}
			got, err := g.CreateRepository(tt.args.ctx, tt.args.opts)
			if !tt.wantErr(t, err, fmt.Sprintf("CreateRepository(%v, %v)", tt.args.ctx, tt.args.opts)) {
				return
			}
			assert.Equalf(t, tt.want, got, "CreateRepository(%v, %v)", tt.args.ctx, tt.args.opts)
		})
	}
}

func Test_newAdo(t *testing.T) {
	type args struct {
		opts *ProviderOptions
	}
	tests := []struct {
		name    string
		args    args
		want    Provider
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newAdo(tt.args.opts)
			if !tt.wantErr(t, err, fmt.Sprintf("newAdo(%v)", tt.args.opts)) {
				return
			}
			assert.Equalf(t, tt.want, got, "newAdo(%v)", tt.args.opts)
		})
	}
}
