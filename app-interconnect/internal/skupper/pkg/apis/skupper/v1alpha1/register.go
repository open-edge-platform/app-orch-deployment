package v1alpha1

import (
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/pkg/apis/skupper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   skupper.GroupName,
	Version: "v1alpha1",
}

func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &SkupperClusterPolicy{}, &SkupperClusterPolicyList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
