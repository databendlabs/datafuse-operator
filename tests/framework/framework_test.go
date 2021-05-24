package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateTestID(t *testing.T) {
	tests := []struct {
		name string
		time int64
		want string
	}{
		{
			name: "basic",
			time: 1212131312312123,
			want: "generatetestid-basic-bxnxfr1pob",
		},
		{
			name: "basic/abcd",
			time: 1212131312312123,
			want: "generatetestid-basic-abcd-bxnxfr1pob",
		},
		{
			name: "",
			time: 1212131312312123,
			want: "generatetestid-#00-bxnxfr1pob",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generated := GenerateTestID(tt.time, t)
			assert.Equal(t, tt.want, generated)
		})
	}
}
