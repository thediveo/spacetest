// Copyright 2025 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/thediveo/spacetest/spacer/service"
	"github.com/thediveo/spacetest/uds"
	"golang.org/x/sys/unix"

	"github.com/thediveo/spacetest/spacer"
)

var _ = spacer.New // ... so that [spacer.Client] gets a proper hyperlink.

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))

	slog.Info("spacetest/spacer/service/cmd/spacer-service started",
		slog.Int("pid", os.Getpid()))
	defer slog.Info("spacetest/spacer/service/cmd/spacer-service terminated",
		slog.Int("pid", os.Getpid()))

	// If we get don't get started in a new child PID namespace, we can't be
	// ever PID1 -- at least not with someone actively trying to fool us. But if
	// we're PID1 (sorry, süsstehm-deh) then the currently mounted /proc will
	// bite us when later spawning a new sub service and trying to get its
	// namespaces via /proc. Thus, after making sure that we are in charge, we
	// expect to be also in a new mount namespace and thus remount all mounts
	// recursively as private. Finally, we mount a new procfs onto /proc.
	//
	// To illustrate: "unshare -Upf ps" will show you all(!) your processes,
	// where you should only see your PID1.
	//
	// In contrast, "unshare -Upfm --mount-proc=/proc ps" shows you only your
	// PID.
	//
	// For related background information, especially the infamous "fork: cannot
	// allocate memory", see
	// https://linuxvox.com/blog/unshare-pid-bin-bash-fork-cannot-allocate-memory
	if os.Getpid() == 1 {
		slog.Info("I've got the power of 1!")
		cmdline, err := os.ReadFile("/proc/1/cmdline")
		if err == nil && string(cmdline) != os.Args[0]+"\x00" {
			err = unix.Mount("none", "/", "/", unix.MS_REC|unix.MS_PRIVATE, "")
			if err == nil {
				err = unix.Mount("none", "/proc", "proc",
					unix.MS_NODEV|unix.MS_NOEXEC|unix.MS_NOSUID|unix.MS_RELATIME,
					"")
			}
		}
		if err != nil {
			slog.Error("cannot remount /proc", slog.String("err", err.Error()))
		} else {
			slog.Info("remounted /proc")
		}
	}

	dupont, err := uds.NewUnixConn(3, "dupont")
	if err != nil {
		slog.Error("invalid fd 3", slog.String("err", err.Error()))
		os.Exit(1)
	}
	service.Serve(context.Background(), dupont, &service.Spacemaker{})
}
