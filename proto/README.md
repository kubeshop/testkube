# Testkube protobuf definitions

This directory stores all of the Testkube protobuf definitions for gRPC communication with a Control Plane.

All protobuf definitions are built, linted, and generated using [`buf`](https://buf.build/).

## How to use this?

For all new gRPC endpoint create an appropriate package, probably under the `testkube` directory.
`buf lint` can be used to ensure you are correctly structuring your protobuf and not breaking backwards compatibility.
Try to keep gRPC `Services` separate, one of the issues we have with the older protobuf definitions is that it is difficult to split up
`rpc`s and required permissions when dealing with newer multi-agent architecture and more limited agent types (such as Runner or Listener agents).

Attempt to follow protobuf best practices for all new protobuf definitions:
- https://protobuf.dev/best-practices/dos-donts/
- https://protobuf.dev/best-practices/1-1-1/

Once you have created your protobuf you can simply `go generate` and the generate statement in `gen.go` will handle the rest for you.

For more detailed information about what is linted and what configuration options are available check out the [`buf` documentation](https://buf.build/docs/).

## Why `buf`?

Using `buf` gives us a number of improvements over the default `protoc`:
- Can be installed as part of our `go tool` set, ensuring that every contributor can easily have the same version of required build tools.
  - Using `protoc` requires installing specific versions of `protoc`, `proto-gen-go`, and `proto-gen-go-grpc` on contributor's machines.
- Simplifies setup to a simple `go generate ./proto`. This will install the correct version of `buf` and build and generate Go code from protobuf definitions.
- Opinionated protobuf definition linting, this will ensure consistency and standardisation across protobuf definitions and generated code.
- Configuration files ensure standard code generation across all protobuf definitions.
