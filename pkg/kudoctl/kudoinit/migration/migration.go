package migration

import "github.com/kudobuilder/kudo/pkg/kudoctl/kube"

type Migrater interface {
	CanMigrate(client *kube.Client) error
	Migrate(client *kube.Client) error
}
