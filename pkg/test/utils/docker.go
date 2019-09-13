package utils

import (
	"context"

	dockertypes "github.com/docker/docker/api/types"
	volumetypes "github.com/docker/docker/api/types/volume"
)

// DockerClient is a wrapper interface for the Docker library to support unit testing.
type DockerClient interface {
	VolumeCreate(context.Context, volumetypes.VolumesCreateBody) (dockertypes.Volume, error)
}
