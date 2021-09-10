package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func labels(component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "wordpress",
		"app.kubernetes.io/component": component,
	}
}

func (r *WordpressReconciler) ensureSecret(s *corev1.Secret) (*ctrl.Result, error) {
	found := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      s.Name,
		Namespace: s.Namespace,
	}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			err = r.Client.Create(context.TODO(), s)
			if err != nil {
				return &ctrl.Result{}, err
			}
		}
		return &ctrl.Result{}, err
	}

	return nil, nil
}

func (r *WordpressReconciler) ensurePVC(pvc *corev1.PersistentVolumeClaim) (*ctrl.Result, error) {
	found := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: pvc.Namespace,
		Name:      pvc.Name,
	}, found)

	if err != nil {
		if errors.IsNotFound(err) {
			err = r.Client.Create(context.TODO(), pvc)
			if err != nil {
				return &ctrl.Result{}, err
			}
		}
		return &ctrl.Result{}, err
	}

	return nil, nil
}

func (r *WordpressReconciler) ensureDeployment(dep *appsv1.Deployment) (*ctrl.Result, error) {
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: dep.Namespace,
		Name:      dep.Name,
	}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			err = r.Create(context.TODO(), dep)
			return &ctrl.Result{}, err
		}

		return &ctrl.Result{}, err
	}
	return nil, nil
}

func (r *WordpressReconciler) ensureService(svc *corev1.Service) (*ctrl.Result, error) {
	found := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: svc.Namespace,
		Name:      svc.Name,
	}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			err = r.Create(context.TODO(), svc)
			if err != nil {
				return &ctrl.Result{}, err
			}
		}
		return &ctrl.Result{}, err
	}

	return nil, nil
}
