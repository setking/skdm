package v1

import (
	dv1 "changeme/backed/api/apiserver/v1"
	"changeme/backed/internal/apiserver/store"
)

// EventSrv defines functions used to handle user request.
type EventSrv interface {
	InsertEvent(gid, eventType, eventData string) error
	ListEventsByGID(gid string) ([]dv1.EventRecord, error)
}

type eventService struct {
	store store.Factory
}

func (e eventService) InsertEvent(gid, eventType, eventData string) error {
	return e.store.Events().InsertEvent(gid, eventType, eventData)
}

func (e eventService) ListEventsByGID(gid string) ([]dv1.EventRecord, error) {
	return e.store.Events().ListEventsByGID(gid)
}

var _ EventSrv = (*eventService)(nil)

func newEvent(srv *service) *eventService {
	return &eventService{store: srv.store}
}
