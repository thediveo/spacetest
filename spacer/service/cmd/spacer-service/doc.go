/*
Package main provides the command for running a spacer service as a separate
process. This command is not intended to be run directly from the CLI, but
instead only from [spacer.Client] and [service.SpaceMaker].

The command expects the file descriptor number 3 to be open and to be a
connected unix domain socket. This socket is then used to receive service
requests and to send back responses.

The command terminates when the connected peer socket closes (disconnects).
*/
package main
