package server

import (
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
