package storage

type Client interface {
	CreateBucket(bucket string) error
	DeleteBucket(bucket string, force bool) error
	ListBuckets() ([]string, error)
	ListFiles(bucket string) ([]string, error)
	SaveFile(bucket, filePath string) error
	DownloadFile(bucket, file string) error
}
