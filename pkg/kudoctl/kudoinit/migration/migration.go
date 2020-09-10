package migration

type Migrater interface {
	CanMigrate() error
	Migrate() error
}
