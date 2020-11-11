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
#include <unistd.h>

#include <grpcpp/grpcpp.h>

#include "sdk.grpc.pb.h"

using grpc::Server;
using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::ServerWriter;
using grpc::ServerReader;
using grpc::Status;

using namespace carrier::dev::sdk;

class ServerImpl: public SDK::Service {
  Status GetGameServer(ServerContext* context, const Empty*, GameServer* response) override {
    (void)context;
    std::cout<< __func__ <<std::endl;
    mock_GameServer(response);
    return Status::OK;
  }

  Status WatchGameServer(ServerContext *context, const Empty *, ServerWriter<GameServer> *writer) override {
    (void)context;
    std::cout<< __func__ <<std::endl;
    GameServer server;
    do {
      sleep(1);
      ServerImpl::mock_GameServer(&server);
    } while (writer->Write(server));
    std::cout<< "watch closed"<<std::endl;
    return Status::OK;
  }

  Status SetLabel(ServerContext* context, const KeyValue* request, Empty*) override {
    (void)context;
    std::cout<< __func__ <<": "<<request->key()<<" = " << request->value()<<std::endl;
    return Status::OK;
  }
  Status SetAnnotation(ServerContext* context, const KeyValue* request, Empty*) override {
    (void)context;
    std::cout<< __func__ <<": "<<request->key()<<" = " << request->value()<<std::endl;
    return Status::OK;
  }
  Status SetCondition(ServerContext* context, const KeyValue* request, Empty*) override {
    (void)context;
    std::cout<< __func__ <<": "<<request->key()<<" = " << request->value()<<std::endl;
    return Status::OK;
  }

public:
  static void mock_GameServer(GameServer *response) {
    GameServer_ObjectMeta *meta = new GameServer_ObjectMeta();
    meta->set_name("mock_server");
    response->set_allocated_object_meta(meta);

    GameServer_Status *status = new GameServer_Status();
    GameServer_Status_LoadBalancerStatus *lbStatus = new GameServer_Status_LoadBalancerStatus();
    GameServer_Status_LoadBalancerStatus_LoadBalancerIngress *ingress = lbStatus->add_ingress();
    ingress->set_ip("10.10.10.10");

    GameServer_Status_LoadBalancerStatus_LoadBalancerIngress_LoadBalancerPort *port = ingress->add_ports();

    google::protobuf::Int32Value *value = new google::protobuf::Int32Value();
    value->set_value(80);
    port->set_allocated_container_port(value);

    status->set_allocated_load_balancer_status(lbStatus);
    response->set_allocated_status(status);
  }
};

int main() {
  std::string server_address("127.0.0.1:9020");

  ServerBuilder builder;
  ServerImpl service;

  builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
  builder.RegisterService(&service);
  std::unique_ptr<Server> server(builder.BuildAndStart());
  std::cout << "Server listening on " << server_address << std::endl;
  server->Wait();
}
