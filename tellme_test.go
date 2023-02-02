package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func getDefaults() []configFileValue {
	configDefaults := []configFileValue{
		{
			comment: "yes-no option `[yes | no]`. Default no",
			key:     "YESNOOPT",
			value:   "no",
			fname:   "y",
			ftype:   "yesno",
		}, {
			comment: "`path` option. Default .",
			key:     "PATHOPT",
			value:   ".",
			fname:   "p",
			ftype:   "path",
		}, {
			comment: "language `[en | es | de | etc]`. Default en",
			key:     "LANG",
			value:   "en",
			fname:   "l",
			ftype:   "lang",
		}, {
			comment: "audio files type `[mp3 | ogg ]`. Default mp3",
			key:     "ATYPE",
			value:   "mp3",
			fname:   "t",
			ftype:   "aformat",
		},
	}
	return configDefaults
}

func TestSetDefaultConfigValues(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{
			name: "Yes/no option",
			key:  "YESNOOPT",
			val:  "no",
		}, {
			name: "Path option",
			key:  "PATHOPT",
			val:  ".",
		}, {
			name: "Language option",
			key:  "LANG",
			val:  "en",
		}, {
			name: "Audio file type",
			key:  "ATYPE",
			val:  "mp3",
		},
	}

	cfg := setDefaultConfigValues(getDefaults())

	t.Run("Number of config options", func(t *testing.T) {
		if len(cfg) != 4 {
			t.Errorf("len(cfg) == %d; expected %d", len(cfg), len(tests))
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := cfg[tt.key]
			if !ok {
				t.Errorf("Can not find key %s in default config", tt.key)
			}

			if value != cfg[tt.key] {
				t.Errorf("cfg[%s] == %s; expected %s", tt.key, value, tt.val)
			}
		})
	}
}

func TestCreateNewConf(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_config")

	cfgDefaults := getDefaults()
	cfg := setDefaultConfigValues(cfgDefaults)

	createNewConf(cfg, tmpFile, cfgDefaults)
	fileContent, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Errorf("Can not read test config: %s", err)
	}
	got := string(fileContent)
	want := fmt.Sprintf(`# TellMe configuration file

# yes-no option %c[yes | no]%c. Default no
YESNOOPT=no

# %cpath%c option. Default .
PATHOPT=.

# language %c[en | es | de | etc]%c. Default en
LANG=en

# audio files type %c[mp3 | ogg ]%c. Default mp3
ATYPE=mp3

`, '`', '`', '`', '`', '`', '`', '`', '`')
	if got != want {
		t.Errorf("Wrong content of test config. Got:\n`%s`\nWant:\n`%s`", got, want)
	}
}
