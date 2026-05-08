package common

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventType string

const (
	EventTypeCreate EventType = "create"
	EventTypeUpdate EventType = "update"
	EventTypeDelete EventType = "delete"
)

func GetUpdateTime(t metav1.ObjectMeta) time.Time {
	updateTime := t.CreationTimestamp.Time
	if t.DeletionTimestamp != nil {
		updateTime = t.DeletionTimestamp.Time
	} else {
		for _, field := range t.ManagedFields {
			if field.Time != nil && field.Time.After(updateTime) {
				updateTime = field.Time.Time
			}
		}
	}

	return updateTime
}
