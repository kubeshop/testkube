package client

import "time"

// MockCopyFileAPI is the mock API client for uploading files to be used in tests
type MockCopyFileAPI struct {
	UploadFileFn func(parentName string, parentType TestingType, filePath string, fileContent []byte, timeout time.Duration) error
}

func (m *MockCopyFileAPI) UploadFile(parentName string, parentType TestingType, filePath string, fileContent []byte, timeout time.Duration) error {
	if m.UploadFileFn == nil {
		panic("not implemented")
	}
	return m.UploadFileFn(parentName, parentType, filePath, fileContent, timeout)

}
