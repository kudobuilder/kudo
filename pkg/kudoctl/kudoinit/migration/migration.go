package migration

type Migrater interface {
	// CanMigrate checks if there are any conditions that would prevent this migration to run
	CanMigrate() error

	// Migrate executes the
	Migrate() error
}
