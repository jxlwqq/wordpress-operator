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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const mysqlSecretName = "mysql-pass"
const mysqlVolumeName = "mysql-persistent-storage"
const mysqlClaimName = "mysql-pv-claim"
const mysqlServiceName = "mysql-svc"
const mysqlDeploymentName = "mysql"

func (r *WordpressReconciler) secretForMysql(w *appv1alpha1.Wordpress) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: w.Namespace,
			Name:      mysqlSecretName,
		},
		StringData: map[string]string{
			"password": "xyz",
		},
		Type: corev1.SecretTypeOpaque,
	}

	_ = controllerutil.SetControllerReference(w, secret, r.Scheme)

	return secret
}

func (r *WordpressReconciler) pvcForMysql(w *appv1alpha1.Wordpress) *corev1.PersistentVolumeClaim {
	labels := labels("database")

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: w.Namespace,
			Name:      mysqlClaimName,
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

func (r *WordpressReconciler) deploymentForMysql(w *appv1alpha1.Wordpress) *appsv1.Deployment {

	password := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlSecretName},
			Key:                  "password",
		},
	}

	labels := labels("database")

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: w.Namespace,
			Name:      mysqlDeploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "mysql",
						Image:           "mysql:5.7",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:      "MYSQL_ROOT_PASSWORD",
								ValueFrom: password,
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 3306,
							Name:          "mysql",
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      mysqlVolumeName,
							MountPath: "/var/lib/mysql",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: mysqlVolumeName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: mysqlClaimName,
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

func (r *WordpressReconciler) serviceForMysql(w *appv1alpha1.Wordpress) *corev1.Service {
	labels := labels("database")

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: w.Namespace,
			Name:      mysqlServiceName,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Name:       "mysql",
				Port:       3306,
				TargetPort: intstr.FromInt(3306),
			}},
			ClusterIP: corev1.ClusterIPNone,
		},
	}

	_ = controllerutil.SetControllerReference(w, svc, r.Scheme)

	return svc
}

func (r *WordpressReconciler) isMysqlUp(w *appv1alpha1.Wordpress) bool {
	dep := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: w.Namespace,
		Name:      mysqlDeploymentName,
	}, dep)

	if err != nil {
		return false
	}

	if dep.Status.ReadyReplicas == 1 {
		return true
	}

	return false
}
