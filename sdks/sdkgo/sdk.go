// Copyright 2021 The OCGI Authors.
// Copyright 2018 Google LLC All Rights Reserved.
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

// Package sdkgo is the Go game server sdk
package sdkgo

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	sdk "github.com/ocgi/carrier-sdk/sdks/sdkgo/api/v1alpha1"
)

// GameServerCallback is a function definition to be called
// when a GameServer CRD has been changed
type GameServerCallback func(gs *sdk.GameServer)

// SDK is an instance of the Carrier SDK
type SDK struct {
	client sdk.SDKClient
	ctx    context.Context
}

// NewSDK starts a new SDK instance, and connects to
// localhost on port 9020. Blocks until connection and handshake are made.
// Times out after 30 seconds.
func NewSDK() (*SDK, error) {
	p := os.Getenv("CARRIER_SDK_GRPC_PORT")
	if p == "" {
		p = "9020"
	}
	addr := fmt.Sprintf("localhost:%s", p)
	s := &SDK{
		ctx: context.Background(),
	}
	// block for at least 30 seconds
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return s, errors.Wrapf(err, "could not connect to %s", addr)
	}
	s.client = sdk.NewSDKClient(conn)
	return s, nil
}

// SetLabel sets a metadata label on the `GameServer` with the prefix
// carrier.ocgi.dev/sdk-
func (s *SDK) SetLabel(key, value string) error {
	kv := &sdk.KeyValue{Key: key, Value: value}
	_, err := s.client.SetLabel(s.ctx, kv)
	return errors.Wrap(err, "could not set label")
}

// SetAnnotation sets a metadata annotation on the `GameServer` with the prefix
// carrier.ocgi.dev/sdk-
func (s *SDK) SetAnnotation(key, value string) error {
	kv := &sdk.KeyValue{Key: key, Value: value}
	_, err := s.client.SetAnnotation(s.ctx, kv)
	return errors.Wrap(err, "could not set annotation")
}

// SetCondition sets a condition on the `GameServer`
func (s *SDK) SetCondition(key, value string) error {
	kv := &sdk.KeyValue{Key: key, Value: value}
	_, err := s.client.SetCondition(s.ctx, kv)
	return errors.Wrap(err, "could not set condition")
}

// GameServer retrieve the GameServer details
func (s *SDK) GameServer() (*sdk.GameServer, error) {
	gs, err := s.client.GetGameServer(s.ctx, &sdk.Empty{})
	return gs, errors.Wrap(err, "could not retrieve gameserver")
}

// WatchGameServer asynchronously calls the given GameServerCallback with the current GameServer
// configuration when the backing GameServer configuration is updated.
// This function can be called multiple times to add more than one GameServerCallback.
func (s *SDK) WatchGameServer(f GameServerCallback) error {
	stream, err := s.client.WatchGameServer(s.ctx, &sdk.Empty{})
	if err != nil {
		return errors.Wrap(err, "could not watch gameserver")
	}

	go func() {
		for {
			var gs *sdk.GameServer
			gs, err = stream.Recv()
			if err != nil {
				if err == io.EOF {
					_, _ = fmt.Fprintln(os.Stderr, "gameserver event stream EOF received")
					return
				}
				_, _ = fmt.Fprintf(os.Stderr, "error watching GameServer: %s\n", err.Error())
				// This is to wait for the reconnection, and not peg the CPU at 100%
				time.Sleep(time.Second)
				continue
			}
			f(gs)
		}
	}()
	return nil
}
