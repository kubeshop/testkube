package proto

// Exclude legacy proto definitions from new generation.
// TODO: migrate legacy proto definitions to new generation so they can be removed from here.
//go:generate go tool -modfile=internal/buf/go.mod buf generate --exclude-path service.proto --exclude-path logs.proto

// Generate legacy proto definitions.
// This uses an older buf generation that builds to a different location and using older generation plugins.
//go:generate go tool -modfile=internal/buf/go.mod buf generate --template buf.gen.old.yaml
