package testkube

// Internal Testkube naming for the git-integration actor. The wire value
// stays as "gitintegration" (see the swagger-generated
// GITINTEGRATION_TestWorkflowRunningContextActorType) so every customer-facing
// surface (REST, CLI, CRD, helm charts) keeps the provider-agnostic name;
// Go code reads QUALITYLOOP to reflect the internal product concept.
const QUALITYLOOP_TestWorkflowRunningContextActorType = GITINTEGRATION_TestWorkflowRunningContextActorType
