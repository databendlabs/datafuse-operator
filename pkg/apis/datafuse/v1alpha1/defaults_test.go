package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetDatafuseComputeGroupDefaults(t *testing.T) {
	tests := []struct {
		name          string
		parent        *DatafuseOperator
		group         *DatafuseComputeGroup
		expectedGroup *DatafuseComputeGroup
	}{
		{
			name:          "nil operator",
			parent:        nil,
			group:         nil,
			expectedGroup: nil,
		},
		{
			name: "ok",
			parent: &DatafuseOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			group: &DatafuseComputeGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-group",
				},
			},
			expectedGroup: &DatafuseComputeGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-group",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDatafuseComputeGroupDefaults(tt.parent, tt.group)
			if tt.group == nil || tt.parent == nil {
				return
			}
			assert.Equal(t, tt.expectedGroup.Name, tt.group.Name)
			assert.Equal(t, tt.group.OwnerReferences, []metav1.OwnerReference{*metav1.NewControllerRef(tt.parent, SchemeGroupVersion.WithKind("DatafuseOperator"))})
		})
	}
}

func TestSetDatafuseOperatorDefault(t *testing.T) {
	tests := []struct {
		name             string
		parent           *DatafuseOperator
		expectedOperator *DatafuseOperator
	}{
		{
			name:             "nil operator",
			parent:           nil,
			expectedOperator: nil,
		},
		{
			name: "ok",
			parent: &DatafuseOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expectedOperator: &DatafuseOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: DatafuseOperatorSpec{
					ComputeGroups: []*DatafuseComputeGroupSpec{},
				},
				Status: DatafuseOperatorStatus{
					Status: OperatorCreated,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arg := tt.parent.DeepCopy()
			SetDatafuseOperatorDefault(arg)
			if tt.parent == nil {
				return
			}
			assert.Equal(t, tt.expectedOperator, arg)
		})
	}
}

func strPtr(input string) *string {
	return &input
}
