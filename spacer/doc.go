/*
Package spacer provides the client to create and communicate with spacer
services in order to create new Linux kernel namespaces. This design leverages
individual spacer service (child) processes that then enables creating the
hierarchical user and PID namespaces, as these cannot be created directly (via
[unshare(2)]) from a multi-threaded process.

Package spacer thus is a “pure Go” alternative to test shell scripts juggling
around with especially the [unshare(1)] command, avoiding juggling and dropping
the many unshare CLI flags, as well as the tedious and brittle passing of
namespace information back into the Go test code.

# Important

Make sure to call [gexec.CleanupBuildArtefacts] in your AfterSuite when using
this package.

[unshare(2)]: https://www.man7.org/linux/man-pages/man2/unshare.2.html
[unshare(1)]: https://www.man7.org/linux/man-pages/man1/unshare.1.html
*/
package spacer

import "github.com/thediveo/spacetest"

var _ = spacetest.NewTransient // make spacetest.xxx true hyperlinks
