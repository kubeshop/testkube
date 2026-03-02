package scrapertypes

import (
	"io"
)

type DataType string

type Object struct {
	Name     string
	Size     int64
	Data     io.Reader
	DataType DataType
}
