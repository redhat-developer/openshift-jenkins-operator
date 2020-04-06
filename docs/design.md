# Design

OpenShift Jenkins Operator follows operator sdk recommendations for its design.
We rely on simple and small Custom Resources Definitions (CRDs) that have strict separation 
of concerns.
As a guideline for our CRs: A minimum CR with just the Kind must trigger and create the minimal 
resources that it operates.

Design proposals are welcome if they allow simplification and separation of concerns.

## Jenkins Controller

The Jenkins Controller operates on the Jenkins crd. A Jenkins custom resource defines a Jenkins 
instance.
By default, it creates:
- a Deployment used to launch a Jenkins instance pod using the default OpenShift Jenkins 2 Image located
 19 in the ImageStream openshift/jenkins:2
- the required ServiceAccount with proper annotation for OpenShift Login Plugin to work
- RoleBinding, Service and Route (or Ingress TBD)  

## Jenkins Image Controller
The Jenkins Image Controller operates on the JenkinsImage crd. A Jenkins Image custom resource defines a 
custom build of a Jenkins Image using the s2i mechanism built in OpenShift Jenkins 2 image. 
By default, it creates:
- an ImageStream in the current project named after the cr.
- a BuildConfig of type Binary build using the plugin list defined in the Jenkins Image cr (empty if not defined).
- starts the build of this custom image (runs oc start-build)
- an ImageStreamTag pointing to latest on the previously created ImageStream.

## Jenkins Backup Controller
TBD
