package utils

import (
	"context"
	"io"

	dockertypes "github.com/docker/docker/api/types"
	volumetypes "github.com/docker/docker/api/types/volume"
)

// DockerClient is a wrapper interface for the Docker library to support unit testing.
type DockerClient interface {
	NegotiateAPIVersion(context.Context)
	VolumeCreate(context.Context, volumetypes.VolumeCreateBody) (dockertypes.Volume, error)
	ImageSave(context.Context, []string) (io.ReadCloser, error)
}
