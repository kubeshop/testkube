package testkube

// StopActor identifies the component that requested an abort or a cancel. The worker
// writes it into the job annotations, so the execution result names who stopped it.
type StopActor string

const (
	// StopActorUser is a person who cancels through the API, the CLI, or the dashboard.
	StopActorUser StopActor = "user"
	// StopActorControlPlane is the control plane, for example a queue or execution timeout.
	StopActorControlPlane StopActor = "control-plane"
	// StopActorTrigger is the trigger cleanup that stops executions of a deleted trigger.
	StopActorTrigger StopActor = "trigger"
	// StopActorFailFast is a parallel step that stops its workers after the first failure.
	StopActorFailFast StopActor = "fail-fast"
	// StopActorRunner is the runner itself, for example the check for stuck executions.
	StopActorRunner StopActor = "runner"
	// StopActorQualityLoop is the quality loop in the control plane, which cancels a run
	// that a newer commit superseded.
	StopActorQualityLoop StopActor = "quality-loop"
	// StopActorAPI is the standalone agent API abort endpoint.
	StopActorAPI StopActor = "api"
	// StopActorSystem is the fallback when the caller does not identify itself.
	StopActorSystem StopActor = "system"
)

// Sentence returns the words the result reader uses for the actor after
// "The execution has been aborted". The reason, when set, follows them.
func (a StopActor) Sentence() string {
	switch a {
	case StopActorUser:
		return "by the user"
	case StopActorControlPlane:
		return "by the control plane"
	case StopActorTrigger:
		return "because its trigger was deleted"
	case StopActorFailFast:
		return "because another parallel worker failed"
	case StopActorRunner:
		return "by the runner"
	case StopActorQualityLoop:
		return "by the quality loop"
	case StopActorAPI:
		return "through the API"
	default:
		return "by the system"
	}
}
