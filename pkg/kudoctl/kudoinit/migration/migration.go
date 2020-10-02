package migration

// Migrater is the base interface for migrations.
type Migrater interface {
	// CanMigrate checks if there are any conditions that would prevent this migration to run
	// This function should only return an error if it is sure that the migration can not be executed
	CanMigrate() error

	// Migrate executes the migration. The call must be idempotent and ignore already migrated resources
	// It can be called multiple times on the same cluster and encounter migrated and unmigrated resources.
	Migrate() error
}
