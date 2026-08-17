package gui

import (
	"sort"
	"testing"

	"github.com/jesseduffield/lazydocker/pkg/commands"
	"github.com/jesseduffield/lazydocker/pkg/config"
	"github.com/jesseduffield/lazydocker/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestBindingGetKeyUsesDisplayKey(t *testing.T) {
	binding := &Binding{Key: 'g', DisplayKey: "gg"}

	assert.Equal(t, "gg", binding.GetKey())
}

func TestLogsAndConfigEditorBindingsShareTheSamePanels(t *testing.T) {
	userConfig := config.GetDefaultConfig()
	appConfig := &config.AppConfig{UserConfig: &userConfig}
	log := commands.NewDummyLog()
	osCommand := commands.NewOSCommand(log, appConfig)
	dockerCommand := &commands.DockerCommand{Config: appConfig, OSCommand: osCommand, Log: log}
	tr := i18n.NewTranslationSet(log, "en")
	gui, err := NewGui(log, dockerCommand, osCommand, tr, appConfig, make(chan error))
	if !assert.NoError(t, err) {
		return
	}
	gui.SetupFakeGui()

	viewsByKey := map[rune][]string{'m': {}, 'M': {}}
	for _, binding := range gui.GetInitialKeybindings() {
		key, ok := binding.Key.(rune)
		if !ok || (key != 'm' && key != 'M') {
			continue
		}
		viewsByKey[key] = append(viewsByKey[key], binding.ViewName)
	}
	for key := range viewsByKey {
		sort.Strings(viewsByKey[key])
	}

	expectedViews := []string{"containers", "project", "services"}
	assert.Equal(t, expectedViews, viewsByKey['m'])
	assert.Equal(t, expectedViews, viewsByKey['M'])
}
