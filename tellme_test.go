package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestUpdateFromConfigFile(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{
			name: "Nonchanged option",
			key:  "NONCHANGED",
			val:  "val",
		}, {
			name: "Changed option",
			key:  "CHANGED",
			val:  "val_new",
		}, {
			name: "Nonexisted option",
			key:  "NONEXISTED",
			val:  "val_new",
		},
	}

	tmpDir, err := os.MkdirTemp("", t.Name()+"*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		dirs, err := filepath.Glob(filepath.Join(os.TempDir(), t.Name()+"*"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, dir := range dirs {
			os.RemoveAll(dir)
		}
	}()
	tmpFile := filepath.Join(tmpDir, "test_config")
	f, err := os.OpenFile(tmpFile, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(f, "NONCHANGED=val")
	fmt.Fprintln(f, "CHANGED=val_new")
	fmt.Fprintln(f, "NONEXISTED=val_new")
	err = f.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg := make(Config)
	cfg["NONCHANGED"] = "val"
	cfg["CHANGED"] = "val"

	t.Run("Fails on nonexisted option", func(t *testing.T) {
		if os.Getenv("BE_CRASHER") == "1" {
			updateFromConfigFile(cfg, tmpFile)
			return
		}
		cmd := exec.Command(os.Args[0], "--test.run=TestUpdateFromConfigFile")
		cmd.Env = append(os.Environ(), "BE_CRASHER=1")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err = cmd.Run()
		rightErr := strings.Index(stderr.String(), "nonexisting key in config file")
		if e, ok := err.(*exec.ExitError); ok && e.Success() || rightErr != 0 {
			t.Error("Should fail on nonexisting option")
		}
	})

	cfg["NONEXISTED"] = "val"
	newCfg := updateFromConfigFile(cfg, tmpFile)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.val != newCfg[tt.key] {
				t.Errorf("cfg[%s] == %s; expected %s", tt.key, newCfg[tt.key], tt.val)
			}
		})
	}
}

func TestUpdateFromCmdLine(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{
			name: "Audio type",
			key:  "ATYPE",
			val:  "ogg",
		}, {
			name: "Language",
			key:  "LANG",
			val:  "nl",
		}, {
			name: "Path",
			key:  "PATHOPT",
			val:  "/random/path",
		}, {
			name: "Yes/no option",
			key:  "YESNOOPT",
			val:  "yes",
		},
	}
	cfgDefaults := getDefaults()
	config = setDefaultConfigValues(cfgDefaults)

	tmp := make([]string, len(os.Args))
	copy(tmp, os.Args)
	os.Args = []string{"test", "-y", "yes", "-p", "/random/path",
		"-l", "nl", "-t", "ogg"}
	updateFromCmdLine(cfgDefaults)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.val != config[tt.key] {
				t.Errorf("cfg[%s] == %s; expected %s", tt.key, config[tt.key], tt.val)
			}
		})
	}

	os.Args = make([]string, len(tmp))
	copy(os.Args, tmp)
}

func TestPronCheck(t *testing.T) {
	cfg := make(Config)
	cfg["VERBOSE"] = "no"
	cfg["LANG"] = "en"
	getHTML = getTestURL

	t.Run("Pronunciation found", func(t *testing.T) {
		if pronCheck(cfg, "test") == false {
			t.Errorf("Pronunciation for word `test` is not found")
		}
	})
	t.Run("Pronunciation does not found", func(t *testing.T) {
		if pronCheck(cfg, "tafel") == true {
			t.Errorf("Pronunciation for word `tafel` should not be found")
		}
	})
}

func TestGetPronList(t *testing.T) {
	tests := []struct {
		name string
		pron []Pron
	}{
		{
			name: "Word test",
			pron: []Pron{
				Pron{
					word:       "test",
					author:     "Author1",
					sex:        "male",
					country:    "United Kingdom",
					mp3:        audioURL + "/mp3/test.mp3",
					ogg:        audioURL + "/ogg/test.ogg",
					aFile:      "test.mp3",
					aURL:       audioURL + "/mp3/test.mp3",
					fullAuthor: "Author1 (male from United Kingdom)",
					cacheDir:   "mp3/en/09",
					cacheFile:  "mp3/en/09/test_Author1.mp3",
				},
				Pron{
					word:       "test",
					author:     "Author2",
					sex:        "male",
					country:    "Unknown",
					mp3:        "https://audio00.forvo.com/audios/mp3/test.mp3",
					ogg:        "https://audio00.forvo.com/audios/ogg/test.ogg",
					aFile:      "test.mp3",
					aURL:       "https://audio00.forvo.com/audios/mp3/test.mp3",
					fullAuthor: "Author2 (male from Unknown)",
					cacheDir:   "mp3/en/09",
					cacheFile:  "mp3/en/09/test_Author2.mp3",
				},
				Pron{
					word:       "test",
					author:     "Author3",
					sex:        "male",
					country:    "USA",
					mp3:        "https://audio00.forvo.com/audios/mp3/test.mp3",
					ogg:        "https://audio00.forvo.com/audios/ogg/test.ogg",
					aFile:      "test.mp3",
					aURL:       "https://audio00.forvo.com/audios/mp3/test.mp3",
					fullAuthor: "Author3 (male from USA)",
					cacheDir:   "mp3/en/09",
					cacheFile:  "mp3/en/09/test_Author3.mp3",
				},
			},
		},
	}

	cfg := make(Config)
	cfg["VERBOSE"] = "no"
	cfg["LANG"] = "en"
	cfg["ATYPE"] = "mp3"
	cfg["PRONUNCIATION_CHECK"] = "no"
	getHTML = getTestURL

	list := getPronList(cfg, "test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, pron := range tt.pron {
				pronCompare(t, i, pron, list[i])
			}
		})
	}
}

func pronCompare(t *testing.T, i int, want, got Pron) {
	if want.word != got.word {
		t.Errorf("list[%d].word == '%s'; expected '%s'", i, got.word, want.word)
	}
	if want.author != got.author {
		t.Errorf("list[%d].author == '%s'; expected '%s'", i, got.author, want.author)
	}
	if want.sex != got.sex {
		t.Errorf("list[%d].sex == '%s'; expected '%s'", i, got.sex, want.sex)
	}
	if want.country != got.country {
		t.Errorf("list[%d].country == '%s'; expected '%s'", i, got.country, want.country)
	}
	if want.mp3 != got.mp3 {
		t.Errorf("list[%d].mp3 == '%s'; expected '%s'", i, got.mp3, want.mp3)
	}
	if want.ogg != got.ogg {
		t.Errorf("list[%d].ogg == '%s'; expected '%s'", i, got.ogg, want.ogg)
	}
	if want.aFile != got.aFile {
		t.Errorf("list[%d].aFile == '%s'; expected '%s'", i, got.aFile, want.aFile)
	}
	if want.aURL != got.aURL {
		t.Errorf("list[%d].aURL == '%s'; expected '%s'", i, got.aURL, want.aURL)
	}
	if want.fullAuthor != got.fullAuthor {
		t.Errorf("list[%d].fullAuthor == '%s'; expected '%s'", i, got.fullAuthor, want.fullAuthor)
	}
	if want.cacheDir != got.cacheDir {
		t.Errorf("list[%d].cacheDir == '%s'; expected '%s'", i, got.cacheDir, want.cacheDir)
	}
	if want.cacheFile != got.cacheFile {
		t.Errorf("list[%d].cacheFile == '%s'; expected '%s'", i, got.cacheFile, want.cacheFile)
	}
}

func TestSaveWord(t *testing.T) {
	cfg := make(Config)
	cfg["VERBOSE"] = "no"
	cfg["CACHE"] = "no"
	cfg["DOWNLOAD"] = "yes"
	cfg["INTERACTIVE"] = "no"
	cfg["LANG"] = "en"
	cfg["ATYPE"] = "mp3"
	cfg["PRONUNCIATION_CHECK"] = "no"
	getHTML = getTestURL
	getAudio = downloadTestFile
	tmpDir := t.TempDir()

	wantFile, err := os.Open("local_files/forvo_en_test.mp3")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer wantFile.Close()
	text, err := io.ReadAll(wantFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	wantMD5 := md5.Sum(text)

	list := getPronList(cfg, "test")
	list[0].aFile = filepath.Join(tmpDir, "test.mp3")
	saveWord(cfg, list[0])

	gotFile, err := os.Open(list[0].aFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer gotFile.Close()
	text, err = io.ReadAll(gotFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	gotMD5 := md5.Sum(text)

	for i := 0; i < len(wantMD5); i++ {
		if wantMD5[i] != gotMD5[i] {
			t.Errorf("md5sum local_files/forvo_en_test.mp3 and %s are not equal", list[0].aFile)
			return
		}
	}
}
