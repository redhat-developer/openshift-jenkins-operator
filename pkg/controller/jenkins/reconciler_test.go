package jenkins

import (
	"fmt"
	"testing"
	"time"

	"reflect"

	"github.com/redhat-developer/openshift-jenkins-operator/test/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	test_ns   = "test"
	test_name = "test-jenkins"
)

func TestReconcilerCreateResourceIfNotPresent(t *testing.T) {
	t.FailNow()
}

func TesNewDeploymentConfig(t *testing.T) {

}

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

		svc := newJenkinsService(mocks.JenkinsCRMock(test_ns, test_name), "test", jenkinsPort)
		fmt.Printf("%#v \n", svc)

		if !reflect.DeepEqual(svc.Spec, checkSvc.Spec) {
			t.FailNow()
		}
	})
}

func TestNewJenkinsPvc(t *testing.T) {
	t.FailNow()
}
