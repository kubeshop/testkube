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
//
// # Agent API address
//
// An execution in AgentNamespace gets Connection.LocalApiUrl with the short Service name, which every toolkit reads.
// An execution in another namespace gets CrossNamespaceLocalApiUrl, an absolute name <service>.<namespace>.svc.<domain>.
// with a trailing dot. The DNS search list of the execution pod then cannot resolve the name to a Service of the
// execution namespace. The toolkit in proxy mode (--proxy) reads the Service and its namespace from that name. Proxy
// mode needs services/proxy on the agent Service in AgentNamespace for the service account of the execution. The chart
// generates no such Role in any namespace, so an operator who uses --proxy grants it by hand.
package kubernetesworker
