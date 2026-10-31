// Package webhook delivers Testkube events to webhook subscribers.
//
// # Delivery retries
//
// The listener (listener.go) retries the outbound HttpClient.Do for sendRetryCount attempts with a
// linear sendRetryBaseDelay. Network errors, 5xx and 429 are retried. Any other 4xx
// short-circuits, so a bad URL or an authentication failure is not spammed at the subscriber.
// Delivery is intentionally at-least-once: subscribers own deduplication, as with Stripe, GitHub
// and Slack webhooks.
package webhook
