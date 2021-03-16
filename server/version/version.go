package version

import (
	"context"

	"github.com/argoproj/argocd-autopilot/pkg/apis/version"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	version.UnimplementedVersionServer
}

func (s *Server) Version(context.Context, *emptypb.Empty) (*version.VersionResponse, error) {
	v := store.Get().Version
	return &version.VersionResponse{
		Version:    v.Version,
		BuildDate:  v.BuildDate,
		GitCommit:  v.GitCommit,
		GoVersion:  v.GoVersion,
		GoCompiler: v.GoCompiler,
	}, nil
}
