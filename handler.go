package simpleproto2

import "log"

// ErrorHandler handles irremediable events.
type ErrorHandler interface {
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.

	// Handle handles any error deemed irremediable by an OpenTelemetry
	// component.
	Handle(error)
	// DO NOT CHANGE: any modification will not be backwards compatible and
	// must never be done outside of a new major release.
}

func Handle(h ErrorHandler, err error) { h.Handle(err) }

type BasicErrorHandler struct {
}

func (b BasicErrorHandler) Handle(err error) {
	log.Default().Println(err)
}
