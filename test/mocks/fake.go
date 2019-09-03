package mocks

import (
	"testing"

	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

// resource details employed in mocks
const (
	CRDName            = "jenkins.jenkins.dev"
	CRDVersion         = "apiextensions.k8s.io/v1beta1"
	CRDKind            = "Jenkins"
	OperatorKind       = "Jenkins"
	OperatorAPIVersion = "apiextensions.k8s.io/v1beta1"
)

type Fake struct {
	t    *testing.T       // testing instance
	ns   string           // namespace
	S    *runtime.Scheme  // runtime client scheme
	objs []runtime.Object // all fake objects
}

// AddMockedJenkins add mocked object from  JenkinsMock
func (f *Fake) AddMockedJenkins(name, ref string, matchLabels map[string]string) *jenkinsv1alpha1.Jenkins {
	f.S.AddKnownTypes(jenkinsv1alpha1.SchemeGroupVersion, &jenkinsv1alpha1.Jenkins{})
	jenkins := jenkinsCRMock(f.ns, name, ref, matchLabels)
	f.objs = append(f.objs, jenkins)
	return jenkins
}

// NewFake instantiate Fake type.
func NewFake(t *testing.T, ns string) *Fake {
	return &Fake{t: t, ns: ns, S: scheme.Scheme}
}

// FakeDynClient returns fake dynamic api client.
func (f *Fake) FakeDynClient() fakedynamic.FakeDynamicClient {
	fakeDynClient := fakedynamic.NewSimpleDynamicClient(f.S, f.objs...)
	return *fakeDynClient
}
