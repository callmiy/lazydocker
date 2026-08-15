package gui

import (
	"testing"

	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/jesseduffield/lazydocker/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func TestFilterIdentityStrings(t *testing.T) {
	container := &commands.Container{
		Name: "container-name",
		ID:   "container-id",
		Container: dockerContainer.Summary{
			Image: "sha256:container-image",
		},
	}

	assert.Equal(t, []string{"project-name"}, projectFilterIdentityStrings(&commands.Project{Name: "project-name"}))
	assert.Equal(t, []string{"service-name"}, serviceFilterIdentityStrings(&commands.Service{Name: "service-name"}))
	assert.Equal(
		t,
		[]string{"service-name", "container-image"},
		serviceFilterIdentityStrings(&commands.Service{Name: "service-name", Container: container}),
	)
	assert.Equal(t, []string{"container-name", "container-id", "container-image"}, containerFilterIdentityStrings(container))
	assert.Equal(
		t,
		[]string{"image-name", "image-tag", "image-id"},
		imageFilterIdentityStrings(&commands.Image{Name: "image-name", Tag: "image-tag", ID: "sha256:image-id"}),
	)
	assert.Equal(t, []string{"volume-name"}, volumeFilterIdentityStrings(&commands.Volume{Name: "volume-name"}))
	assert.Equal(t, []string{"network-name"}, networkFilterIdentityStrings(&commands.Network{Name: "network-name"}))
}
