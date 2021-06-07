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

// Package sdkserver implements SDK server (sidecar).
package sdkserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	"github.com/ocgi/carrier-sdk/pkg/util"
	sdkapi "github.com/ocgi/carrier-sdk/sdks/sdkgo/api/v1alpha1"
	carrierv1alpha1 "github.com/ocgi/carrier/pkg/apis/carrier/v1alpha1"
	"github.com/ocgi/carrier/pkg/client/clientset/versioned"
	typedv1 "github.com/ocgi/carrier/pkg/client/clientset/versioned/typed/carrier/v1alpha1"
	listerv1 "github.com/ocgi/carrier/pkg/client/listers/carrier/v1alpha1"
	"github.com/ocgi/carrier/pkg/util/kube"
)

const (
	// metadataPrefix prefix for labels and annotations
	metadataPrefix = "carrier.ocgi.dev/sdk-"
)

var (
	_ sdkapi.SDKServer = &SDKServer{}
)

type f func(context.Context, *sdkapi.KeyValue) (*sdkapi.Empty, error)

// SDKServer is a gRPC server, that is meant to be a sidecar
// for a GameServer that will update the game server status on SDK requests
type SDKServer struct {
	gameServerName   string
	namespace        string
	client           typedv1.CarrierV1alpha1Interface
	server           *http.Server
	gameServerLister listerv1.GameServerLister

	// Queue for sdk client request
	requestQueue     chan *util.SetRequest
	streamMutex      sync.RWMutex
	connectedStreams []sdkapi.SDK_WatchGameServerServer
	recorder         record.EventRecorder
	notifyQueue      chan *sdkapi.GameServer
	stopCh           <-chan struct{}

	sdkapi.UnimplementedSDKServer
}

// NewSDKServer creates a SDKServer that sets up an
// InClusterConfig for Kubernetes
func NewSDKServer(gameServerName, namespace string, kubeClient kubernetes.Interface, carrierClient versioned.Interface,
	lister listerv1.GameServerLister, sendCh chan *sdkapi.GameServer) (*SDKServer, error) {
	mux := http.NewServeMux()
	s := &SDKServer{
		gameServerName: gameServerName,
		namespace:      namespace,
		client:         carrierClient.CarrierV1alpha1(),
		server: &http.Server{
			Addr:    ":8080",
			Handler: mux,
		},
		streamMutex:      sync.RWMutex{},
		requestQueue:     make(chan *util.SetRequest, 10),
		gameServerLister: lister,
		notifyQueue:      sendCh,
	}
	scm := scheme.Scheme
	// Register crd types with the runtime scheme.
	scm.AddKnownTypes(carrierv1alpha1.SchemeGroupVersion, &carrierv1alpha1.GameServer{})

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&k8sv1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	s.recorder = eventBroadcaster.NewRecorder(scm, corev1.EventSource{Component: "carrier-sdkserver"})

	// endpoint for check SDKServer
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			klog.Errorf("Send ok response on healthz error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	klog.Info("Created SDK server success")

	return s, nil
}

// syncGameServer synchronises the GameServer with the requested operations.
func (s *SDKServer) syncGameServer(obj interface{}) error {
	req, ok := obj.(*util.SetRequest)
	if !ok {
		klog.Errorf("expected SetRequest in queue, but got %T", obj)
		// we don't want the obj be reprocessed, so return nil
		return nil
	}

	switch req.Type {
	case util.SetCondition:
		return s.updateCondition(req)
	case util.SetLabel:
		return s.updateLabels(req)
	case util.SetAnnotation:
		return s.updateAnnotations(req)
	}

	return errors.Errorf("could not sync game server set request: %v", *req)
}

func (s *SDKServer) gameServer() (*carrierv1alpha1.GameServer, error) {
	return s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
}

// updateCondition updates the condition on this GameServer
func (s *SDKServer) updateCondition(req *util.SetRequest) error {
	klog.V(5).Infof("Updating condition, request: %+v", req)
	gameServers := s.client.GameServers(s.namespace)
	gs, err := s.gameServer()
	if err != nil {
		return err
	}
	if req.KeyValue == nil {
		return errors.New("condition value is nil")
	}

	condition, err := util.SetRequestToGameServerCondition(*req)
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()
	util.SetGameServerCondition(&gsCopy.Status, *condition)

	patch, err := kube.CreateMergePatch(gs, gsCopy)
	if err != nil {
		return err
	}

	gs, err = gameServers.Patch(gs.Name, types.MergePatchType, patch, "status")
	if err != nil {
		return errors.Wrapf(err, "failed to update GameServer %s/%s condition: %v",
			s.namespace, s.gameServerName, err)
	}

	message := fmt.Sprintf("Condition changed: %v to %v", condition.Type, condition.Status)
	s.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), message)

	return nil
}

// updateLabels updates the labels on this GameServer
// with the prefix of "carrier.ocgi.dev/sdk-"
func (s *SDKServer) updateLabels(req *util.SetRequest) error {
	klog.V(5).Infof("Updating label, request: %+v", req)

	if req.KeyValue == nil {
		return errors.New("label value is nil")
	}

	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()

	if gsCopy.ObjectMeta.Labels == nil {
		gsCopy.ObjectMeta.Labels = map[string]string{}
	}
	kv := req.KeyValue
	gsCopy.ObjectMeta.Labels[metadataPrefix+kv.Key] = kv.Value

	patch, err := kube.CreateMergePatch(gs, gsCopy)
	if err != nil {
		return err
	}

	_, err = s.client.GameServers(s.namespace).Patch(gs.Name, types.MergePatchType, patch)
	return err
}

// updateAnnotations updates the Annotations on this GameServer
// with the prefix of "carrier.ocgi.dev/sdk-"
func (s *SDKServer) updateAnnotations(req *util.SetRequest) error {
	klog.V(5).Infof("Updating annotation, request: %+v", req)

	if req.KeyValue == nil {
		return errors.New("annotation value is nil")
	}
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()

	if gsCopy.ObjectMeta.Annotations == nil {
		gsCopy.ObjectMeta.Annotations = map[string]string{}
	}
	kv := req.KeyValue
	gsCopy.ObjectMeta.Annotations[metadataPrefix+kv.Key] = kv.Value

	patch, err := kube.CreateMergePatch(gs, gsCopy)
	if err != nil {
		return err
	}

	_, err = s.client.GameServers(s.namespace).Patch(gs.Name, types.MergePatchType, patch)
	return err
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (s *SDKServer) Run(stop <-chan struct{}) {
	s.stopCh = stop
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				klog.Errorf("Health check: http server closed, error: %v", err)
			} else {
				err = errors.Wrap(err, "Could not listen on :8080")
				runtime.HandleError(err)
			}
			return
		}
		defer s.server.Close()
	}()
	go func() {
		for gs := range s.notifyQueue {
			for _, stream := range s.connectedStreams {
				klog.Info("Send notification to stream clients.")
				stream.Send(gs)
			}
		}
	}()
	go func() {
		for gs := range s.requestQueue {
			if err := s.syncGameServer(gs); err != nil {
				klog.Info(err)
			}
		}
	}()
	return
}

// SetLabel adds the Key/Value to be used to set the label with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetLabel(_ context.Context, kv *sdkapi.KeyValue) (*sdkapi.Empty, error) {
	klog.V(5).Info("Adding SetLabel to queue")
	s.requestQueue <- &util.SetRequest{
		Type: util.SetLabel,
		KeyValue: &util.KeyValue{
			Key:   kv.Key,
			Value: kv.Value,
		},
	}
	return &sdkapi.Empty{}, nil
}

// SetAnnotation adds the Key/Value to be used to set the annotations with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetAnnotation(_ context.Context, kv *sdkapi.KeyValue) (*sdkapi.Empty, error) {
	klog.V(5).Info("Adding SetAnnotation to queue")

	s.requestQueue <- &util.SetRequest{
		Type: util.SetAnnotation,
		KeyValue: &util.KeyValue{
			Key:   kv.Key,
			Value: kv.Value,
		},
	}
	return &sdkapi.Empty{}, nil
}

// SetCondition adds the Key/Value to be used to set the condition to GameServer status
func (s *SDKServer) SetCondition(_ context.Context, kv *sdkapi.KeyValue) (*sdkapi.Empty, error) {
	klog.V(5).Info("Adding SetCondition to queue")

	s.requestQueue <- &util.SetRequest{
		Type: util.SetCondition,
		KeyValue: &util.KeyValue{
			Key:   kv.Key,
			Value: kv.Value,
		},
	}
	return &sdkapi.Empty{}, nil
}

// GetGameServer returns the current GameServer configuration and state from the backing GameServer CRD
func (s *SDKServer) GetGameServer(context.Context, *sdkapi.Empty) (*sdkapi.GameServer, error) {
	klog.V(5).Info("Received GetGameServer request")
	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}
	return util.Convert_Carrier_To_GRPC(gs), nil
}

// WatchGameServer sends events through the stream when changes occur to the
// backing GameServer configuration / status
func (s *SDKServer) WatchGameServer(_ *sdkapi.Empty, stream sdkapi.SDK_WatchGameServerServer) error {
	klog.V(5).Info("Received WatchGameServer request, adding stream to connectedStreams")

	s.streamMutex.Lock()
	s.connectedStreams = append(s.connectedStreams, stream)
	s.streamMutex.Unlock()
	// don't exit until we shutdown, because that will close the stream
	<-s.stopCh

	return nil
}
