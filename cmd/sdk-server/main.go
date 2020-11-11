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

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/ocgi/carrier-sdk/pkg/controller"
	"github.com/ocgi/carrier-sdk/pkg/sdkserver"
	"github.com/ocgi/carrier-sdk/pkg/util"
	sdkapi "github.com/ocgi/carrier-sdk/sdks/sdkgo/api/v1alpha1"
	"github.com/ocgi/carrier/pkg/client/clientset/versioned"
	"github.com/ocgi/carrier/pkg/client/informers/externalversions"
)

var (
	address  string
	grpcPort int
	httpPort int

	// K8s config
	kubeApiQps     float64
	kubeApiBurst   int
	masterUrl      string
	kubeconfigPath string

	gameServerNameEnv string
	podNamespaceEnv   string
)

func init() {
	flag.StringVar(&address, "address", "127.0.0.1", `The Address to bind the server grpcPort to.`)
	flag.IntVar(&grpcPort, "grpc-port", 9020, `Port on which to bind the gRPC server.`)
	flag.IntVar(&httpPort, "http-port", 9021, `Port on which to bind the HTTP server.`)
	flag.Float64Var(&kubeApiQps, "kube-api-qps", 5.0, `QPS limit when making requests to Kubernetes apiserver`)
	flag.IntVar(&kubeApiBurst, "kube-api-burst", 10.0, `QPS burst limit when making requests to Kubernetes apiserver`)
	flag.StringVar(&masterUrl, "master", "", `K8s master url.`)
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", `K8s config file path.`)
	initEnv()
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	stop := util.SetupSignalHandler()

	// create kube client
	config := createKubeConfig(float32(kubeApiQps), kubeApiBurst)
	kubeClient := kubernetes.NewForConfigOrDie(config)
	carrierClient := versioned.NewForConfigOrDie(config)
	factory := externalversions.NewFilteredSharedInformerFactory(carrierClient, 0, podNamespaceEnv, func(opts *metav1.ListOptions) {
		selcetor := fields.OneTermEqualSelector("metadata.name", gameServerNameEnv)
		opts.FieldSelector = selcetor.String()
	})

	sendCh := make(chan *sdkapi.GameServer, 10)
	defer close(sendCh)

	gsInformer := factory.Carrier().V1alpha1().GameServers()
	c := controller.NewController(podNamespaceEnv, gameServerNameEnv, kubeClient, carrierClient, gsInformer, sendCh)

	factory.Start(stop)

	klog.V(4).Info("Wait for cache sync")
	if !cache.WaitForCacheSync(stop, gsInformer.Informer().HasSynced) {
		klog.Fatal("failed to wait for caches to sync")
	}

	gsLister := gsInformer.Lister()
	s, err := sdkserver.NewSDKServer(gameServerNameEnv,
		podNamespaceEnv, kubeClient, carrierClient, gsLister, sendCh)
	if err != nil {
		klog.Fatalf("Could not start sidecar err: %v", err)
	}

	s.Run(stop)
	c.SetNotifyFunc(s.SetCondition)
	go c.Run(stop)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// grpc endpoint
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()
	sdkapi.RegisterSDKServer(grpcServer, s)

	grpcEndpoint := fmt.Sprintf("%s:%d", address, grpcPort)
	go runGrpc(grpcServer, grpcEndpoint)

	// grpc-gateway endpoint
	mux := gwruntime.NewServeMux()
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", address, httpPort),
		Handler: mux,
	}
	defer httpServer.Close()
	go runGateway(ctx, grpcEndpoint, mux, httpServer)
	<-stop
	klog.Info("shutting down sdk server")
}

func initEnv() {
	gameServerNameEnv = os.Getenv("GAMESERVER_NAME")
	if gameServerNameEnv == "" {
		panic("can not get gamerserver pod name env")
	}

	podNamespaceEnv = os.Getenv("POD_NAMESPACE")
	if podNamespaceEnv == "" {
		panic("can not get pod namespace")
	}
}

func createKubeConfig(kubeApiQps float32, kubeApiBurst int) *rest.Config {
	var (
		config *rest.Config
		err    error
	)

	if masterUrl == "" && kubeconfigPath == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags(masterUrl, kubeconfigPath)
	}
	if err != nil {
		klog.Fatalf("Failed to create config err: %v", err)
	}

	config.QPS = kubeApiQps
	config.Burst = kubeApiBurst
	return config
}

// runGrpc runs the grpc service
func runGrpc(grpcServer *grpc.Server, grpcEndpoint string) {
	lis, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		klog.Fatalf("Could not listen on grpc endpoint err: %v", err)
	}

	klog.Infof("Starting SDKServer grpc service with grpcEndpoint: %v", grpcEndpoint)
	if err := grpcServer.Serve(lis); err != nil {
		klog.Fatalf("Could not serve grpc server err: %v", err)
	}
}

// runGateway runs the grpc-gateway
func runGateway(ctx context.Context, grpcEndpoint string, mux *gwruntime.ServeMux, httpServer *http.Server) {
	conn, err := grpc.DialContext(ctx, grpcEndpoint, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		klog.Fatalf("Could not dial grpc server err: %v", err)
	}

	if err := sdkapi.RegisterSDKHandler(ctx, mux, conn); err != nil {
		klog.Fatalf("Could not register sdk grpc-gateway err: %v", err)
	}

	klog.Infof("Starting SDKServer grpc-gateway with httpEndpoint: %v", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			klog.Infof("http server closed err: %v", err)
		} else {
			klog.Fatalf("Could not serve http server err: %v", err)
		}
	}
}
