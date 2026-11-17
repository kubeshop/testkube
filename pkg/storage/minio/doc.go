// Package minio is the S3-compatible object storage client of the agent.
//
// # Object retention
//
// Retention is driven by STORAGE_EXPIRATION (the whole bucket, in days) and
// STORAGE_CACHE_EXPIRATION (step dependency caches under the .tkcache/v1 prefix, in days).
// STORAGE_CACHE_EXPIRATION defaults to 1 day; STORAGE_EXPIRATION stays opt-in. The difference is
// the filter, not taste: the cache rule is confined to the cache prefix and can only delete
// caches, where the bucket-wide rule is unfiltered and governs artifacts and logs too, so a
// default there would delete a deployment's results on an upgrade. TestExpirationDefaults pins
// both.
//
// SetExpirationPolicies applies them with SetBucketLifecycle, which replaces the bucket
// lifecycle wholesale, so it reads the existing configuration first and carries through every
// rule Testkube does not own, matched by ID (mergeLifecycleRules). That merge is what makes a
// default safe at all: without it, defaulting either setting would drop the rules of
// installations whose bucket lifecycle is managed elsewhere, purely by upgrading. It fails
// closed: if the existing lifecycle cannot be read, nothing is written.
//
// That read is a permission earlier versions did not need (s3:GetLifecycleConfiguration on S3
// and MinIO, storage.buckets.get on GCS), and because STORAGE_CACHE_EXPIRATION defaults to 1 the
// call happens on every installation rather than only those configuring an expiration, so an
// upgrade can need permissions the deployment never granted. The error names them.
//
// The bucket-wide rule is unfiltered and so covers cache objects too, and the earlier expiration
// wins: a cache TTL can only bring eviction forward, never postpone it, and MustGetMinioClient
// warns when it is set longer than the bucket-wide one.
package minio
