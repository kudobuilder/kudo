package migration

var _ Migrator = &To1_11Migration{}

type To1_11Migration struct {
}

func To1_11() Migrator {
	return &To1_11Migration{}
}

func (m *To1_11Migration) String() string {
	return "to 1.11"
}

func (m *To1_11Migration) CanMigrate() error {
	return nil
}

func (m *To1_11Migration) Migrate() error {
	return nil
}
