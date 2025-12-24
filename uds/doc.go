/*
Package uds supports transferring open file descriptors across process
boundaries using peer-to-peer pairs of datagram unix domain sockets.

Using datagram unix domain sockets has the benefit of implicit message
boundaries as long as messages are rather moderate in size.

# Trivia

“UDS” is short for “unix domain socket”.
*/
package uds
