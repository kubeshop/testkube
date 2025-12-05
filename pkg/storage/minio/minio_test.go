package minio

import (
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

func TestVirtualHostedStyleOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		useVirtualHostedStyle     bool
		expectedBucketLookupType  minio.BucketLookupType
	}{
		{
			name:                     "virtual hosted style enabled",
			useVirtualHostedStyle:    true,
			expectedBucketLookupType: minio.BucketLookupDNS,
		},
		{
			name:                     "virtual hosted style disabled",
			useVirtualHostedStyle:    false,
			expectedBucketLookupType: minio.BucketLookupPath,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts []Option
			if tt.useVirtualHostedStyle {
				opts = append(opts, VirtualHostedStyle())
			}

			connecter := NewConnecter(
				"test-endpoint.com",
				"access-key",
				"secret-key",
				"us-east-1",
				"",
				"test-bucket",
				zap.NewNop().Sugar(),
				opts...,
			)

			// Apply options
			for _, opt := range connecter.Opts {
				err := opt(connecter)
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.useVirtualHostedStyle, connecter.UseVirtualHostedStyle)
		})
	}
}
