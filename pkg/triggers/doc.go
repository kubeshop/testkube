// Package triggers runs TestTriggers: it watches the events they select and schedules their
// actions.
//
// # HTTP content triggers
//
// A trigger with contentSelector.http fires the http-changed event. http_trigger.go polls it as
// the trigger-http-watcher leader task registered in NewService. Unlike the git informer it lists
// nothing: it polls the triggers already in the service's triggerStatus, one goroutine per
// trigger, restarted with a fresh baseline when the HTTP spec changes.
//
// The last value is in memory only, so after a restart or a leader change the first poll is a
// baseline again (or fires, with fireOnFirstObservation). A failed poll (a network error, an
// unexpected status, a missing field) is never a change, and a change whose firing fails is
// retried on the next poll.
//
// The event object is {value, statusCode, url} and the old object {value}, so match[] paths and
// run config expressions read .value or resource.value, and oldResource.value. Validation and the
// interval and timeout defaults live in api/testtriggers/v1/validation_content.go. Unlike other
// match[] triggers an HTTP one needs no listener binding, because it matches the response rather
// than a cluster object.
package triggers
