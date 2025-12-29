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
	"context"
	"log/slog"
	"time"

	"github.com/thediveo/spacetest/spacer/api"
	"github.com/thediveo/spacetest/uds"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
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

	It("runs the service until cancelled", func(ctx context.Context) {
		dupond, dupont := Successful2R(uds.NewPair())
		defer func() {
			_ = dupond.Close()
			_ = dupont.Close()
		}()

		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			Serve(ctx, dupont, &Spacemaker{Exe: "/not-existing"})
		}()

		Eventually(done).Within(5 * time.Second).Should(BeClosed())
	})

	It("terminates the service when the client disconnects", func(ctx context.Context) {
		dupond, dupont := Successful2R(uds.NewPair())
		defer func() {
			_ = dupond.Close()
			_ = dupont.Close()
		}()

		armed := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			close(armed)
			Serve(ctx, dupont, &Spacemaker{Exe: "/not-existing"})
		}()

		Eventually(armed).Within(1 * time.Second).Should(BeClosed())
		time.Sleep(1 * time.Second)
		Expect(dupond.Close()).To(Succeed())
		Eventually(done).Within(5 * time.Second).Should(BeClosed())
	})

})

type closingmock struct{ conn *uds.Conn }

var _ Spacer = (*closingmock)(nil)

func (m *closingmock) Room(*api.RoomsRequest) api.Response {
	_ = m.conn.Close()
	return &api.ErrorResponse{Reason: "not mocked"}
}

func (m *closingmock) Subspace(req *api.SubspaceRequest) api.Response {
	_ = m.conn.Close()
	return &api.ErrorResponse{Reason: "not mocked"}
}
