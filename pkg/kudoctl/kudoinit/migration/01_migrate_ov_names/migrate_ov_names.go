//nolint:golint,stylecheck
package _01_migrate_ov_names

import (
	"context"
	"fmt"

	"github.com/thoas/go-funk"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

	migratedOVs map[string]*kudoapi.OperatorVersion
}

func New(client *kube.Client, dryRun bool) *MigrateOvNames {
	return &MigrateOvNames{
		client:      client,
		dryRun:      dryRun,
		migratedOVs: map[string]*kudoapi.OperatorVersion{},
	}
}

func (m *MigrateOvNames) CanMigrate() error {
	return nil
}

func (m *MigrateOvNames) Migrate() error {
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
	if !m.dryRun {
		var err error
		clog.V(4).Printf("Create copy of OperatorVersion with name %q", ov.Name)
		ov, err = m.client.KudoClient.KudoV1beta1().OperatorVersions(ov.Namespace).Create(context.TODO(), ov, v1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	// Store migrated operator versions for instance owner references
	m.migratedOVs[oldName] = ov

	if !m.dryRun {
		clog.V(4).Printf("Delete old OperatorVersion with name %q", oldName)
		if err := m.client.KudoClient.KudoV1beta1().OperatorVersions(ov.Namespace).Delete(context.TODO(), oldName, v1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete old OperatorVersion %q: %v", oldName, err)
		}
	}
	return nil
}

func (m *MigrateOvNames) migrateInstance(i *kudoapi.Instance) error {
	newOwner, ovWasMigrated := m.migratedOVs[i.Spec.OperatorVersion.Name]
	if ovWasMigrated {
		oldName := i.Spec.OperatorVersion.Name
		clog.V(1).Printf("Adjust OperatorVersion and owner reference of Instance %q", i.Name)

		// Set OperatorVersion
		i.Spec.OperatorVersion.Name = newOwner.Name

		// Replace OwnerReference
		//nolint:errcheck
		i.OwnerReferences = funk.Filter(i.OwnerReferences, func(o v1.OwnerReference) bool { return o.Name != oldName }).([]v1.OwnerReference)
		if err := controllerutil.SetOwnerReference(newOwner, i, m.client.Scheme); err != nil {
			return fmt.Errorf("failed to set resource ownership for the new instance: %v", err)
		}

		// Save update
		if !m.dryRun {
			clog.V(4).Printf("Update instance %q with new owner reference", i.Name)
			_, err := m.client.KudoClient.KudoV1beta1().Instances(i.Namespace).Update(context.TODO(), i, v1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update Instance %q with new owner reference: %v", i.Name, err)
			}
		}
	}
	return nil
}
