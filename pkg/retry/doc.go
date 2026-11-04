// Package retry is the one way to retry a bounded call in Testkube.
//
// A Policy names how many attempts a call gets, the wait between them, and which errors are worth
// another attempt. Do runs the call under the policy:
//
//   - The wait before attempt n+1 is n times the delay by default. An exponential policy doubles
//     the delay n-1 times, and a fixed policy keeps it. A wait is never more than the maximum delay.
//   - No wait follows the last attempt.
//   - An error that RetryIf rejects returns at once. A nil RetryIf retries every error.
//   - A context that ends stops the loop. The error then holds the last error of the call and the
//     cause of the context, so the caller still sees what failed.
//
// A loop that runs for the whole life of a process, for example a poll loop or a watch, does not
// use this package. It keeps a github.com/cloudflare/backoff value, takes Duration after each
// failure, and calls Reset after a success.
package retry
