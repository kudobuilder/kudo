package migration

type Migrator interface {
	CanMigrate() error
	Migrate() error
}
