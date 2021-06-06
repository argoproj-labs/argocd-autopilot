package application

import (
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"

	billyUtils "github.com/go-git/go-billy/v5/util"
)

func serverToClusterName(repofs fs.FS, server string) (string, error) {
	confs, err := billyUtils.Glob(repofs, repofs.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, "*.json"))
	if err != nil {
		return "", err
	}

	for _, confFile := range confs {
		conf := &ClusterResConfig{}
		if err = repofs.ReadYamls(confFile, conf); err != nil {
			return "", err
		}
		if conf.Server == server {
			return conf.Name, nil
		}
	}

	return "", nil
}
