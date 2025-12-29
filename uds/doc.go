/*
Package uds supports transferring open file descriptors across process
boundaries using peer-to-peer pairs of (stream) unix domain sockets.

Using stream unix domain sockets has the benefit of being able to detect when
the “other” side has disconnected.

# Trivia

“[UDS]” is short for “unix domain socket”.

[UDS]: https://en.wikipedia.org/wiki/Unix_domain_socket
*/
package uds
