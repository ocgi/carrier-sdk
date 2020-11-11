// Copyright 2021 The OCGI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sdkserver

import (
	netcontext "golang.org/x/net/context"
	"google.golang.org/grpc/metadata"

	sdkapi "github.com/ocgi/carrier-sdk/sdks/sdkgo/api/v1alpha1"
)

type gameServerMockStream struct {
	msgs chan *sdkapi.GameServer
}

// newGameServerMockStream implements SDK_WatchGameServerServer for testing
func newGameServerMockStream() *gameServerMockStream {
	return &gameServerMockStream{
		msgs: make(chan *sdkapi.GameServer, 10),
	}
}

func (m *gameServerMockStream) Send(gs *sdkapi.GameServer) error {
	m.msgs <- gs
	return nil
}

func (*gameServerMockStream) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (*gameServerMockStream) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (*gameServerMockStream) SetTrailer(metadata.MD) {
	panic("implement me")
}

func (*gameServerMockStream) Context() netcontext.Context {
	panic("implement me")
}

func (*gameServerMockStream) SendMsg(m interface{}) error {
	panic("implement me")
}

func (*gameServerMockStream) RecvMsg(m interface{}) error {
	panic("implement me")
}