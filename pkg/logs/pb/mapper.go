package pb

import (
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

// TODO figure out how to pass errors better
func MapResponseToPB(r events.LogResponse) *LogResponse {
	chunk := r.Log
	content := chunk.Content
	isError := false
	if r.Error != nil {
		content = r.Error.Error()
		isError = true
	}
	return &LogResponse{
		Time:     timestamppb.New(chunk.Time),
		Content:  content,
		Error:    isError,
		Type:     chunk.Type,
		Source:   chunk.Source,
		Metadata: chunk.Metadata,
		Version:  string(chunk.Version),
	}
}

func MapFromPB(chunk *LogResponse) events.Log {
	return events.Log{
		Time:     chunk.Time.AsTime(),
		Content:  chunk.Content,
		Error:    chunk.Error,
		Type:     chunk.Type,
		Source:   chunk.Source,
		Metadata: chunk.Metadata,
		Version:  events.LogVersion(chunk.Version),
	}
}
