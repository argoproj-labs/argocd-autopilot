package server

import (
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	MaxGRPCMessageSize = (1 << 20) * 100 // 100MB
	StaticAssetsPath   = "./server/assets"
)

var defaultBackoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func RegisterHealthCheck(mux *http.ServeMux, handler func(*http.Request) error) {
	mux.HandleFunc("/healthz", func(rw http.ResponseWriter, r *http.Request) {
		if err := handler(r); err != nil {
			rw.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(rw, err)
		} else {
			fmt.Fprint(rw, "ok")
		}
	})
}
