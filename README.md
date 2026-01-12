# `spacetest`

[![PkgGoDev](https://pkg.go.dev/badge/github.com/thediveo/spacetest)](https://pkg.go.dev/github.com/thediveo/spacetest)
[![GitHub](https://img.shields.io/github/license/thediveo/spacetest)](https://img.shields.io/github/license/thediveo/spacetest)
![build and test](https://github.com/thediveo/spacetest/actions/workflows/buildandtest.yaml/badge.svg?branch=master)
[![goroutines](https://img.shields.io/badge/go%20routines-not%20leaking-success)](https://pkg.go.dev/github.com/onsi/gomega/gleak)
[![file descriptors](https://img.shields.io/badge/file%20descriptors-not%20leaking-success)](https://pkg.go.dev/github.com/thediveo/fdooze)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/spacetest)](https://goreportcard.com/report/github.com/thediveo/spacetest)
![Coverage](https://img.shields.io/badge/Coverage-88.8%25-brightgreen)

A small package to help with creating transient Linux namespaces in unit
testing, without having to deal with the tedious details of proper and robust
setup and teardown – and without destroying your host filesystem. It even allows
Go tests to create child user and PID namespaces (with the help of forking child
processes into new user and PID namespaces).

`spacetest` leverages the [Ginkgo](https://github.com/onsi/ginkgo) testing
framework with [Gomega](https://github.com/onsi/gomega) matchers.

The origins of this module lie in
[thediveo/notwork](https://github.com/thediveo/notwork): in order to reuse the
namespace-specific elements without the need to pull in dependencies related
solely to "notwork" testing, `spacetest` was born. And in the tradition of short
and preably misleading idiomatic Go package names, the name `spacetest` was
chosen. Because "namespacetest" would have been much too precise. We promise
that our module won't explode in your face.

## Example

Creating and entering a transient network namespace for testing purposes become
as easy and concise as:

```go
import (
    "github.com/thediveo/spacetest/netns"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("...", func() {

    It("tests something inside a transient network namespace", func() {
        defer netns.EnterTransient()() // !!! double ()()
        // ...
    })

})
```

Trust us, you don't want to repeat over and over again as well as seeing all the
time the individual step sequence and many sanity checks inside
`EnterTransient()`. There's a limit to inflicting full code complexity on
developers, and good reason for appropriate abstractions.

## Using the Spacer

Using what we call the "spacer" gives you access to creating new child (and
grandchild) user and PID namespaces, which isn't directly possible from any
multi-threaded Go application. The spacer leverages exec.Command.Start where the
kernel allows us to start the child process in new namespaces.

But first, if you use spacetest in a devcontainer, make sure that the Go
toolchain is accessible to the devcontainer's root user (it isn't necessarily,
depending on your base image and further features):

```jsonc
{
    // ...
    "postCreateCommand": "echo 'Defaults:vscode secure_path = \"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin\"' | sudo tee -a /etc/sudoers.d/vscode",
    // ...
}
```

Then to create child user and PID child namespaces below your test's user and
PID namespaces:

```go
import (
    "context"

    "github.com/thediveo/spacetest/spacer"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("...", func() {

    It("tests something inside transient user and PID namespaces", func(ctx context.Context) {
        spcclnt := spacer.New(ctx)
		DeferCleanup(func() {
			spcclnt.Close()
		})

		subclnt, subspc := spcclnt.Subspace(true, true)
		DeferCleanup(func() {
			_ = unix.Close(subspc.PID)
			_ = unix.Close(subspc.User)
			subclnt.Close()
		})
        // ...
    })

})
```

You can use the returned sub client to create more user and PID namespaces
further down the hierarchy.

## DevContainer

> [!CAUTION]
>
> Do **not** use VSCode's "~~Dev Containers: Clone Repository in Container
> Volume~~" command, as it is utterly broken by design, ignoring
> `.devcontainer/devcontainer.json`.

1. `git clone https://github.com/thediveo/spacetest`
2. in VSCode: Ctrl+Shift+P, "Dev Containers: Open Workspace in Container..."
3. select `spacetest.code-workspace` and off you go...

## Supported Go Versions

`spacetest` supports versions of Go that are noted by the [Go release
policy](https://golang.org/doc/devel/release.html#policy), that is, major
versions _N_ and _N_-1 (where _N_ is the current major version).

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## Copyright and License

Copyright 2023–25 Harald Albrecht, licensed under the Apache License, Version 2.0.
