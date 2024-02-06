package pb

import (
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

func MapResponseToPB(r events.LogResponse) *Log {
	log := r.Log
	if r.Error != nil {
		log.Content = r.Error.Error()
	}
	return MapToPB(log)
}

func MapToPB(r events.Log) *Log {
	return &Log{
		Time:     timestamppb.New(r.Time),
		Content:  r.Content,
		Error:    r.Error_,
		Type:     r.Type_,
		Source:   r.Source,
		Metadata: r.Metadata,
		Version:  r.Version,
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
