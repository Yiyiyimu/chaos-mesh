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

package controllers

import (
	"github.com/go-logr/logr"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HelloWorldChaosReconciler reconciles a HelloWorldChaos object
type HelloWorldChaosReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=chaos-mesh.org,resources=helloworldchaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos-mesh.org,resources=helloworldchaos/status,verbs=get;update;patch

// Reconcile reconciles an HelloWorldChaos resource
func (r *HelloWorldChaosReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("reconciler", "helloworldchaos")

	// the main logic of `HelloWorldChaos`, it prints a log `Hello World!` and returns nothing.
	logger.Info("Hello World!")

	return ctrl.Result{}, nil
}

func (r *HelloWorldChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// exports `HelloWorldChaos` object, which represents the yaml schema content the user applies.
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.HelloWorldChaos{}).
		Complete(r)
}
