package gui

import (
	"testing"

	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/jesseduffield/lazydocker/pkg/commands"
	"github.com/jesseduffield/lazydocker/pkg/gui/panels"
	"github.com/stretchr/testify/assert"
)

func TestFilterIdentities(t *testing.T) {
	container := &commands.Container{
		Name: "container-name",
		ID:   "container-id",
		Container: dockerContainer.Summary{
			Image: "sha256:container-image",
		},
	}

	assert.Equal(t, []panels.FilterIdentity{{Value: "project-name", DisplayCell: 0}}, projectFilterIdentities(&commands.Project{Name: "project-name"}))
	assert.Equal(t, []panels.FilterIdentity{{Value: "service-name", DisplayCell: 2}}, serviceFilterIdentities(&commands.Service{Name: "service-name"}))
	assert.Equal(
		t,
		[]panels.FilterIdentity{{Value: "service-name", DisplayCell: 2}, {Value: "container-image", DisplayCell: 5}},
		serviceFilterIdentities(&commands.Service{Name: "service-name", Container: container}),
	)
	assert.Equal(
		t,
		[]panels.FilterIdentity{
			{Value: "container-name", DisplayCell: 2},
			{Value: "container-id", DisplayCell: panels.HiddenFilterIdentityCell},
			{Value: "container-image", DisplayCell: 5},
		},
		containerFilterIdentities(container),
	)
	assert.Equal(
		t,
		[]panels.FilterIdentity{
			{Value: "image-name", DisplayCell: 0},
			{Value: "image-tag", DisplayCell: 1},
			{Value: "image-id", DisplayCell: panels.HiddenFilterIdentityCell},
		},
		imageFilterIdentities(&commands.Image{Name: "image-name", Tag: "image-tag", ID: "sha256:image-id"}),
	)
	assert.Equal(t, []panels.FilterIdentity{{Value: "volume-name", DisplayCell: 1}}, volumeFilterIdentities(&commands.Volume{Name: "volume-name"}))
	assert.Equal(t, []panels.FilterIdentity{{Value: "network-name", DisplayCell: 1}}, networkFilterIdentities(&commands.Network{Name: "network-name"}))
}
