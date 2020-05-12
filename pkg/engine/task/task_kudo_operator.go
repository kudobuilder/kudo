package task

import (
	"errors"
)

// KudoOperatorTask installs an instance of a KUDO operator in a cluster
type KudoOperatorTask struct {
	Name            string
	Package         string
	InstanceName    string
	AppVersion      string
	OperatorVersion string
	RepoURL         string
}

// Run method for the KudoOperatorTask. Not yet implemented
func (dt KudoOperatorTask) Run(ctx Context) (bool, error) {
	return false, errors.New("kudo-operator task is not yet implemented. Stay tuned though ;)")
}
