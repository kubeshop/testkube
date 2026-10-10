// Package watchers follows the Kubernetes objects of one execution: the job, the pod, and the
// events of both. It builds the state of the execution from them.
//
// # Events
//
// The events hold the cause of a pod that waits, for example a volume that Kubernetes cannot mount.
// The events watcher follows this contract:
//
//   - It lists the events, then watches them from the version of the list.
//   - The listener gets each version of an event one time. A list can return a version that the
//     watch already sent, so the watcher keeps the versions that it sent.
//   - After an error of a list or a watch, it lists and watches again with a delay that grows.
//     A watch that worked starts the delay again from the base. A list alone does not.
//   - An error event in the watch, for example a version that is too old, starts a new list.
//   - It retries only while the job watcher or the pod watcher observes the execution. After both
//     end, an error ends the events watcher, so the caller learns that it cannot observe the execution.
//   - A list does not hold the lock of the watcher, so a slow list does not stop another read.
//
// Before the execution watcher commits an ended state, it lists the job events and the pod events
// one more time, with a time limit. A watch can stay silent under load, and that list gives the
// state the events that the watch did not deliver.
package watchers
