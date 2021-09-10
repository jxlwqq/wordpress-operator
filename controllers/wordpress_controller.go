/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	appv1alpha1 "github.com/jxlwqq/wordpress-operator/api/v1alpha1"
)

// WordpressReconciler reconciles a Wordpress object
type WordpressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=wordpresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=wordpresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.jxlwqq.github.io,resources=wordpresses/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Wordpress object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *WordpressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := ctrllog.FromContext(ctx)
	reqLogger.Info("---Reconciling Wordpress---")

	wordpress := &appv1alpha1.Wordpress{}
	err := r.Client.Get(ctx, req.NamespacedName, wordpress)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var result *reconcile.Result

	// MySQL
	reqLogger.Info("---MySQL Secret---")
	result, err = r.ensureSecret(r.secretForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---MySQL PVC---")
	result, err = r.ensurePVC(r.pvcForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---MySQL Deployment---")
	result, err = r.ensureDeployment(r.deploymentForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---MySQL Service---")
	result, err = r.ensureService(r.serviceForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---MySQL Check Status---")
	if !r.isMysqlUp(wordpress) {
		delay := time.Second * time.Duration(5)
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	// WordPress
	reqLogger.Info("---WordPress PVC---")
	result, err = r.ensurePVC(r.pvcForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---WordPress Deployment---")
	result, err = r.ensureDeployment(r.deploymentForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---WordPress Service---")
	result, err = r.ensureService(r.serviceForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	reqLogger.Info("---WordPress Handle Changes---")
	result, err = r.handleWordpressChanges(wordpress)
	if result != nil {
		return *result, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WordpressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.Wordpress{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
