package minio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBucketName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		parentName   string
		parentType   string
		expectedName string
	}{
		{
			name:         "bucket name is less than 63 chars",
			parentName:   "testName",
			parentType:   "test",
			expectedName: "test-testName",
		},
		{
			name:         "bucket name is 63 chars",
			parentName:   "O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox00EwMTeJVs1V",
			parentType:   "test",
			expectedName: "test-O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox00EwMTeJVs1V",
		},
		{
			name: "bucket name is over 63 chars",
			parentName: "O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox00EwMTeJVs1V" +
				"O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox00EwMTeJVs1V",
			parentType:   "test",
			expectedName: "test-O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox0-3877779712",
		},
	}
	var c Client
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actualName := c.GetValidBucketName(tt.parentType, tt.parentName)
			assert.Equal(t, tt.expectedName, actualName)
			assert.LessOrEqual(t, len(actualName), 63)
		})
	}
}
