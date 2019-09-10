package jenkins

import (
	"testing"
	"time"

	"github.com/redhat-developer/openshift-jenkins-operator/test/mocks"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	test_ns   = "test"
	test_name = "test-jenkins"
)

func TestNewJenkinsService(t *testing.T) {
	t.Run("TestNewJenkinsService", func(t *testing.T) {
		jenkinsPort := corev1.ServicePort{
			Name:     JenkinsWebPortName,
			Port:     JenkinsWebPort,
			Protocol: JenkinsWebPortProtocol,
			TargetPort: intstr.IntOrString{
				IntVal: JenkinsWebPortAsInt,
				StrVal: JenkinsWebPortAsStr,
			},
		}

		checkSvc := corev1.Service{
			TypeMeta: metav1.TypeMeta{Kind: "", APIVersion: ""},
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

		// Testing the things that are bound to match.
		svc := newJenkinsService(mocks.JenkinsCRMock(test_ns, test_name), "test", jenkinsPort)
		require.Equal(t, svc.Spec, checkSvc.Spec)
		require.Equal(t, svc.ObjectMeta.Labels, checkSvc.ObjectMeta.Labels)
		require.Equal(t, svc.ObjectMeta.Name, checkSvc.ObjectMeta.Name)
		require.Equal(t, svc.ObjectMeta.Namespace, checkSvc.ObjectMeta.Namespace)
		require.Equal(t, svc.Status, checkSvc.Status)

	})
}

func TestNewJenkinsPvc(t *testing.T) {
	t.Run("TestNewJenkinsPvc", func(t *testing.T) {

		checkPvc := corev1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{Kind: "", APIVersion: ""},
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

		pvc := newJenkinsPvc(mocks.JenkinsCRMock(test_ns, test_name), "test")

		// Testing the things that are bound to match.
		// TODO : Add Spec checking
		require.Equal(t, pvc.ObjectMeta.Name, checkPvc.ObjectMeta.Name)
		require.Equal(t, pvc.ObjectMeta.Namespace, checkPvc.ObjectMeta.Namespace)
		require.Equal(t, pvc.Status, checkPvc.Status)

	})
}

func TestNewJenkinsDeploymentConfig(t *testing.T) {
	t.Run("TestNewJenkinsDc", func(t *testing.T) {
		checkDc := corev1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{Kind: "", APIVersion: ""},
			ObjectMeta: metav1.ObjectMeta{
				Name:                       test_name,
				GenerateName:               "",
				Namespace:                  test_ns,
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

		dc := newJenkinsDeploymentConfig(mocks.JenkinsCRMock(test_ns, test_name))

		// Testing the things that are bound to match.
		// TODO : Add Spec and Status checking
		require.Equal(t, dc.ObjectMeta.Name, checkDc.ObjectMeta.Name)

	})
}
