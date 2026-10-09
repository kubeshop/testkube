// Package imageinspector reads the metadata of a container image from the registry and keeps it in
// a cache. The metadata holds the entrypoint, the command, the user and the shell of the image.
//
// # Error messages
//
// An error of this package becomes the message of a failed execution, so it names each fact one
// time, in words for the user:
//
//   - The inspector names the image once: `the image "<name>" cannot be read: <cause>`. A pull
//     secret that cannot be read gives `the pull secret "<name>" cannot be read: <cause>`.
//   - The fetcher reads the answer of the registry from the structured error of the registry client,
//     not from its text. It writes `the registry <host> answered <CODE>: <text>`, without the URL
//     and without the details of the answer. It drops the text when the text only repeats the code.
//   - For the common codes, it adds one sentence with the next step. Docker Hub answers
//     UNAUTHORIZED also for a repository that does not exist, so that sentence names both causes.
//   - An error that does not come from the registry, for example a DNS or a TLS error, keeps its text.
package imageinspector
