package executioncache

// Args is the cache specification the processor hands to the toolkit.
//
// It lives here rather than in either of them because both sides have to agree on it
// exactly: the processor encodes it into a container argument and the toolkit decodes it
// in another process, so a field that drifted would fail at runtime, in a pod, and only
// for workflows that use the field.
//
// It travels base64-encoded. testworkflow-init resolves every container argument with
// expressions.FinalizerFail and exits the step when one cannot be resolved, and Key
// legitimately holds an expression only the toolkit can satisfy - hash_files() over a
// lockfile that exists once the repository is cloned. A missing lockfile has to be a
// cache miss, not a failed step, so the payload stays inert until the toolkit reads it.
type Args struct {
	// Key is the unresolved key template, resolved by the toolkit in the pod.
	Key string `json:"key"`
	// RestoreKeys are unresolved fallback prefixes, in declared priority order.
	RestoreKeys []string `json:"restoreKeys,omitempty"`
	// Paths are the directories to cache, relative to the working directory.
	Paths []string `json:"paths"`
	// Scope is the sharing boundary, already defaulted by the processor.
	Scope string `json:"scope,omitempty"`
	// State is where the restore stage records what it resolved, so the save stage can
	// read it back. Empty on the save side, which takes it from the environment.
	State string `json:"state,omitempty"`
}

// Hit describes how a restore resolved.
type Hit string

const (
	// HitExact means the key itself was found, so there is nothing new to save.
	HitExact Hit = "exact"
	// HitPartial means a restore key matched, so the step should still save its own key.
	HitPartial Hit = "partial"
	// HitMiss means nothing was restored.
	HitMiss Hit = "miss"
)

// State is what the restore stage tells the save stage.
//
// The save stage cannot simply recompute the key. The two run in separate containers,
// and an install may rewrite the very lockfile the key hashes - npm ci does - so a
// recomputed key could differ from the one just looked up, and the entry would be stored
// where nothing will ever search for it.
type State struct {
	// Key is the key as the restore stage resolved it.
	Key string `json:"key"`
	// Hit is how the restore resolved.
	Hit Hit `json:"hit"`
	// MatchedKey is the entry actually restored, empty on a miss.
	MatchedKey string `json:"matchedKey,omitempty"`
	// Paths are the resolved absolute paths, so the save stage packs exactly what was
	// restored into.
	Paths []string `json:"paths,omitempty"`
	// Scope is the sharing boundary the restore used.
	Scope string `json:"scope,omitempty"`
}
