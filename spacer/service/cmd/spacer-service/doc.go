/*
Package main provides the command for running a spacer service as a separate
process. This command is not intended to be run directly from the CLI, but
instead only from [spacer.Client] and [service.SpaceMaker].

The command expects the file descriptor number 3 to be open and to be a
connected unix domain socket. This socket is then used to receive service
requests and to send back responses.

The command terminates when the connected peer socket closes (disconnects).

When the command runs as PID 1 in a new child PID namespace and if
“/proc/1/cmdline” differs from running this command, then the command assumes
that it has been spawned additionally into a new mount namespace and thus
firstly remounts recursively all existing mount points as “private” before
secondly mount a fresh “procfs” onto “/proc”. This is necessary in order to
allow creating grandchild PID namespaces.
*/
package main
