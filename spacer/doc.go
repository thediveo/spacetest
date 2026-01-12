/*
Package spacer provides a client to create and communicate with so-called
“spacer services” that on demand create new Linux kernel namespaces, especially
sub user and PID namespaces.

Using spacer clients (and the spacer services they are automatically connected
to) works around the restrictions Linux imposes multi-threaded processes such as
Go programs, where no new sub user namespaces can be created after the “before
the Go runtime phase”. Also, the spacer design works around the general
restriction where neither process nor thread/task can switch themselves into a
child PID namespace (they can only set the child PID namespace for, well, child
processes).

This design leverages individual spacer service (child) processes that then
enables creating the hierarchical user and PID namespaces, as these cannot be
created directly (via [unshare(2)]) from a multi-threaded process.

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
