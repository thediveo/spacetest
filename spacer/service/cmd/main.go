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
)

func main() {
	slog.Info("spacetest/spacer/service/cmd started")
	defer slog.Info("spacetest/spacer/service/cmd terminated")

	dupont, err := uds.NewUnixConn(3, "dupont")
	if err != nil {
		slog.Error("invalid fd 3", slog.String("err", err.Error()))
		os.Exit(1)
	}
	var statinfo unix.Stat_t
	if err := unix.Fstat(3, &statinfo); err != nil {
		slog.Error("cannot stat inherited unix domain socket", slog.String("err", err.Error()))
	} else {
		slog.Info("unix domain socket", slog.Int64("ino", int64(statinfo.Ino)))
	}
	service.Serve(context.Background(), dupont, &service.Spacemaker{})
}
