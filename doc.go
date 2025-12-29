/*
Package spacetest supports tests working with Linux kernel namespaces.

This package leverages the [Ginkgo] testing framework with [Gomega] matchers.

The top-level “spacetest” package is mostly agnostic to the particular type of
namespace handled, but different test helper functions come with different
restrictions due to restrictions imposed by the Linux kernel. Please carefully
check the documentation for the individual helper functions.

# Network Namespaces

The spacetest/netns package is basically just a convenience wrapper around the
generic test helper functions from the base spacetest package. Yet, this package
helps DRY, especially avoiding litanies of [unix.CLONE_NEWNET].

# Mount Namespaces

In some situations, mount namespaces need special treatment which the dedicated
spacetest/mntns package takes care of: in particular, the test helper functions
in this package ensure to remount the “/” in the new mount namespace with private
mount point propagation to avoid mishaps (trust us, we /know/).

Other than that, this package helps again with DRY, such as [unix.CLONE_NEWNS]
litanies.

# PID and User Namespaces

Please note that user and PID namespaces are notoriously difficult to work with,
especially in multi-threaded Go tests. Thus, the spacetest package has somewhat
limited support for dealing with user namespaces. Please refer to the
[github.com/thediveo/spacetest/spacer] package for details.

# Background

The origins of this module lie in [thediveo/notwork]: in order to reuse the
namespace-specific elements without the need to pull in dependencies related
solely to “notwork” testing, “spacetest” was born. And in the tradition of short
and preably misleading idiomatic Go package names, the name “spacetest” was
chosen. Because “namespacetest” would have been much too precise. We promise
that our module won't explode in your face.

[Ginkgo]: https://github.com/onsi/ginkgo
[Gomega]: https://github.com/onsi/gomega
[thediveo/notwork]: https://github.com/thediveo/notwork
*/
package spacetest
