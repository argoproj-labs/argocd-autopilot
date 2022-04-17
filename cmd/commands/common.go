package commands

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/argoproj-labs/argocd-autopilot/pkg/argocd"
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	"github.com/argoproj-labs/argocd-autopilot/pkg/kube"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

  appset "github.com/argoproj/applicationset/api/v1alpha1"
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// used for mocking
var (
	die  = util.Die
	exit = os.Exit

	DefaultApplicationSetGeneratorInterval int64 = 20

	//go:embed assets/cluster_res_readme.md
	clusterResReadmeTpl []byte

	//go:embed assets/projects_readme.md
	projectReadme []byte

	//go:embed assets/apps_readme.md
	appsReadme []byte

	getRepo = func(ctx context.Context, cloneOpts *git.CloneOptions) (git.Repository, fs.FS, error) {
		return cloneOpts.GetRepo(ctx)
	}

	prepareRepo = func(ctx context.Context, cloneOpts *git.CloneOptions, projectName string) (git.Repository, fs.FS, error) {
		log.G(ctx).WithFields(log.Fields{
			"repoURL":  cloneOpts.URL(),
			"revision": cloneOpts.Revision(),
			"forWrite": cloneOpts.CloneForWrite,
		}).Debug("starting with options: ")

		// clone repo
		log.G(ctx).Infof("cloning git repository: %s", cloneOpts.URL())
		r, repofs, err := getRepo(ctx, cloneOpts)
		if err != nil {
			return nil, nil, fmt.Errorf("failed cloning the repository: %w", err)
		}

		root := repofs.Root()
		log.G(ctx).Infof("using revision: \"%s\", installation path: \"%s\"", cloneOpts.Revision(), root)
		if !repofs.ExistsOrDie(store.Default.BootsrtrapDir) {
			return nil, nil, fmt.Errorf("bootstrap directory not found, please execute `repo bootstrap` command")
		}

		if projectName != "" {
			projExists := repofs.ExistsOrDie(repofs.Join(store.Default.ProjectsDir, projectName+".yaml"))
			if !projExists {
				return nil, nil, fmt.Errorf(util.Doc(fmt.Sprintf("project '%[1]s' not found, please execute `<BIN> project create %[1]s`", projectName)))
			}
		}

		log.G(ctx).Debug("repository is ok")

		return r, repofs, nil
	}
)

type createAppOptions struct {
	name        string
	namespace   string
	repoURL     string
	revision    string
	srcPath     string
	destServer  string
	noFinalizer bool
	labels      map[string]string
}

func createApp(opts *createAppOptions) ([]byte, error) {
	if opts.destServer == "" {
		opts.destServer = store.Default.DestServer
	}

	app := &argocdv1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       argocdv1alpha1.ApplicationSchemaGroupVersionKind.Kind,
			APIVersion: argocdv1alpha1.ApplicationSchemaGroupVersionKind.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.namespace,
			Name:      opts.name,
			Labels: map[string]string{
				store.Default.LabelKeyAppManagedBy: store.Default.LabelValueManagedBy,
				"app.kubernetes.io/name":           opts.name,
			},
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: argocdv1alpha1.ApplicationSpec{
			Project: "default",
			Source: argocdv1alpha1.ApplicationSource{
				RepoURL:        opts.repoURL,
				Path:           opts.srcPath,
				TargetRevision: opts.revision,
			},
			Destination: argocdv1alpha1.ApplicationDestination{
				Server:    opts.destServer,
				Namespace: opts.namespace,
			},
			SyncPolicy: &argocdv1alpha1.SyncPolicy{
				Automated: &argocdv1alpha1.SyncPolicyAutomated{
					SelfHeal:   true,
					Prune:      true,
					AllowEmpty: true,
				},
				SyncOptions: []string{
					"allowEmpty=true",
				},
			},
			IgnoreDifferences: []argocdv1alpha1.ResourceIgnoreDifferences{
				{
					Group: "argoproj.io",
					Kind:  "Application",
					JSONPointers: []string{
						"/status",
					},
				},
			},
		},
	}
	if opts.noFinalizer {
		app.ObjectMeta.Finalizers = []string{}
	}
	if len(opts.labels) > 0 {
		for k, v := range opts.labels {
			app.ObjectMeta.Labels[k] = v
		}
	}

	return yaml.Marshal(app)
}

func waitAppSynced(ctx context.Context, f kube.Factory, timeout time.Duration, appName, namespace, revision string, waitForCreation bool) error {
	return f.Wait(ctx, &kube.WaitOptions{
		Interval: store.Default.WaitInterval,
		Timeout:  timeout,
		Resources: []kube.Resource{
			{
				Name:      appName,
				Namespace: namespace,
				WaitFunc:  argocd.GetAppSyncWaitFunc(revision, waitForCreation),
			},
		},
	})
}

type createAppSetOptions struct {
	name                        string
	namespace                   string
	appName                     string
	appNamespace                string
	appProject                  string
	repoURL                     string
	revision                    string
	srcPath                     string
	destServer                  string
	destNamespace               string
	prune                       bool
	preserveResourcesOnDeletion bool
	appLabels                   map[string]string
	generators                  []appset.ApplicationSetGenerator
}

func createAppSet(o *createAppSetOptions) ([]byte, error) {
	if o.destServer == "" {
		o.destServer = store.Default.DestServer
	}

	if o.appProject == "" {
		o.appProject = "default"
	}

	if o.appLabels == nil {
		// default labels
		o.appLabels = map[string]string{
			store.Default.LabelKeyAppManagedBy: store.Default.LabelValueManagedBy,
			"app.kubernetes.io/name":           o.appName,
		}
	}

	appSet := &appset.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet",
			APIVersion: appset.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.name,
			Namespace: o.namespace,
			Annotations: map[string]string{
				"argocd.argoproj.io/sync-wave": "0",
			},
		},
		Spec: appset.ApplicationSetSpec{
			Generators: o.generators,
			Template: appset.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: appset.ApplicationSetTemplateMeta{
					Namespace: o.appNamespace,
					Name:      o.appName,
					Labels:    o.appLabels,
				},
				Spec: argocdv1alpha1.ApplicationSpec{
					Project: o.appProject,
					Source: argocdv1alpha1.ApplicationSource{
						RepoURL:        o.repoURL,
						Path:           o.srcPath,
						TargetRevision: o.revision,
					},
					Destination: argocdv1alpha1.ApplicationDestination{
						Server:    o.destServer,
						Namespace: o.destNamespace,
					},
					SyncPolicy: &argocdv1alpha1.SyncPolicy{
						Automated: &argocdv1alpha1.SyncPolicyAutomated{
							SelfHeal:   true,
							Prune:      o.prune,
							AllowEmpty: true,
						},
					},
					IgnoreDifferences: []argocdv1alpha1.ResourceIgnoreDifferences{
						{
							Group: "argoproj.io",
							Kind:  "Application",
							JSONPointers: []string{
								"/status",
							},
						},
					},
				},
			},
			SyncPolicy: &appset.ApplicationSetSyncPolicy{
				PreserveResourcesOnDeletion: o.preserveResourcesOnDeletion,
			},
		},
	}

	return yaml.Marshal(appSet)
}

var getInstallationNamespace = func(repofs fs.FS) (string, error) {
	path := repofs.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName+".yaml")
	a := &argocdv1alpha1.Application{}
	if err := repofs.ReadYamls(path, a); err != nil {
		return "", fmt.Errorf("failed to unmarshal namespace: %w", err)
	}

	return a.Spec.Destination.Namespace, nil
}
