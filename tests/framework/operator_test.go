package framework

import (
	"testing"

	"datafuselabs.io/datafuse-operator/tests/utils/convert"
	"github.com/stretchr/testify/assert"
)

func TestMakeDatafuseOperator(t *testing.T) {
	op, err := MakeDatafuseOperatorFromYaml("../e2e/testfiles/default_operator.yaml")
	assert.NoError(t, err)
	assert.Equal(t, op.Name, "datafusecluster-default")
	assert.Equal(t, op.Namespace, "default")
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.Image, convert.StringToPtr("zhihanz/fuse-query:latest"))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.ImagePullPolicy, convert.StringToPtr("Always"))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.ClickhousePort, convert.Int32ToPtr(9000))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.MysqlPort, convert.Int32ToPtr(3306))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.RPCPort, convert.Int32ToPtr(9091))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.HTTPPort, convert.Int32ToPtr(8080))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.Cores, convert.Int32ToPtr(1))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.CoreLimit, convert.Int32ToPtr(2))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.Memory, convert.Int32ToPtr(512))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.MemoryLimit, convert.Int32ToPtr(1024))
	assert.Equal(t, op.Spec.ComputeGroups[0].ComputeLeaders.Replicas, convert.Int32ToPtr(1))
}

func TestReadUnKnownFile(t *testing.T) {
	_, err := MakeDatafuseOperatorFromYaml("./iqhdioqw")
	assert.Error(t, err)
}
