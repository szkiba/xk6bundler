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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	os.Exit(run(os.Args))
}

var version = "dev"

func run(args []string) int {
	opts, err := parseOptions(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())

		return 1
	}

	if opts.About {
		fmt.Fprintf(os.Stderr, "%s/%s %s/%s\n", app, version, runtime.GOOS, runtime.GOARCH)

		return 0
	}

	for _, platform := range opts.platforms {
		if err := build(platform, opts); err != nil {
			fmt.Fprintln(os.Stderr, err)

			return 1
		}
	}

	if !isGitHubAction() {
		return 0
	}

	if err := outGitHubAction(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	return 0
}

func outGitHubAction(opts *options) error {
	const format = "::set-output name=%s::%s\n"

	fmt.Fprintf(os.Stdout, format, "name", opts.Name)
	fmt.Fprintf(os.Stdout, format, "version", opts.Version)

	v := &vars{
		Name:    opts.Name,
		Version: opts.Version,
		Os:      "linux",
		Arch:    "amd64",
		Ext:     "",
	}

	out, err := expandTemplate("output", opts.Output, v)
	if err != nil {
		return err
	}

	dir := filepath.Dir(out)

	fmt.Fprintf(os.Stdout, format, "dockerdir", dir)
	fmt.Fprintf(os.Stdout, format, "dockerfile", filepath.Join(dir, "Dockerfile"))

	return nil
}
