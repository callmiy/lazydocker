package commands

import (
	"os/exec"
	"testing"

	"github.com/jesseduffield/lazydocker/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestServiceResolvedConfigUsesConfiguredTemplateAndProject(t *testing.T) {
	userConfig := config.GetDefaultConfig()
	userConfig.CommandTemplates.DockerCompose = "custom-compose --file stack.yml"
	userConfig.CommandTemplates.ServiceConfig = "{{ .DockerCompose }} render-service {{ .Service.Name }}"
	appConfig := &config.AppConfig{UserConfig: &userConfig}
	osCommand := NewOSCommand(NewDummyLog(), appConfig)

	var commandName string
	var commandArgs []string
	osCommand.SetCommand(func(name string, args ...string) *exec.Cmd {
		commandName = name
		commandArgs = args
		return exec.Command("printf", "resolved service config")
	})

	dockerCommand := &DockerCommand{Config: appConfig, OSCommand: osCommand}
	service := &Service{
		Name:          "apipatients",
		ProjectName:   "accloud-lde",
		OSCommand:     osCommand,
		DockerCommand: dockerCommand,
	}

	output, err := service.ResolvedConfig()

	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "resolved service config", output)
	assert.Equal(t, "custom-compose", commandName)
	assert.Equal(t, []string{"--file", "stack.yml", "-p", "accloud-lde", "render-service", "apipatients"}, commandArgs)
}

func TestServiceResolvedConfigReturnsCommandError(t *testing.T) {
	userConfig := config.GetDefaultConfig()
	appConfig := &config.AppConfig{UserConfig: &userConfig}
	osCommand := NewOSCommand(NewDummyLog(), appConfig)
	osCommand.SetCommand(func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 23")
	})
	dockerCommand := &DockerCommand{Config: appConfig, OSCommand: osCommand}
	service := &Service{Name: "broken", ProjectName: "local", OSCommand: osCommand, DockerCommand: dockerCommand}

	_, err := service.ResolvedConfig()

	assert.Error(t, err)
}
