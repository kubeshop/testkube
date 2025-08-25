# Testkube protobuf definitions

This directory stores all of the Testkube protobuf definitions for gRPC communication with a Control Plane.

It is split into two distinct parts:
- `legacy` for our older gRPC connections, these are built using `protoc` and we are gradually migrating away from many of them.
- All other protobuf definitions which are built, linted, and generated using [`buf`](https://buf.build/).

## How to use this?

For all new gRPC endpoint create an appropriate package, probably under the `testkube` directory.
`buf lint` can be used to ensure you are correctly structuring your protobuf and not breaking backwards compatibility.
Try to keep gRPC `Services` separate, one of the issues we have with the older protobuf definitions is that it is difficult to split up
`rpc`s and required permissions when dealing with newer multi-agent architecture and more limited agent types (such as Runner or Listener agents).

Once you have created your protobuf you can simply `go generate` and the generate statement in `gen.go` will handle the rest for you.

For more detailed information about what is linted and what configuration options are available check out the [`buf` documentation](https://buf.build/docs/).

## Why `buf`?

Using `buf` gives us a number of improvements over the default `protoc`:
- Can be installed as part of our `go tool` set, ensuring that every contributor can easily have the same version of required build tools.
  - Using `protoc` requires installing specific versions of `protoc`, `proto-gen-go`, and `proto-gen-go-grpc` on contributor's machines.
- Simplifies setup to a simple `go generate ./proto`. This will install the correct version of `buf` and build and generate Go code from protobuf definitions.
- Opinionated protobuf definition linting, this will ensure consistency and standardisation across protobuf definitions and generated code.
- Configuration files ensure standard code generation across all protobuf definitions.

### Why not use `buf` to generate from legacy definitions?

Whilst `buf` can make use of `protoc` and `protoc` plugins to build code they still need to be available in `$PATH`,
this requirement would prevent the use of a simple `go generate ./proto` on a clean machine to generate the new protobuf
outputs.