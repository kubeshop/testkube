package v1

// Internal Testkube naming for the git-integration actor. The wire value on
// the CRD (see the swagger-annotated
// GITINTEGRATION_TestWorkflowRunningContextActorType) stays as
// "gitintegration" so customer-applied YAML keeps the provider-agnostic name;
// Go code reads QUALITYLOOP to reflect the internal product concept.
const QUALITYLOOP_TestWorkflowRunningContextActorType = GITINTEGRATION_TestWorkflowRunningContextActorType
