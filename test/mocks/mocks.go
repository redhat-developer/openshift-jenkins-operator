package mocks

import (
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func JenkinsCRMock(
	ns, name string) *jenkinsv1alpha1.Jenkins {
	return &jenkinsv1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: jenkinsv1alpha1.JenkinsSpec{},
	}
}
