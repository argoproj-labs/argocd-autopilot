package gitops

import (
	"context"

	v1 "github.com/argoproj/argocd-autopilot/pkg/apis/gitops/v1"
	"github.com/codefresh-io/pkg/log"
)

type Server struct {
	v1.UnimplementedGitopsServer

	log log.Logger
}

func (s *Server) Add(context.Context, *v1.AddManifestRequest) (*v1.AddManifestResponse, error) {
	s.log.Debug("Add called")
	return &v1.AddManifestResponse{
		Path: "/foo/bar",
	}, nil
}
