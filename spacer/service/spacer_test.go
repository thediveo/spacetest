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

package service

import (
	"log/slog"
	"os"
	"time"

	"github.com/thediveo/spacetest/spacer/api"
	"golang.org/x/sys/unix"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
)

var _ = Describe("serving space", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		goodgos := Goroutines()
		DeferCleanup(func() {
			Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(100 * time.Millisecond).
				ShouldNot(HaveLeaked(goodgos))
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})

		oldDefault := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(GinkgoWriter, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
		DeferCleanup(func() { slog.SetDefault(oldDefault) })
	})

	Context("Subspace service", func() {

		It("rejects invalid params", func() {
			sm := &Spacemaker{}
			Expect(sm.Subspace(&api.SubspaceRequest{})).To(api.HaveFailed())
			Expect(sm.Subspace(&api.SubspaceRequest{Spaces: ^uint64(0)})).To(api.HaveFailed())
		})

		It("fails when unable to start new subspace service", func() {
			sm := &Spacemaker{Exe: "/not-existing"}
			Expect(sm.Subspace(&api.SubspaceRequest{Spaces: unix.CLONE_NEWUSER})).To(api.HaveFailed())
		})

	})

	Context("Room service", func() {

		It("rejects invalid params", func() {
			sm := &Spacemaker{}
			Expect(sm.Room(&api.RoomsRequest{})).To(api.HaveFailed())
			Expect(sm.Room(&api.RoomsRequest{Spaces: ^uint64(0)})).To(api.HaveFailed())
		})

		It("reports when powerless", func() {
			if os.Getuid() == 0 {
				Skip("no root")
			}
			sm := &Spacemaker{}
			Expect(sm.Room(&api.RoomsRequest{
				Spaces: unix.CLONE_NEWNET | unix.CLONE_NEWUTS,
			})).To(api.HaveFailed())
		})

	})

	Context("newNamespace", func() {

		It("reports failure when not able to determine current namespace", func() {
			if os.Getuid() == 0 {
				Skip("root")
			}
			Expect(newNamespace(0)).Error().To(HaveOccurred())
		})

		It("reports failure when not able to create new namespace", func() {
			if os.Getuid() == 0 {
				Skip("root")
			}
			Expect(newNamespace(unix.CLONE_NEWNET)).Error().To(HaveOccurred())
		})

	})

})
