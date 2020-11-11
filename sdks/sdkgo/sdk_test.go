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

package sdkgo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	sdk "github.com/ocgi/carrier-sdk/sdks/sdkgo/api/v1alpha1"
)

func TestSDK(t *testing.T) {
	sm := &sdkMock{
		conditions: map[string]string{},
	}
	s := SDK{
		ctx:    context.Background(),
		client: sm,
	}

	err := s.SetCondition("test", "True")
	assert.Nil(t, err)
	assert.Contains(t, sm.conditions, "test")

	gs, err := s.GameServer()
	assert.Nil(t, err)
	assert.NotNil(t, gs)
}

var _ sdk.SDKClient = &sdkMock{}

type sdkMock struct {
	labels      map[string]string
	annotations map[string]string
	conditions  map[string]string
}

func (m *sdkMock) SetLabel(ctx context.Context, in *sdk.KeyValue, opts ...grpc.CallOption) (*sdk.Empty, error) {
	m.labels[in.Key] = in.Value
	return &sdk.Empty{}, nil
}

func (m *sdkMock) SetAnnotation(ctx context.Context, in *sdk.KeyValue, opts ...grpc.CallOption) (*sdk.Empty, error) {
	m.annotations[in.Key] = in.Value
	return &sdk.Empty{}, nil
}

func (m *sdkMock) SetCondition(ctx context.Context, in *sdk.KeyValue, opts ...grpc.CallOption) (*sdk.Empty, error) {
	m.conditions[in.Key] = in.Value
	return &sdk.Empty{}, nil
}

func (m *sdkMock) WatchGameServer(ctx context.Context, in *sdk.Empty, opts ...grpc.CallOption) (sdk.SDK_WatchGameServerClient, error) {
	panic("implement me")
}

func (m *sdkMock) GetGameServer(ctx context.Context, in *sdk.Empty, opts ...grpc.CallOption) (*sdk.GameServer, error) {
	return &sdk.GameServer{}, nil
}