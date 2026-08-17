package gui

import (
	"errors"
	"os"
	"strings"
	"testing"

	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/jesseduffield/lazydocker/pkg/commands"
	"github.com/jesseduffield/lazydocker/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestGeneratedConfigFilePattern(t *testing.T) {
	assert.Equal(t, "lazydocker-service-api-patients-*.yaml", generatedConfigFilePattern("service", "/api patients"))
	assert.Equal(t, "lazydocker-container-config-*.yaml", generatedConfigFilePattern("container", "///"))
}

func TestWithGeneratedConfigFileCreatesPrivateFileAndRemovesIt(t *testing.T) {
	osCommand := commands.NewDummyOSCommand()
	var filename string

	err := withGeneratedConfigFile(osCommand, "service", "apipatients", "services:\n  apipatients: {}\n", func(path string) error {
		filename = path
		info, err := os.Stat(path)
		if !assert.NoError(t, err) {
			return err
		}
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

		content, err := os.ReadFile(path)
		if !assert.NoError(t, err) {
			return err
		}
		assert.True(t, strings.HasPrefix(string(content), generatedConfigWarning))
		assert.Contains(t, string(content), "apipatients")
		return nil
	})

	if !assert.NoError(t, err) {
		return
	}
	_, err = os.Stat(filename)
	assert.True(t, os.IsNotExist(err))
}

func TestWithGeneratedConfigFileRemovesFileWhenEditorFails(t *testing.T) {
	osCommand := commands.NewDummyOSCommand()
	var filename string
	editorErr := errors.New("editor failed")

	err := withGeneratedConfigFile(osCommand, "project", "accloud-lde", "services: {}\n", func(path string) error {
		filename = path
		return editorErr
	})

	assert.ErrorIs(t, err, editorErr)
	_, err = os.Stat(filename)
	assert.True(t, os.IsNotExist(err))
}

func TestContainerInspectYamlIncludesFullInspectResponse(t *testing.T) {
	ctr := &commands.Container{
		Details: dockerContainer.InspectResponse{
			ContainerJSONBase: &dockerContainer.ContainerJSONBase{
				ID: "container-id",
				HostConfig: &dockerContainer.HostConfig{
					NetworkMode: "bridge",
				},
			},
			Config: &dockerContainer.Config{
				Image: "example/image:tag",
				Env:   []string{"SECRET=value"},
			},
		},
	}

	content, err := containerInspectYaml(ctr)

	if !assert.NoError(t, err) {
		return
	}
	assert.Contains(t, string(content), "Id: container-id")
	assert.Contains(t, string(content), "NetworkMode: bridge")
	assert.Contains(t, string(content), "SECRET=value")
}

func TestComposeConfigAvailabilityError(t *testing.T) {
	gui := &Gui{
		DockerCommand: &commands.DockerCommand{LocalProjectName: "local"},
		Tr: &i18n.TranslationSet{
			ComposeConfigUnavailable: "not in compose directory",
			CannotViewNonLocalConfig: "non-local project",
		},
	}

	assert.Equal(t, "not in compose directory", gui.composeConfigAvailabilityError("local"))

	gui.DockerCommand.InDockerComposeProject = true
	assert.Equal(t, "non-local project", gui.composeConfigAvailabilityError("remote"))
	assert.Empty(t, gui.composeConfigAvailabilityError("local"))
}
