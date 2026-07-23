// Copyright 2022 The Goploy Authors. All rights reserved.
// Use of this source code is governed by a GPLv3-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

var (
	AssetDir string
)

func GetAssetDir() string {
	if AssetDir != "" {
		return AssetDir
	}
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		panic(err)
	}
	app, err := filepath.Abs(file)
	if err != nil {
		panic(err)
	}
	AssetDir = filepath.Dir(app)
	return AssetDir
}

func GetConfigFile() string {
	return filepath.Join(GetAssetDir(), "goploy.toml")
}

func GetPidFile() string {
	return filepath.Join(GetAssetDir(), "goploy.pid")
}

func GetRepositoryPath() string {
	if Toml.APP.RepositoryPath != "" {
		return filepath.Join(Toml.APP.RepositoryPath, "repository")
	}
	return filepath.Join(GetAssetDir(), "repository")
}

func GetProjectFilePath(projectID int64) string {
	return filepath.Join(GetRepositoryPath(), "project-file", "project_"+strconv.FormatInt(projectID, 10))
}

func GetProjectPath(projectID int64) string {
	return filepath.Join(GetRepositoryPath(), "project_"+strconv.FormatInt(projectID, 10))
}

func GetTerminalLogPath(tlID int64) string {
	return filepath.Join(GetRepositoryPath(), "terminal-log", strconv.FormatInt(tlID, 10)+".cast")
}
