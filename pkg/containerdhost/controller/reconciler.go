package controller

import (
	"context"
	"io/fs"
	"os"
	"path"

	"github.com/octohelm/kubekit/pkg/operator"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler struct {
	ConfigPath string `flags:",omitempty"`
	mgr        controllerruntime.Manager
}

func (r *Reconciler) Run(ctx context.Context) error {
	return operator.ReconcilerRegistryContext.From(ctx).RegisterReconciler(r)
}

func (r *Reconciler) AddToScheme(s *runtime.Scheme) error {
	return corev1.AddToScheme(s)
}

func (r *Reconciler) SetupWithManager(mgr controllerruntime.Manager) error {
	r.mgr = mgr

	c := controllerruntime.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(
			predicate.NewPredicateFuncs(func(object client.Object) bool {
				return IsContainerdHostConfig(object)
			}),
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
				},
			}),
		)

	return c.Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	list := &corev1.ConfigMapList{}

	if err := r.mgr.GetClient().List(ctx, list, client.InNamespace(request.Namespace), &FilterHostConfig{}); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, r.sync(ctx, list.Items)
}

func (r *Reconciler) sync(ctx context.Context, items []corev1.ConfigMap) error {
	if len(items) == 0 {
		return nil
	}

	l := r.mgr.GetLogger()

	currentHosts := map[string]bool{}

	_, err := os.Stat(r.ConfigPath)
	if err == nil {
		if err := fs.WalkDir(os.DirFS(r.ConfigPath), ".", func(path string, d fs.DirEntry, err error) error {
			if path == "" || path == "." {
				return nil
			}
			currentHosts[path] = true
			return fs.SkipDir
		}); err != nil {
			return err
		}
	}

	for _, c := range items {
		host, err := r.syncByConfigMap(ctx, c)
		if err != nil {
			return err
		}
		l.WithValues("containerd.host", host).Info("Synced")
		delete(currentHosts, host)
	}

	for host := range currentHosts {
		_ = os.RemoveAll(path.Join(r.ConfigPath, host))
	}

	return nil
}

func (r *Reconciler) syncByConfigMap(ctx context.Context, cm corev1.ConfigMap) (string, error) {
	if len(cm.Data) == 0 {
		return "", errors.Errorf("missing data at %s.%s", cm.Name, cm.Namespace)
	}

	host := cm.Data["host"]
	if host == "" {
		return "", errors.Errorf("missing host value at %s.%s", cm.Name, cm.Namespace)
	}

	// https://github.com/containerd/containerd/blob/main/docs/hosts.md

	baseDir := path.Join(r.ConfigPath, host)

	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return "", err
	}

	for filename, data := range cm.Data {
		if filename == "host" {
			continue
		}

		if err := put(path.Join(baseDir, filename), data); err != nil {
			return "", err
		}
	}

	return host, nil
}

func put(filename string, data string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(data)
	return err
}
