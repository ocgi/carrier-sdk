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

#include <iostream>
#include <memory>
#include <string>

#include <grpcpp/grpcpp.h>

#include "sdk.grpc.pb.h"

using grpc::Channel;
using grpc::ClientContext;
using grpc::ClientWriter;
using grpc::ClientReader;
using grpc::Status;

using carrier::dev::sdk::BoolValue;
using carrier::dev::sdk::Empty;
using carrier::dev::sdk::GameServer;
using carrier::dev::sdk::KeyValue;
using carrier::dev::sdk::SDK;

class SdkClient
{
public:
  SdkClient() {
    const char* p = getenv("CARRIER_SDK_GRPC_PORT");
    if(p == nullptr) {
      p = "9020";
    }
    auto addr = std::string("localhost:") + p;
    auto channel = grpc::CreateChannel(addr, grpc::InsecureChannelCredentials());
    stub_ = SDK::NewStub(channel);
  }

  // this need a persist *context
  std::unique_ptr<ClientWriter<Empty>> Health(ClientContext *context) {
    Empty reply;
    return stub_->Health(context, &reply);
  }

  Status GetGameServer(GameServer *reply) {
    ClientContext context;
    Empty request;
    return stub_->GetGameServer(&context, request, reply);
  }

  // this need a persist *context
  std::unique_ptr<ClientReader<GameServer>> WatchGameServer(ClientContext *context) {
    Empty request;
    return stub_->WatchGameServer(context, request);
  }

  Status SetLabel(const std::string& key, const std::string& value) {
    ClientContext context;
    KeyValue request;

    request.set_key(key);
    request.set_value(value);

    Empty reply;
    return stub_->SetLabel(&context, request, &reply);
  }

  Status SetAnnotation(const std::string& key, const std::string& value) {
    ClientContext context;
    KeyValue request;

    request.set_key(key);
    request.set_value(value);

    Empty reply;
    return stub_->SetAnnotation(&context, request, &reply);
  }

  Status SetCondition(const std::string& key, const std::string& value) {
    ClientContext context;
    KeyValue request;

    request.set_key(key);
    request.set_value(value);

    Empty reply;
    return stub_->SetCondition(&context, request, &reply);
  }

private:
  std::unique_ptr<SDK::Stub> stub_;
};
