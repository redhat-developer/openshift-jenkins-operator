package jenkins

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/redhat-developer/openshift-jenkins-operator/test/mocks"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

		svc := newJenkinsService(mocks.JenkinsCRMock(test_ns, test_name), "test", jenkinsPort)
		mockSvc := mocks.JenkinsServiceMock(test_ns, test_name)
		b, err := json.MarshalIndent(svc, "", "  ")
		if err == nil {
			fmt.Println(string(b))
		}
		// Testing the things that are bound to match.
		require.Equal(t, svc.Spec, mockSvc.Spec)
		require.Equal(t, svc.ObjectMeta.Labels, mockSvc.ObjectMeta.Labels)
		require.Equal(t, svc.ObjectMeta.Name, mockSvc.ObjectMeta.Name)
		require.Equal(t, svc.ObjectMeta.Namespace, mockSvc.ObjectMeta.Namespace)
		require.Equal(t, svc.Status, mockSvc.Status)

	})
}

func TestNewJenkinsPvc(t *testing.T) {
	t.Run("TestNewJenkinsPvc", func(t *testing.T) {

		pvc := newJenkinsPvc(mocks.JenkinsCRMock(test_ns, test_name), "test")
		mockPvc := mocks.JenkinsPvcMock(test_ns, test_name)
		// Testing the things that are bound to match.
		// TODO : Add Spec checking
		require.Equal(t, pvc.ObjectMeta.Name, mockPvc.ObjectMeta.Name)
		require.Equal(t, pvc.ObjectMeta.Namespace, mockPvc.ObjectMeta.Namespace)
		require.Equal(t, pvc.Status, mockPvc.Status)

	})
}

func TestNewJenkinsDeploymentConfig(t *testing.T) {
	t.Run("TestNewJenkinsDc", func(t *testing.T) {
		dc := newJenkinsDeploymentConfig(mocks.JenkinsCRMock(test_ns, test_name), JenkinsServiceName, JenkinsJNLPServiceName, true)

		mockDc := mocks.JenkinsDCMock(test_ns, test_name)
		// Testing the things that are bound to match.
		// TODO : Add Spec and Status checking
		b, err := json.MarshalIndent(dc, "", "  ")
		if err == nil {
			fmt.Println(string(b))
		}
		require.Equal(t, dc.ObjectMeta.Name, mockDc.ObjectMeta.Name)

	})
}

