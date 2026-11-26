// Package watchers follows the Kubernetes objects of one execution: the job, the pod, and the
// events of both. It builds the state of the execution from them.
//
// # The job and the pod
//
// The job watcher and the pod watcher follow the same contract as each other:
//
//   - Each lists its object, then watches it from the version of the list.
//   - A watch that ends without the context being done is a failure, not a success. That covers a
//     watch the API server closed with no event, and an error event in the watch such as a version
//     that is too old.
//   - After a failure the watcher lists again, so the next watch starts from a version the API
//     server still knows, and waits first with a delay that grows. A watch that delivered an event
//     starts the delay again from the base.
//   - A list repeats the current object to the listener. The listener stores one value per
//     execution, so a repeat replaces it rather than adding to it.
//
// Before this contract the two watchers reported a closed watch as a success and reconnected at
// once on the same version, which an API server restart turned into thousands of rejected watches
// a second.
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
//     A watch that delivered an event starts the delay again from the base. A list alone does not.
//   - An error event in the watch, for example a version that is too old, starts a new list.
//   - A watch that closes with no event counts as an error, because the watch has no time limit of
//     its own. A watch that closes after an event starts again at once from the last version.
//   - It retries only while the job watcher or the pod watcher observes the execution. After both
//     end, an error ends the events watcher, and Err returns that error.
//   - A list does not hold the lock of the watcher, so a slow list does not stop another read.
//
// Before the execution watcher commits an ended state, it lists the job events and the pod events
// one more time, in parallel and with a time limit. A watch can stay silent under load, and that
// list gives the state the events that the watch did not deliver. The list works after an error
// ended the events watcher, and the update of the execution watcher closes only after that commit.
package watchers
