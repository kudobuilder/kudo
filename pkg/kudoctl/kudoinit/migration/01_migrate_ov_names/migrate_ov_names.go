//nolint:golint,stylecheck
package _01_migrate_ov_names

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/migration"
)

var (
	_ migration.Migrater = &MigrateOvNames{}
)

type MigrateOvNames struct {
	client *kube.Client
	dryRun bool
	ctx    context.Context

	migratedOVs map[string]*kudoapi.OperatorVersion
}

func New(client *kube.Client, dryRun bool) *MigrateOvNames {
	return &MigrateOvNames{
		client:      client,
		dryRun:      dryRun,
		ctx:         context.TODO(),
		migratedOVs: map[string]*kudoapi.OperatorVersion{},
	}
}

func (m *MigrateOvNames) CanMigrate() error {
	// No migrate check required for this migration
	return nil
}

func (m *MigrateOvNames) Migrate() error {
	if !m.dryRun {
		clog.V(0).Printf("Migrate OperatorVersion names")
	}
	return migration.ForEachNamespace(m.client, m.migrateInNamespace)
}

func (m *MigrateOvNames) migrateInNamespace(ns string) error {
	clog.V(1).Printf("Run OperatorVersion name migration on namespace %q", ns)
	if err := migration.ForEachOperatorVersion(m.client, ns, m.migrateOperatorVersion); err != nil {
		return fmt.Errorf("failed to migrate OperatorVersions in namespace %q: %v", ns, err)
	}
	if err := migration.ForEachInstance(m.client, ns, m.migrateInstance); err != nil {
		return fmt.Errorf("failed to migrate Instance ownership in namespace %q: %v", ns, err)
	}

	return nil
}

func (m *MigrateOvNames) migrateOperatorVersion(ov *kudoapi.OperatorVersion) error {
	expectedName := ov.FullyQualifiedName()
	if ov.Name == expectedName {
		// Nothing to migrate
		return nil
	}

	oldName := ov.Name
	clog.V(0).Printf("Migrate OperatorVersion from %q to %q", ov.Name, ov.FullyQualifiedName())

	ov.Name = expectedName

	// Reset Read-Only fields
	ov.Status = kudoapi.OperatorVersionStatus{}
	ov.ResourceVersion = ""
	ov.Generation = 0
	ov.UID = ""
	ov.CreationTimestamp = v1.Time{}

	if !m.dryRun {
		var err error
		clog.V(4).Printf("Create copy of OperatorVersion with name %q", ov.Name)
		ov, err = m.client.KudoClient.KudoV1beta1().OperatorVersions(ov.Namespace).Create(m.ctx, ov, v1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create copy of operator version: %v", err)
		}
	}

	// Store migrated operator versions for instance migration
	m.migratedOVs[oldName] = ov

	if !m.dryRun {
		clog.V(4).Printf("Delete old OperatorVersion with name %q", oldName)
		if err := m.client.KudoClient.KudoV1beta1().OperatorVersions(ov.Namespace).Delete(m.ctx, oldName, v1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete old OperatorVersion %q: %v", oldName, err)
		}
	}
	return nil
}

func (m *MigrateOvNames) migrateInstance(i *kudoapi.Instance) error {
	newOwner, ovWasMigrated := m.migratedOVs[i.Spec.OperatorVersion.Name]
	if ovWasMigrated {
		clog.V(1).Printf("Adjust OperatorVersion of Instance %q", i.Name)

		// Set OperatorVersion
		i.Spec.OperatorVersion.Name = newOwner.Name

		// Save update
		if !m.dryRun {
			clog.V(4).Printf("Update instance %q with new owner reference", i.Name)
			_, err := m.client.KudoClient.KudoV1beta1().Instances(i.Namespace).Update(m.ctx, i, v1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update Instance %q with new operator version: %v", i.Name, err)
			}
		}
	}
	return nil
}
