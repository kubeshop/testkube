package files

type File interface {
	GetContents(location string) (string, error)
}
