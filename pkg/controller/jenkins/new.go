package jenkins

import (
	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// newDeploymentConfigForCR returns a jenkins DeploymentConfig with the same name/namespace as the cr
func newJenkinsDeploymentConfig(cr *jenkinsv1alpha1.Jenkins, jenkinsService, jenkinsJNLPService string) *appsv1.DeploymentConfig {
	labels := map[string]string{
		JenkinsAppLabelName: cr.Name,
		JenkinsNameLabel:    JenkinsContainerName,
	}
	jenkinsInstanceName := cr.Name

	envVars := []corev1.EnvVar{
		corev1.EnvVar{Name: "OPENSHIFT_ENABLE_OAUTH", Value: "true"},
		corev1.EnvVar{Name: "OPENSHIFT_ENABLE_REDIRECT_PROMPT", Value: "true"},
		corev1.EnvVar{Name: "DISABLE_ADMINISTRATIVE_MONITORS", Value: "false"},
		corev1.EnvVar{Name: "KUBERNETES_MASTER", Value: "https://kubernetes.default:443"},
		corev1.EnvVar{Name: "KUBERNETES_TRUST_CERTIFICATES", Value: "true"},
		corev1.EnvVar{Name: "JENKINS_SERVICE_NAME", Value: jenkinsService},
		corev1.EnvVar{Name: "JNLP_SERVICE_NAME", Value: jenkinsJNLPService},
		corev1.EnvVar{Name: "ENABLE_FATAL_ERROR_LOG_FILE", Value: "false"},
		corev1.EnvVar{Name: "JENKINS_UC_INSECURE", Value: "false"},
	}

	livenessProbe := newProbe("/login", 8080, 420, 240, 360)
	readinessProbe := newProbe("/login", 8080, 3, 240, 0)

	dc := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jenkinsInstanceName,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: labels,
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.DeploymentStrategyTypeRecreate},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: JenkinsImage,
							Name:  JenkinsContainerName,
							VolumeMounts: []corev1.VolumeMount{
								{Name: JenkinsVolumeName, MountPath: JenkinsVolumeMountPath},
							},
							Env:                    envVars,
							LivenessProbe:          &livenessProbe,
							ReadinessProbe:         &readinessProbe,
							TerminationMessagePath: "/dev/termination-log",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: JenkinsVolumeName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: jenkinsInstanceName},
							},
						},
					},
					ServiceAccountName: jenkinsInstanceName,
				},
			},
		},
	}

	return dc
}

func newProbe(path string, port int, initialDelaySeconds, timeoutSeconds, periodSeconds int32) corev1.Probe {
	probe := corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt(port),
			},
		},
		FailureThreshold:    2,
		InitialDelaySeconds: initialDelaySeconds,
		TimeoutSeconds:      timeoutSeconds,
	}

	if periodSeconds > 0 {
		probe.PeriodSeconds = periodSeconds
	}

	return probe
}

// newJenkinsServicefor templates a new Service for Jenkins
func newJenkinsService(cr *jenkinsv1alpha1.Jenkins, name string, port corev1.ServicePort) *corev1.Service {
	labels := map[string]string{
		JenkinsAppLabel: cr.Name,
	}
	ports := []corev1.ServicePort{port}
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: cr.Namespace,
		Labels:    labels,
	}, Spec: corev1.ServiceSpec{
		Ports:    ports,
		Selector: labels,
		Type:     corev1.ServiceTypeClusterIP},
	}
	return svc
}

func newJenkinsRoute(cr *jenkinsv1alpha1.Jenkins, svc *corev1.Service) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		Spec: routev1.RouteSpec{
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
				Termination:                   routev1.TLSTerminationEdge,
			},
			To: routev1.RouteTargetReference{
				Kind: svc.Kind,
				Name: svc.Name,
			},
		},
	}
}

func newJenkinsPvc(cr *jenkinsv1alpha1.Jenkins, name string) *corev1.PersistentVolumeClaim {
	accessModes := []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	var quantity = resource.MustParse(JenkinsPvcSize)
	resources := corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: quantity}}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources:   resources,
		},
	}

	return pvc
}

func newJenkinsServiceAccount(cr *jenkinsv1alpha1.Jenkins, name string) *corev1.ServiceAccount {
	labels := map[string]string{
		JenkinsAppLabel:  cr.Name,
		JenkinsNameLabel: JenkinsServiceName,
	}
	annotationKey := "serviceaccounts.openshift.io/oauth-redirectreference." + cr.Name
	annotationValue := "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"" + cr.Name + "\"}}"
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				annotationKey: annotationValue,
			},
		},
	}
}

func newJenkinsRoleBinding(cr *jenkinsv1alpha1.Jenkins, jenkinsServiceAccountName string) *rbacv1.RoleBinding {
	labels := map[string]string{
		JenkinsAppLabel:  cr.Name,
		JenkinsNameLabel: JenkinsServiceName,
	}
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "_edit",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "edit",
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: jenkinsServiceAccountName,
			},
		},
	}
}
