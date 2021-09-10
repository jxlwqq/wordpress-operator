package controllers

import (
	"context"
	appv1alpha1 "github.com/jxlwqq/wordpress-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

const wordpressClaimName = "wp-pv-claim"
const wordpressVolumeName = "wordpress-persistent-storage"
const wordpressServiceName = "wordpress-svc"
const wordpressServiceNodePort = 30690
const wordpressDeploymentName = "wordpress"
const wordpressImageName = "wordpress"

func (r *WordpressReconciler) pvcForWordpress(w *appv1alpha1.Wordpress) *corev1.PersistentVolumeClaim {
	labels := labels("frontend")
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: w.Namespace,
			Name:      wordpressClaimName,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	_ = controllerutil.SetControllerReference(w, pvc, r.Scheme)

	return pvc
}

func (r *WordpressReconciler) deploymentForWordpress(w *appv1alpha1.Wordpress) *appsv1.Deployment {
	labels := labels("frontend")

	password := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlSecretName},
			Key:                  "password",
		},
	}

	size := w.Spec.Size
	image := wordpressImageName + ":" + w.Spec.Version

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: w.Namespace,
			Name:      wordpressDeploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Name:            "wordpress",
						Env: []corev1.EnvVar{
							{
								Name:  "WORDPRESS_DB_HOST",
								Value: mysqlServiceName,
							},
							{
								Name:      "WORDPRESS_DB_PASSWORD",
								ValueFrom: password,
							},
						},
						Ports: []corev1.ContainerPort{{
							Name:          "wordpress",
							ContainerPort: 80,
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      wordpressVolumeName,
							MountPath: "/var/www/html",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: wordpressVolumeName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: wordpressClaimName,
							},
						},
					}},
				},
			},
		},
	}

	_ = controllerutil.SetControllerReference(w, dep, r.Scheme)

	return dep
}

func (r *WordpressReconciler) serviceForWordpress(w *appv1alpha1.Wordpress) *corev1.Service {
	labels := labels("frontend")
	svc := &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{
			Namespace: w.Namespace,
			Name:      wordpressServiceName,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.FromInt(80),
				Name:       "wordpress",
				NodePort:   wordpressServiceNodePort,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	_ = controllerutil.SetControllerReference(w, svc, r.Scheme)

	return svc
}

func (r *WordpressReconciler) handleWordpressChanges(w *appv1alpha1.Wordpress) (*ctrl.Result, error) {
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: w.Namespace,
		Name:      wordpressDeploymentName,
	}, found)
	if err != nil {
		return &ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	size := w.Spec.Size
	if size != *found.Spec.Replicas {
		found.Spec.Replicas = &size
		err = r.Client.Update(context.TODO(), found)
		if err != nil {
			return &ctrl.Result{}, err
		}
	}

	version := w.Spec.Version
	image := wordpressImageName + ":" + version
	existing := (*found).Spec.Template.Spec.Containers[0].Image
	if image != existing {
		(*found).Spec.Template.Spec.Containers[0].Image = image
		err = r.Client.Update(context.TODO(), found)
		if err != nil {
			return &ctrl.Result{}, err
		}
	}

	return nil, nil
}
