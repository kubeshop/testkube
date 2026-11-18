// Package kubernetesworker starts the Job of a Test Workflow execution in Kubernetes and follows it.
//
// # Execution namespace
//
// The worker picks the namespace of an execution in this order:
//
//  1. spec.job.namespace of the workflow, when it is set.
//  2. metadata.namespace of the workflow, when ExecuteInSourceNamespace is on and Cluster.Namespaces holds that
//     namespace. GITOPS_EXECUTE_IN_SOURCE_NAMESPACE sets ExecuteInSourceNamespace and is false by default.
//  3. The default namespace of the runner.
//
// A metadata.namespace that Cluster.Namespaces does not hold falls back to the default namespace without an error,
// because the operator did not allow it on this runner. The worker cannot tell how a workflow was created, so a
// workflow from the API or the CLI with a namespace also runs there when the flag is on. Execute refuses a namespace
// from spec.job.namespace that Cluster.Namespaces does not hold.
//
// # Service account
//
// The default service account of the chosen namespace in Cluster.Namespaces replaces the default one of the runner.
// For a source namespace it does so also when it is empty, so that Kubernetes uses the default account of that
// namespace. A service account in spec.pod of the workflow wins over both.
package kubernetesworker
