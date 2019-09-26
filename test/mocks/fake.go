package mocks

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
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
	t  *testing.T      // testing instance
	ns string          // namespace
	S  *runtime.Scheme // runtime client scheme
}

// AddMockedJenkins add mocked object from  JenkinsMock
// func (f *Fake) AddMockedJenkins(name, ref string, matchLabels map[string]string) *jenkinsv1alpha1.Jenkins {
// 	f.S.AddKnownTypes(jenkinsv1alpha1.SchemeGroupVersion, &jenkinsv1alpha1.Jenkins{})
// 	jenkins := JenkinsCRMock(f.ns, name, ref, matchLabels)
// }

// Add fakes for resource checking

// NewFake instantiate Fake type.
func NewFake(t *testing.T, ns string) *Fake {
	return &Fake{t: t, ns: ns, S: scheme.Scheme}
}

// TODO : Move to static client
