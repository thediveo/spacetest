/*
Package api defines the specific protocol requests and responses used between
clients and servers of the “spacer” namespace creation service. These protocol
elements are exchanged using the [gob] encoding/decoding scheme.

The api package automatically registers the individual protocol element types so
that they can be especially used in receiving (polymorphous) interface values.
*/
package api
