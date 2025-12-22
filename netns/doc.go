/*
Package netns supports running unit tests in separated transient network
namespaces, handling cleanup and error checking automatically.

This package also helps dealing with network namespace identiers in form of
inode numbers. Furthermore it supports executing functions in any other (not
necessarily transient) network namespace.

This package can be combined with [thediveo/notwork], but does not depend on it
(in fact, it has been factored out of the latter to remove the dependency on any
netlink-related packages).

# Usage

The simplest use case is to just call [EnterTransient] and defer its return
value – mind the curse of the double-paired brackets.

	import "github.com/thediveo/spacetest/netns"

	It("tests something inside a transient network namespace", func() {
	  defer netns.EnterTransient()() // !!! double ()()
	  // ...
	})

This example using [EnterTransient] first locks the calling go routine to its
OS-level thread, then creates a new throw-away network namespace, and finally
switches the OS-level thread (and thus its locked go routine) to this new
network namespace.

Deferring the result of [EnterTransient] ensures that when the current test ends
the OS-level thread is switched back to its original network namespace (usually
the host network namespace) and the thread unlocked from the go routine. If
there are no further references alive to the throw-away network namespace, then
the Linux kernel will automatically garbage collect it.

# Advanced Network Namespacing

In more complex scenarios, such as testing with multiple throw-away network
namespaces, multiple network namespaces can be created without automatically
switching into them. In consequence, virtual network interfaces can be created
in these transient network namespaces by either only temporarily switching into
them using [Execute], or by creating a [NewNetlinkHandle] to carry out RTNETLINK
operations on the handle(s) to particular network namespaces.

The following example uses the first method of switching into the first
throw-away network namespace and then creates a VETH pair of network interfaces.
One end is located in the second network interfaces.

	import (
	    "github.com/thediveo/spacetest/netns"
	    "github.com/thediveo/notwork/veth"
	    "github.com/vishvananda/netlink"
	)

	It("tests something inside a temporary network namespace", func() {
	    dupondNetns := netns.NewTransient()
	    dupontNetns := netns.NewTransient()
	    var dupond, dupont netlink.Link
	    netns.Execute(dupondNetns, func() {
	        dupond, dupont = veth.NewTransient(WithPeerNamespace(dupontNetns))
	    })
	})

As for the names of the VETH pair end variables, please refer to [Dupond et
Dupont].

[thediveo/notwork]: https://github.com/thediveo/notwork
[Dupond et Dupont]: https://en.wikipedia.org/wiki/Thomson_and_Thompson
*/
package netns
