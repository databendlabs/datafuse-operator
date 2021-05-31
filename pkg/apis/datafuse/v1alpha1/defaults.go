package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetDatafuseOperatorDefault(operator *DatafuseOperator) {
	if operator == nil {
		return
	}
	if operator.GetNamespace() == "" {
		operator.ObjectMeta.SetNamespace("default")
	}
	if operator.GetName() == "" {
		operator.ObjectMeta.SetName("default-operator")
	}
	if operator.Spec.ComputeGroups == nil {
		operator.Spec.ComputeGroups = make([]*DatafuseComputeGroupSpec, 0)
	}
	if operator.Status.Status == "" {
		operator.Status.Status = OperatorCreated
	}
}

func SetDatafuseComputeGroupDefaults(parentOperator *DatafuseOperator, group *DatafuseComputeGroup) {
	if parentOperator == nil || group == nil {
		return
	}
	if group.OwnerReferences == nil || !metav1.IsControlledBy(group, parentOperator) {
		group.OwnerReferences = []metav1.OwnerReference{}
		group.OwnerReferences = append(group.OwnerReferences, *metav1.NewControllerRef(parentOperator, SchemeGroupVersion.WithKind("DatafuseOperator")))
	}
	if group.GetNamespace() == "" {
		group.SetNamespace("default")
	}
}
