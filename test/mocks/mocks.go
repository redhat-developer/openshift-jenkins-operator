package mocks

import (
	"time"

	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Return a mock of a Jenkins Resource
func JenkinsCRMock(ns, name string) *jenkinsv1alpha1.Jenkins {
	return &jenkinsv1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: jenkinsv1alpha1.JenkinsSpec{},
	}
}

// Return a mock of a Jenkins Deployment Config
func JenkinsDCMock(ns, name string) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       name,
			GenerateName:               "",
			Namespace:                  ns,
			SelfLink:                   "",
			UID:                        "",
			ResourceVersion:            "",
			Generation:                 0,
			CreationTimestamp:          metav1.Time{Time: time.Time{}},
			DeletionTimestamp:          (*metav1.Time)(nil),
			DeletionGracePeriodSeconds: (*int64)(nil),
			Labels:                     map[string]string(nil),
			Annotations:                map[string]string(nil),
			OwnerReferences:            []metav1.OwnerReference(nil),
			Initializers:               (*metav1.Initializers)(nil),
			Finalizers:                 []string(nil), ClusterName: "",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Selector:    (*metav1.LabelSelector)(nil),
			Resources: corev1.ResourceRequirements{
				Limits:   corev1.ResourceList(nil),
				Requests: corev1.ResourceList{}},
			VolumeName: "", StorageClassName: (*string)(nil),
			VolumeMode: (*corev1.PersistentVolumeMode)(nil),
			DataSource: (*corev1.TypedLocalObjectReference)(nil),
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase:       "",
			AccessModes: []corev1.PersistentVolumeAccessMode(nil),
			Capacity:    corev1.ResourceList(nil),
			Conditions:  []corev1.PersistentVolumeClaimCondition(nil)},
	}
}

// Return a mock of a Jenkins Persistent Volume Claim
func JenkinsPvcMock(ns, name string) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "test",
			GenerateName:               "",
			Namespace:                  "test",
			SelfLink:                   "",
			UID:                        "",
			ResourceVersion:            "",
			Generation:                 0,
			CreationTimestamp:          metav1.Time{Time: time.Time{}},
			DeletionTimestamp:          (*metav1.Time)(nil),
			DeletionGracePeriodSeconds: (*int64)(nil),
			Labels:                     map[string]string(nil),
			Annotations:                map[string]string(nil),
			OwnerReferences:            []metav1.OwnerReference(nil),
			Initializers:               (*metav1.Initializers)(nil), Finalizers: []string(nil), ClusterName: ""},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Selector:    (*metav1.LabelSelector)(nil),
			Resources: corev1.ResourceRequirements{
				Limits:   corev1.ResourceList(nil),
				Requests: corev1.ResourceList{"storage": resource.Quantity{}},
			},
			VolumeName:       "",
			StorageClassName: (*string)(nil),
			VolumeMode:       (*corev1.PersistentVolumeMode)(nil),
			DataSource:       (*corev1.TypedLocalObjectReference)(nil),
		},
		Status: corev1.PersistentVolumeClaimStatus{Phase: "", AccessModes: []v1.PersistentVolumeAccessMode(nil), Capacity: v1.ResourceList(nil), Conditions: []v1.PersistentVolumeClaimCondition(nil)},
	}

}

// Return a mock of a Service used by Jenkins
func JenkinsServiceMock(ns, name string) corev1.Service {

	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "test",
			GenerateName:               "",
			Namespace:                  "test",
			SelfLink:                   "",
			UID:                        "",
			ResourceVersion:            "",
			Generation:                 0,
			CreationTimestamp:          metav1.Time{Time: time.Time{}},
			DeletionTimestamp:          (*metav1.Time)(nil),
			DeletionGracePeriodSeconds: (*int64)(nil),
			Labels:                     map[string]string{"app": "test-jenkins"},
			Annotations:                map[string]string(nil),
			OwnerReferences:            []metav1.OwnerReference(nil),
			Initializers:               (*metav1.Initializers)(nil),
			Finalizers:                 []string(nil), ClusterName: ""},
		Spec: corev1.ServiceSpec{
			Ports:                    []corev1.ServicePort{corev1.ServicePort{Name: "web", Protocol: "TCP", Port: 80, TargetPort: intstr.IntOrString{Type: 0, IntVal: 8080, StrVal: "8080"}, NodePort: 0}},
			Selector:                 map[string]string{"app": "test-jenkins"},
			ClusterIP:                "",
			Type:                     "ClusterIP",
			ExternalIPs:              []string(nil),
			SessionAffinity:          "",
			LoadBalancerIP:           "",
			LoadBalancerSourceRanges: []string(nil),
			ExternalName:             "",
			ExternalTrafficPolicy:    "",
			HealthCheckNodePort:      0,
			PublishNotReadyAddresses: false,
			SessionAffinityConfig:    (*corev1.SessionAffinityConfig)(nil),
		},
		Status: corev1.ServiceStatus{LoadBalancer: corev1.LoadBalancerStatus{Ingress: []corev1.LoadBalancerIngress(nil)}},
	}
}
