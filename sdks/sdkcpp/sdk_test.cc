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

#include "sdk.h"

int main() {
  SdkClient client;
  GameServer server;
  assert(client.GetGameServer(&server).ok());
  std::cout<<"GetGameServer: "<<server.DebugString()<<std::endl;

  ClientContext context1;
  auto writer = client.Health(&context1);
  Empty empty;
  writer->Write(empty);
  writer->WritesDone();

  ClientContext context2;
  auto watcher = client.WatchGameServer(&context2);
  watcher->Read(&server);
  std::cout<<"WatchGameServer: "<<server.DebugString()<<std::endl;
  assert(watcher->Read(&server));

  assert(client.SetLabel("key1", "value1").ok());
  assert(client.SetAnnotation("key2", "value2").ok());
  assert(client.SetCondition("key1", "value1").ok());
  return 0;
}
