package jenkinsimage

import (
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	jenkinsv1alpha1 "github.com/redhat-developer/openshift-jenkins-operator/pkg/apis/jenkins/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newImageStream returns an ImageStream in the current namespace with the same name as cr and a tag
// "latest" pointing to it.
func newImageStream(cr *jenkinsv1alpha1.JenkinsImage) *imagev1.ImageStream {
	// build image repository name
	imageName := DefaultRegistryHostname + ImageNameSeparator + cr.Namespace + ImageNameSeparator + cr.Name
	is := &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name: cr.Name,
			Namespace: cr.Namespace,
		},
		Spec: imagev1.ImageStreamSpec{
			Tags: []imagev1.TagReference{
				{
					Name: DefaultImageStreamTag,
					From: &corev1.ObjectReference{
						Kind: DockerImageKind,
						Name: imageName,
					},
				},
			},
		},
	}
	return is
}

// newBuildConfig returns a BuildConfig with binary source strategy using the source image or
// imagestream specified in the CR
func newBuildConfig(cr *jenkinsv1alpha1.JenkinsImage) *buildv1.BuildConfig {
	bc := &buildv1.BuildConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		Spec: buildv1.BuildConfigSpec{
			RunPolicy: buildv1.BuildRunPolicySerial,
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Binary: &buildv1.BinaryBuildSource{},
				},
				Strategy: buildv1.BuildStrategy{
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: corev1.ObjectReference{
							Kind:      ImageStreamTagKind,
							Name:      DefaultJenkinsBaseImage,
							Namespace: DefaultImageNamespace,
						},
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: ImageStreamTagKind,
						Name: cr.Name + ImageToTagSeparator + DefaultImageStreamTag,
					},
				},
			},
		},
	}
	return bc
}
