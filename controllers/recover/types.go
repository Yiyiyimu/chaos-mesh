// Copyright 2019 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package recover

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/chaos-mesh/chaos-mesh/controllers/common"
	"github.com/chaos-mesh/chaos-mesh/controllers/networkchaos/podnetworkmanager"
	"github.com/chaos-mesh/chaos-mesh/controllers/reconciler"
	"github.com/chaos-mesh/chaos-mesh/pkg/utils"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler for common chaos
type Reconciler struct {
	reconciler.InnerReconciler
	client.Client
	client.Reader
	Log logr.Logger
}

type Recoverer interface {
	FinalizersFunc() []string
	AnnotationsFunc() map[string]string
	NameFunc() string
	NamespaceFunc() string
}

type recover struct {
	Finalizers  []string
	Annotations map[string]string
	Name        string
	Namespace   string
}

func (rc *recover) FinalizersFunc() []string {
	return rc.Finalizers
}

func (rc *recover) AnnotationsFunc() map[string]string {
	return rc.Annotations
}

func (rc *recover) NameFunc() string {
	return rc.Name
}

func (rc *recover) NamespaceFunc() string {
	return rc.Namespace
}

func typeof(v interface{}, chaos v1alpha1.InnerObject) (*recover, string) {
	switch v.(type) {
	case *v1alpha1.NetworkChaos:
		somechaos, _ := chaos.(*v1alpha1.StressChaos)
		networkchaos := &recover{Finalizers: somechaos.Finalizers, Annotations: somechaos.Annotations}
		return networkchaos, "NetworkChaos"
	default:
		return nil, "unknown"
	}
}

// Recover means the reconciler recovers the chaos action
func (r *Reconciler) Recover(ctx context.Context, req ctrl.Request, chaos v1alpha1.InnerObject, v interface{}) error {
	stresschaos, ok := chaos.(*v1alpha1.StressChaos)
	somechaos, name := typeof(v, chaos)
	if !ok {
		err := errors.New("chaos is not StressChaos")
		r.Log.Error(err, "chaos is not StressChaos", "chaos", chaos)
		return err
	}

	if err := r.cleanFinalizersAndRecover(ctx, somechaos); err != nil {
		return err
	}
	r.Event(somechaos, v1.EventTypeNormal, utils.EventChaosRecovered, "")

	return nil
}

func (r *Reconciler) cleanFinalizersAndRecover(ctx context.Context, chaos *recover) error {
	var result error

	source := chaos.Namespace + "/" + chaos.Name
	m := podnetworkmanager.New(source, r.Log, r.Client, r.Reader)

	for _, key := range chaos.FinalizersFunc() {
		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

		_ = m.WithInit(types.NamespacedName{
			Namespace: ns,
			Name:      name,
		})

		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

		err = m.Commit(ctx)
		// if pod not found or not running, directly return and giveup recover.
		if err != nil && err != podnetworkmanager.ErrPodNotFound && err != podnetworkmanager.ErrPodNotRunning {
			r.Log.Error(err, "fail to commit")
		}

		finalizer := chaos.FinalizersFunc()
		finalizer = utils.RemoveFromFinalizer(chaos.FinalizersFunc(), key)
	}
	r.Log.Info("After recovering", "finalizers", chaos.FinalizersFunc())

	if chaos.AnnotationsFunc()[common.AnnotationCleanFinalizer] == common.AnnotationCleanFinalizerForced {
		r.Log.Info("Force cleanup all finalizers", "chaos", chaos)
		finalizer := chaos.FinalizersFunc()
		finalizer = chaos.FinalizersFunc()[:0]
		return nil
	}

	return result
}
