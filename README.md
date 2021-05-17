# xk6bundler

xk6bundler is a CLI tool and [GitHub Action](https://docs.github.com/en/actions) makes bundle k6 with extensions as fast and easily as possible.

**Features**

- Build for multiple target platforms
- Create per platform `.tar.gz` archives for releases
- Generate `Dockerfile` for Docker build/push
- Guess reasonable default values
- Almost drop-in [xk6](https://github.com/k6io/xk6) replacement
- Only one GitHub workflow file required to build and publish custom k6 bundle

For a real life example check [k6-crocus](https://github.com/szkiba/k6-crocus) and it's [.github/workflows/xk6bundler.yml](https://github.com/szkiba/k6-crocus/blob/master/.github/workflows/xk6bundler.yml) file

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Parameters](#parameters)
  - [name](#name)
  - [version](#version)
  - [with](#with)
  - [platform](#platform)
  - [output](#output)
  - [archive](#archive)
  - [k6_repo](#k6_repo)
  - [k6_version](#k6_version)
- [GitHub Action](#github-action)
  - [Outputs](#outputs)
  - [GitHub release](#github-release)
  - [Docker Hub push](#docker-hub-push)
- [CLI](#cli)
  - [Install the pre-compiled binary](#install-the-pre-compiled-binary)
  - [Install with Go](#install-with-go)
  - [Running with Docker](#running-with-docker)
  - [Verifying your installation](#verifying-your-installation)
  - [Usage](#usage)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Parameters

The CLI tool and GitHub Action has same parameters. The main difference is how to pass the parameters. Another difference is that using CLI tool you may specify `with` and `platform` parameters multiple times, while using GitHub action you should specify these parameters only once, but you may use a whitespace separated list of values.

The following table show parameter names, CLI option flags and environment variable names for the default value of a given parameter:

Parameter                   | CLI                     | Environment
----------------------------|-------------------------|----------------------
[`name`](#name)             | `-n, --name=name`       | `XK6BUNDLER_NAME`
[`version`](#version)       | `-v, --version=version` | `XK6BUNDLER_VERSION`
[`with`](#with)             | `-w, --with=extension`  | `XK6BUNDLER_WITH`
[`platform`](#platform)     | `-p, --platform=target` | `XK6BUNDLER_PLATFORM`
[`output`](#output)         | `-o, --output=path`     | `XK6BUNDLER_OUTPUT`
[`archive`](#archive)       | `-a, --archive=path`    | `XK6BUNDLER_ARCHIVE`
[`k6_repo`](#k6_repo)       | `--k6-repo=repo`        | `XK6BUNDLER_K6_REPO`
[`k6_version`](#k6_version) | `--k6-version=version`  | `XK6BUNDLER_K6_VERSION`

### name

Short name of the bundle. Optional, if missing then xk6bunder will try to guess from git remote or from current directory name.

### version

Bundle version. Optional, if missing then xk6bundler will try to guess from `GITHUB_REF` or default to `SNAPSHOT`.

### with

xk6 extension to add in `module[@version][=replacement]` format. When using CLI, it can be used multiple times to add extensions by specifying the Go module name and optionally its version, similar to go get. Module name is required, but specific version and/or local replacement are optional. Replacement path must be absolute. When using GitHub Action, it can contains whilespace separated list of modules. Optional, if missing then no xk6 extension will be bundled.

### platform

Target platform in `os/arch` format. When using CLI, it can be used multiple times to add target platform. When using GitHub Action, it can contains whilespace separated list of target platforms. Optinal, default value is `linux/amd64 windows/amd64 darwin/amd64`

### output

[Go template](https://golang.org/pkg/text/template/) of output file path. Optional, default value is `dist/{{.Name}}_{{.Os}}_{{.Arch}}/k6{{.Ext}}`

The following template variables available in template:

Variable  | Descripion
----------|-----------
`Os`      | OS name (values defined by `GOOS`)
`Arch`    | hardware architecture (values defined by `GOARCH`)
`Ext`     | `.exe` on windows empty otherwise
`Name`    | bundle name
`Version` | bundle version

You can use [slim-sprig](https://go-task.github.io/slim-sprig/) template function library as well.

### archive

[Go template](https://golang.org/pkg/text/template/) of archive (.tar.gz) file path. Optional, default value is `dist/{{.Name}}_{{.Version}}_{{.Os}}_{{.Arch}}.tar.gz`

The following template variables available in template:

Variable  | Descripion
----------|-----------
`Os`      | OS name (values defined by `GOOS`)
`Arch`    | hardware architecture (values defined by `GOARCH`)
`Ext`     | `.exe` on windows empty otherwise
`Name`    | bundle name
`Version` | bundle version

You can use [slim-sprig](https://go-task.github.io/slim-sprig/) template function library as well.

### k6_repo

Build using a k6 fork repository. The repo can be a remote repository or local directory path.

### k6_version

The core k6 version to build. Optional, if missing then `latest` will be used.

## GitHub Action

For using xk6bundler as [GitHub Action](https://docs.github.com/en/actions), you should include a workflow step with `uses: szkiba/xk6bundler@v0`

```yaml
- name: Build
  id: build
  uses: szkiba/xk6bundler@v0
  with:
    platform: linux/amd64 windows/amd64
    with: |
      github.com/szkiba/xk6-prometheus@v0.1.2
      github.com/szkiba/xk6-jose@v0.1.1
      github.com/szkiba/xk6-ansible-vault@v0.1.1
```

### Outputs

The xk6bundler GitHub Action outputs the following variables:

#### name

Short name of the bundle.

#### version

Bundle version.

#### dockerfile

Generated Dockerfile path.

#### dockerdir

Docker context directory path. Can be use as `context` parameter for Docker build action (assuming `build` is the id of the xk6bundler step):

```yaml
- name: Docker build and push
  uses: docker/build-push-action@v2
  with:
    context: ./${{ steps.build.outputs.dockerdir }}
```

> The `./` prefix required, it will tell to Docker action that this is a local path (not an URL).

See [samples/sample-dockerhub-workflow.yml](samples/sample-dockerhub-workflow.yml) for complete example.

### GitHub release

The xk6bundler GitHub Action generates result by default in `dist` directory of the current workspace. You can publish generated archive (.tar.gz) files using any thirdpary release GitHub Action.

Sample workflow file for publishing xk6 bundle on GitHub releases page: [samples/sample-workflow.yml](samples/sample-workflow.yml). Put this file in to `.github/workflows` directory.

### Docker Hub push

If target platforms include `linux/amd64` then xk6bundler will generate `Dockerfile` next to `linux/amd64` platform's output file. You can use it to build Docker image and push it for example to Docker Hub registry (or to any other).

Sample workflow file for publishing xk6 bundle on GitHub releases page and on Docker Hub registry: [samples/sample-dockerhub-workflow.yml](samples/sample-dockerhub-workflow.yml). Put this file in to `.github/workflows` directory.


## CLI

You can install the pre-compiled binary or use Docker.

### Install the pre-compiled binary

Download the pre-compiled binaries from the [releases page](https://github.com/szkiba/xk6bundler/releases) and copy to the desired location.

### Install with Go

If you have Go environment set up, you can build xk6bundler from source by running:

```sh
go get github.com/szkiba/xk6bundler/cmd/xk6bundler
```

Binary would be installed to $GOPATH/bin/xk6bundler.

### Running with Docker

You can also use it within a Docker container. To do that, you'll need to
execute the following:

```bash
docker run szkiba/xk6bundler
```

### Verifying your installation

To verify your installation, use the `xk6bundler -V` command:

```bash
$ xk6bundler -V

xk6bundler/0.1.0 linux/amd64
```

You should see `xk6bundler/VERSION` in the output.

### Usage

To print usage information, use the `xk6bundler --help` command.
