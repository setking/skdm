package store

import (
	dv1 "changeme/backed/api/apiserver/v1"
)

// EventStore defines the user storage interface.
type EventStore interface {
	InsertEvent(gid, eventType, eventData string) error
	ListEventsByGID(gid string) ([]dv1.EventRecord, error)
}
