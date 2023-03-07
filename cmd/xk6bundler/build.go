// MIT License
//
// Copyright (c) 2021 Iván Szkiba
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	sprig "github.com/go-task/slim-sprig"
	"go.k6.io/xk6"
)

//go:embed Dockerfile
var dockerTemplate string

func build(platform xk6.Platform, opts *options) error {
	data := &vars{
		Name:    opts.Name,
		Version: opts.Version,
		Os:      platform.OS,
		Arch:    platform.Arch,
		Ext:     "",
	}

	if platform.OS == "windows" {
		data.Ext = ".exe"
	}

	out, err := expandTemplate("output", opts.Output, data)
	if err != nil {
		return err
	}

	archive, err := expandTemplate("archive", opts.Archive, data)
	if err != nil {
		return err
	}

	const dirMode = 0o755

	dir := filepath.Dir(out)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}

	builder := xk6.Builder{ //nolint:exhaustruct
		Compile: xk6.Compile{
			Platform: platform,
			Cgo:      false,
		},
		Extensions:   opts.extensions,
		Replacements: opts.replacements,
		K6Repo:       opts.K6repo,
		K6Version:    opts.K6version,
	}

	if err := builder.Build(context.Background(), out); err != nil {
		return err
	}

	if platform.OS == "linux" && platform.Arch == "amd64" {
		if err := createDockerfile(out); err != nil {
			return err
		}
	}

	return createArchive(archive, out)
}

func createDockerfile(output string) error {
	dir := filepath.Dir(output)

	str, err := expandTemplate("Dockerfile", dockerTemplate, map[string]string{"Output": filepath.Base(output)})
	if err != nil {
		return err
	}

	name := filepath.Join(dir, "Dockerfile")

	const fileMode = 0o644

	return ioutil.WriteFile(name, []byte(str), fileMode)
}

func createArchive(archive string, output string) error {
	tgz, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer tgz.Close()

	gzipWriter := gzip.NewWriter(tgz)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	if err := addToArchive(tarWriter, output, false); err != nil {
		return err
	}

	if err := addToArchive(tarWriter, "LICENSE", true); err != nil {
		return err
	}

	return addToArchive(tarWriter, "README.md", true)
}

func addToArchive(archive *tar.Writer, path string, optional bool) error {
	file, err := os.Open(path)
	if err != nil {
		if optional {
			return nil
		}

		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	name := filepath.Base(path)

	header := &tar.Header{ //nolint:exhaustruct
		Name:    name,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = archive.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(archive, file)
	if err != nil {
		return err
	}

	return nil
}

type vars struct {
	Os      string
	Arch    string
	Ext     string
	Version string
	Name    string
}

func expandTemplate(name string, tmplsrc string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Funcs(sprig.TxtFuncMap()).Parse(tmplsrc)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer

	if err := tmpl.Execute(&buff, data); err != nil {
		return "", err
	}

	return buff.String(), nil
}
