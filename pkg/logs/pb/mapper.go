package pb

import (
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

// TODO figure out how to pass errors better
func MapResponseToPB(r events.LogResponse) *Log {
	log := r.Log
	content := log.Content
	isError := false
	if r.Error != nil {
		content = r.Error.Error()
		isError = true
	}
	return &Log{
		Time:     timestamppb.New(log.Time),
		Content:  content,
		Error:    isError,
		Type:     log.Type_,
		Source:   log.Source,
		Metadata: log.Metadata,
		Version:  string(log.Version),
	}
}

func MapFromPB(log *Log) events.Log {
	return events.Log{
		Time:     log.Time.AsTime(),
		Content:  log.Content,
		Error_:   log.Error,
		Type_:    log.Type,
		Source:   log.Source,
		Metadata: log.Metadata,
		Version:  log.Version,
	}
}
