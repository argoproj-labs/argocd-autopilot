package commands

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"
	appsetv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
)

type (
	BaseOptions struct {
		CloneOptions *git.CloneOptions
		FS           fs.FS
		ProjectName  string
	}
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

	clone = func(ctx context.Context, cloneOpts *git.CloneOptions, filesystem fs.FS) (git.Repository, fs.FS, error) {
		return cloneOpts.Clone(ctx, filesystem)
	}

	prepareRepo = func(ctx context.Context, o *BaseOptions) (git.Repository, fs.FS, error) {
		var (
			r   git.Repository
			err error
		)
		log.G().WithFields(log.Fields{
			"repoURL":  o.CloneOptions.URL,
			"revision": o.CloneOptions.Revision,
		}).Debug("starting with options: ")

		// clone repo
		log.G().Infof("cloning git repository: %s", o.CloneOptions.URL)
		r, repofs, err := clone(ctx, o.CloneOptions, o.FS)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed cloning the repository: %w", err)
		}

		root := repofs.Root()
		log.G().Infof("using revision: \"%s\", installation path: \"%s\"", o.CloneOptions.Revision, root)
		if !repofs.ExistsOrDie(store.Default.BootsrtrapDir) {
			cmd := "repo bootstrap"
			if root != "/" {
				cmd += " --installation-path " + root
			}

			return nil, nil, fmt.Errorf("Bootstrap directory not found, please execute `%s` command", cmd)
		}

		if o.ProjectName != "" {
			projExists := repofs.ExistsOrDie(repofs.Join(store.Default.ProjectsDir, o.ProjectName+".yaml"))
			if !projExists {
				return nil, nil, fmt.Errorf(util.Doc(fmt.Sprintf("project '%[1]s' not found, please execute `<BIN> project create %[1]s`", o.ProjectName)))
			}
		}

		log.G().Debug("repository is ok")

		return r, repofs, nil
	}
)

func addFlags(cmd *cobra.Command) (*BaseOptions, error) {
	cloneOptions, err := git.AddFlags(cmd)
	if err != nil {
		return nil, err
	}

	o := &BaseOptions{
		CloneOptions: cloneOptions,
		FS:           fs.Create(memfs.New()),
	}

	return o, nil
}

type createAppOptions struct {
	name        string
	namespace   string
	repoURL     string
	revision    string
	srcPath     string
	destServer  string
	noFinalizer bool
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
				"app.kubernetes.io/managed-by": store.Default.ManagedBy,
				"app.kubernetes.io/name":       opts.name,
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
					SelfHeal: true,
					Prune:    true,
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

	return yaml.Marshal(app)
}

type createAppSetOptions struct {
	name          string
	namespace     string
	appName       string
	appNamespace  string
	repoURL       string
	revision      string
	srcPath       string
	destServer    string
	destNamespace string
	noFinalizer   bool
	prune         bool
	appLabels     map[string]string
	generators    []appset.ApplicationSetGenerator
}

func createAppSet(o *createAppSetOptions) ([]byte, error) {
	if o.destServer == "" {
		o.destServer = store.Default.DestServer
	}

	appSet := &appset.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet",
			APIVersion: appset.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.name,
			Namespace: o.namespace,
		},
		Spec: appset.ApplicationSetSpec{
			Generators: o.generators,
			Template: appset.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: appset.ApplicationSetTemplateMeta{
					Namespace: o.appNamespace,
					Name:      o.appName,
					Labels:    o.appLabels,
				},
				Spec: appsetv1alpha1.ApplicationSpec{
					Source: appsetv1alpha1.ApplicationSource{
						RepoURL:        o.repoURL,
						Path:           o.srcPath,
						TargetRevision: o.revision,
					},
					Destination: appsetv1alpha1.ApplicationDestination{
						Server:    o.destServer,
						Namespace: o.destNamespace,
					},
					SyncPolicy: &appsetv1alpha1.SyncPolicy{
						Automated: &appsetv1alpha1.SyncPolicyAutomated{
							SelfHeal: true,
							Prune:    o.prune,
						},
					},
				},
			},
		},
	}

	if o.appLabels == nil {
		// default labels
		appSet.Spec.Template.ApplicationSetTemplateMeta.Labels = map[string]string{
			"app.kubernetes.io/managed-by": store.Default.ManagedBy,
			"app.kubernetes.io/name":       o.appName,
		}
	}

	if o.noFinalizer {
		appSet.ObjectMeta.Finalizers = []string{}
	}

	return yaml.Marshal(appSet)
}
