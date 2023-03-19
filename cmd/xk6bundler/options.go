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
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jessevdk/go-flags"
	"gitlab.com/golang-commonmark/markdown"
	"go.k6.io/xk6"
	"gopkg.in/ini.v1"
)

const (
	app  = "xk6bundler"
	desc = `Bundle k6 with extensions as fast and easily as possible`
)

var (
	ErrInvalidPlatform = errors.New("invalid platform")
	ErrMissingModule   = errors.New("module name is required")
	ErrMissingName     = errors.New("couldn't guess bundle name, please specify it")
)

type options struct {
	About     bool     `short:"V" description:"Show version information"`
	Name      string   `short:"n" long:"name" value-name:"name" env:"XK6BUNDLER_NAME" description:"Short name of the bundle."`                                                                                                                                                                                                                                                                                                         //nolint:lll
	Version   string   `short:"v" long:"version" value-name:"version" env:"XK6BUNDLER_VERSION" default:"SNAPSHOT" description:"Bundle version."`                                                                                                                                                                                                                                                                                       //nolint:lll
	With      []string `short:"w" long:"with" value-name:"extension" env:"XK6BUNDLER_WITH" env-delim:"," description:"Add extension in 'module[@version][=replacement]' format. Can be used multiple times to add extensions by specifying the Go module name and optionally its version, similar to go get. Module name is required, but specific version and/or local replacement are optional. Replacement path must be absolute."` //nolint:lll
	Markdown  string   `short:"m" long:"markdown" value-name:"markdown" env:"XK6BUNDLER_MARKDOWN" description:"Extract extension list from Markdown code blocks. Code block language should be 'xk6' and contains extension list in format 'module[@version][=replacement]'."`                                                                                                                                                         //nolint:lll
	Platform  []string `short:"p" long:"platform" value-name:"target" env:"XK6BUNDLER_PLATFORM" default:"linux/amd64" default:"windows/amd64" default:"darwin/amd64" description:"Add target platform in 'os/arch' format. Can be used multiple times to add target platform."`                                                                                                                                                        //nolint:lll
	Output    string   `short:"o" long:"output" value-name:"path" env:"XK6BUNDLER_OUTPUT" default:"dist/{{.Name}}_{{.Os}}_{{.Arch}}/k6{{.Ext}}" description:"Output file path template."`                                                                                                                                                                                                                                              //nolint:lll
	Archive   string   `short:"a" long:"archive" value-name:"path" env:"XK6BUNDLER_ARCHIVE" default:"dist/{{.Name}}_{{.Version}}_{{.Os}}_{{.Arch}}.tar.gz" description:"Archive (.tar.gz) file path template."`                                                                                                                                                                                                                        //nolint:lll
	K6repo    string   `long:"k6-repo" value-name:"repo" env:"XK6BUNDLER_K6_REPO" description:"Build using a k6 fork repository. The repo can be a remote repository or local directory path."`                                                                                                                                                                                                                                        //nolint:lll
	K6version string   `long:"k6-version" value-name:"version" env:"XK6BUNDLER_K6_VERSION" default:"latest" description:"The core k6 version to build."`                                                                                                                                                                                                                                                                               //nolint:lll

	extensions   []xk6.Dependency
	replacements []xk6.Replace
	platforms    []xk6.Platform
}

func parseOptions(args []string) (*options, error) {
	opts := new(options)

	parser, err := newParser(opts)
	if err != nil {
		return nil, err
	}

	if _, err := parser.ParseArgs(args); err != nil {
		return nil, err
	}

	if isGitHubAction() {
		fixGitHubAction(opts)
	}

	if err := extractMarkdown(opts); err != nil {
		return nil, err
	}

	if err := parseWith(opts); err != nil {
		return nil, err
	}

	if err := parsePlatform(opts); err != nil {
		return nil, err
	}

	if opts.Name == "" {
		return nil, ErrMissingName
	}

	return opts, nil
}

func newParser(opts *options) (*flags.Parser, error) {
	parser := flags.NewNamedParser(app, flags.HelpFlag)
	parser.Usage = "[options]"
	parser.Command.Group.LongDescription = desc

	group, err := parser.AddGroup("Options", "", opts)
	if err != nil {
		return nil, err
	}

	if name := guessName(); name != "" {
		oname := parser.Command.Group.FindOptionByLongName("name")
		oname.Default = []string{name}
	}

	if isGitHubAction() {
		prepareGitHubAction(group)
	}

	return parser, nil
}

func prepareGitHubAction(group *flags.Group) {
	for _, o := range group.Options() {
		if o.EnvDefaultKey == "" {
			continue
		}

		o.EnvDefaultKey = strings.ReplaceAll(o.EnvDefaultKey, "XK6BUNDLER", "INPUT")
	}

	if ref := os.Getenv("GITHUB_REF"); ref != "" {
		const fields = 3
		if parts := strings.SplitN(ref, "/", fields); len(parts) == fields {
			os.Setenv("INPUT_VERSION", parts[fields-1])
		}
	}

	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		const fields = 2
		if parts := strings.SplitN(repo, "/", fields); len(parts) == fields {
			if os.Getenv("INPUT_NAME") == "" {
				os.Setenv("INPUT_NAME", parts[fields-1])
			}
		}
	}
}

func isGitHubAction() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

func fixGitHubAction(opts *options) {
	with := make([]string, 0, len(opts.With))

	for _, w := range opts.With {
		with = append(with, strings.Fields(w)...)
	}

	opts.With = with

	platform := make([]string, 0, len(opts.Platform))

	for _, p := range opts.Platform {
		platform = append(platform, strings.Fields(p)...)
	}

	opts.Platform = platform
}

func guessName() string {
	gitcfg, err := ini.Load(filepath.Join(".git", "config"))
	if err != nil {
		dir, err := os.Getwd()
		if err != nil {
			return ""
		}

		return filepath.Base(dir)
	}

	sec, err := gitcfg.GetSection(`remote "origin"`)
	if err != nil {
		return ""
	}

	key, err := sec.GetKey("url")
	if err != nil {
		return ""
	}

	return strings.TrimSuffix(path.Base(key.String()), ".git")
}

func parseWith(opts *options) error {
	opts.extensions = make([]xk6.Dependency, 0, len(opts.With))
	opts.replacements = make([]xk6.Replace, 0)

	for _, w := range opts.With {
		mod, ver, repl, err := splitWith(w)
		if err != nil {
			return err
		}

		mod = strings.TrimSuffix(mod, "/") // easy to accidentally leave a trailing slash if pasting from a URL

		opts.extensions = append(opts.extensions, xk6.Dependency{
			PackagePath: mod,
			Version:     ver,
		})

		if repl != "" {
			if repl == "." {
				repl, err = os.Getwd()
				if err != nil {
					return err
				}
			}

			opts.replacements = append(opts.replacements, xk6.NewReplace(mod, repl))
		}
	}

	return nil
}

func splitWith(arg string) (module, version, replace string, err error) {
	const versionSplit, replaceSplit = "@", "="

	const versionParts, moduleParts = 2, 2

	parts := strings.SplitN(arg, versionSplit, versionParts)
	module = parts[0]

	if len(parts) == 1 {
		parts := strings.SplitN(module, replaceSplit, moduleParts)
		if len(parts) > 1 {
			module = parts[0]
			replace = parts[1]
		}
	} else {
		version = parts[1]
		parts := strings.SplitN(version, replaceSplit, versionParts)
		if len(parts) > 1 {
			version = parts[0]
			replace = parts[1]
		}
	}

	if module == "" {
		err = fmt.Errorf("%w: %s", ErrMissingModule, arg)
	}

	return
}

func splitPlatform(arg string) (os, arch string, err error) {
	const platformSplit = "/"

	const platformParts = 2

	parts := strings.SplitN(arg, platformSplit, platformParts)

	if len(parts) == 1 {
		err = fmt.Errorf("%w: %s", ErrInvalidPlatform, arg)

		return
	}

	os, arch = parts[0], parts[1]

	return
}

func parsePlatform(opts *options) error {
	opts.platforms = make([]xk6.Platform, 0, len(opts.Platform))

	for _, p := range opts.Platform {
		os, arch, err := splitPlatform(p)
		if err != nil {
			return err
		}

		opts.platforms = append(opts.platforms, xk6.Platform{OS: os, Arch: arch, ARM: ""})
	}

	return nil
}

func extractMarkdown(opts *options) error {
	if len(opts.Markdown) == 0 {
		return nil
	}

	data, err := os.ReadFile(opts.Markdown)
	if err != nil {
		return err
	}

	md := markdown.New(markdown.XHTMLOutput(true), markdown.Nofollow(true))
	tokens := md.Parse(data)

	buffer := new(strings.Builder)

	for _, t := range tokens {
		switch t := t.(type) {
		case *markdown.Fence:
			if t.Params == "xk6" {
				if _, err := buffer.WriteString(t.Content); err != nil {
					return err
				}

				if _, err := buffer.WriteRune('\n'); err != nil {
					return err
				}
			}
		default:
		}
	}

	if buffer.Len() > 0 {
		opts.With = append(opts.With, strings.Fields(buffer.String())...)
	}

	return nil
}
