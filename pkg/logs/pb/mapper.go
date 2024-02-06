package pb

import (
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

// TODO figure out how to pass errors better
func MapResponseToPB(r events.LogResponse) *Log {
	chunk := r.Log
	content := chunk.Content
	isError := false
	if r.Error != nil {
		content = r.Error.Error()
		isError = true
	}
	return &Log{
		Time:     timestamppb.New(chunk.Time),
		Content:  content,
		Error:    isError,
		Type:     chunk.Type,
		Source:   chunk.Source,
		Metadata: chunk.Metadata,
		Version:  string(chunk.Version),
	}
}

func MapFromPB(l *Log) events.Log {
	return events.Log{
		Time:     l.Time.AsTime(),
		Content:  l.Content,
		Error:    l.Error,
		Type:     l.Type,
		Source:   l.Source,
		Metadata: l.Metadata,
		Version:  events.LogVersion(l.Version),
	}
}
