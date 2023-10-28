package db_migrations

import "embed"

//go:embed *.up.json *.down.json
var MongoMigrationsFs embed.FS
