package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	grpcgw "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/argoproj/argocd-autopilot/pkg/apis/gitops/v1"
	"github.com/codefresh-io/pkg/helpers"
	"github.com/codefresh-io/pkg/log"
)

type Server struct {
	Options

	log        log.Logger
	grpcServer *grpc.Server
	httpServer *http.Server
	stopChan   <-chan struct{}
}

type Options struct {
	Port                     int
	AccessControlAllowOrigin string
}

func NewOrDie(ctx context.Context, opts *Options) *Server {
	if opts == nil {
		panic(fmt.Errorf("nil options"))
	}

	s := &Server{
		Options:  *opts,
		log:      log.G(ctx),
		stopChan: ctx.Done(),
	}

	s.grpcServer = s.newGRPCServer()
	s.httpServer = s.newHTTPServer(ctx)

	return s
}

func (s *Server) Run() {
	var (
		lis       net.Listener
		listerErr error
	)

	// Start listener
	address := fmt.Sprintf(":%d", s.Port)

	err := wait.ExponentialBackoff(defaultBackoff, func() (bool, error) {
		lis, listerErr = net.Listen("tcp", address)
		if listerErr != nil {
			s.log.Warnf("failed to listen: %v", listerErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		s.log.Fatalf("failed to create listener: %s", err)
	}

	tcpMux := cmux.New(lis)

	httpL := tcpMux.Match(cmux.HTTP1Fast())
	grpcL := tcpMux.Match(cmux.Any())

	s.log.WithField("address", address).Info("handling grpc and http requests")

	go func() { s.checkServerErr(s.grpcServer.Serve(grpcL)) }()
	go func() { s.checkServerErr(s.httpServer.Serve(httpL)) }()
	go func() { s.checkServerErr(tcpMux.Serve()) }()

	<-s.stopChan
}

func (s *Server) newGRPCServer() *grpc.Server {
	uInterceptors := []grpc.UnaryServerInterceptor{
		grpc_prometheus.UnaryServerInterceptor,
		s.PanicLoggerUnaryServerInterceptor(),
	}
	sInterceptors := []grpc.StreamServerInterceptor{
		grpc_prometheus.StreamServerInterceptor,
		s.PanicLoggerStreamServerInterceptor(),
	}

	logrusE, err := log.GetLogrusEntry(s.log)
	if err != nil {
		s.log.Warn("not using logrus logger, no logging middleware")
	} else {
		uInterceptors = append(uInterceptors, grpc_logrus.UnaryServerInterceptor(logrusE))
		sInterceptors = append(sInterceptors, grpc_logrus.StreamServerInterceptor(logrusE))
	}

	sOpts := []grpc.ServerOption{
		grpc.ConnectionTimeout(300 * time.Second),
		grpc.MaxRecvMsgSize(MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(MaxGRPCMessageSize),
		grpc.ChainUnaryInterceptor(grpc_middleware.ChainUnaryServer(uInterceptors...)),
		grpc.ChainStreamInterceptor(grpc_middleware.ChainStreamServer(sInterceptors...)),
	}

	grpcS := grpc.NewServer(sOpts...)

	grpc_prometheus.Register(grpcS)
	reflection.Register(grpcS)

	return grpcS
}

func (s *Server) newHTTPServer(ctx context.Context) *http.Server {
	addr := fmt.Sprintf(":%d", s.Port)
	mux := http.NewServeMux()

	dialOps := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(MaxGRPCMessageSize),
			grpc.MaxCallSendMsgSize(MaxGRPCMessageSize),
		),
		grpc.WithUserAgent("grpc-gateway"),
	}

	gwmux := grpcgw.NewServeMux(
		grpcgw.WithMarshalerOption(grpcgw.MIMEWildcard, &grpcgw.JSONBuiltin{}),
	)

	helpers.Die(v1.RegisterGitopsHandlerFromEndpoint(ctx, gwmux, addr, dialOps))

	mux.Handle("/api/", gwmux)
	mux.Handle("/", http.FileServer(http.Dir(StaticAssetsPath)))
	mux.Handle("/swagger-ui", middleware.Redoc(middleware.RedocOpts{
		BasePath: "/",
		Path:     "/swagger-ui",
		SpecURL:  "/swagger.json",
	}, http.NotFoundHandler()))

	return &http.Server{Addr: addr, Handler: mux}
}

func (s *Server) checkServerErr(err error) {
	if err != nil {
		open := false
		select {
		case _, open = <-s.stopChan:
		default:
		}

		if open {
			s.log.Fatalf("server listen error: %s", err)
		}

		s.log.Infof("graceful shutdown: %v", err)
	}
}
