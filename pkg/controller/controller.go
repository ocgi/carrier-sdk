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

package controller

import (
	"context"
	"reflect"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"

	"github.com/ocgi/carrier-sdk/pkg/client"
	"github.com/ocgi/carrier-sdk/pkg/util"
	sdkapi "github.com/ocgi/carrier-sdk/sdks/sdkgo/api/v1alpha1"
	carrierv1alpha1 "github.com/ocgi/carrier/pkg/apis/carrier/v1alpha1"
	"github.com/ocgi/carrier/pkg/client/clientset/versioned"
	"github.com/ocgi/carrier/pkg/client/informers/externalversions/carrier/v1alpha1"
	listerv1 "github.com/ocgi/carrier/pkg/client/listers/carrier/v1alpha1"
)

type f func(context.Context, *sdkapi.KeyValue) (*sdkapi.Empty, error)

// Controller is a the main GameServer crd controller
type Controller struct {
	namespace          string
	name               string
	gameServerLister   listerv1.GameServerLister
	gameServerSynced   cache.InformerSynced
	gameServerInformer v1alpha1.GameServerInformer
	kubeClient         kubernetes.Interface
	carrierClient      versioned.Interface
	recorder           record.EventRecorder
	webhookConfig      *carrierv1alpha1.WebhookConfiguration
	cancel             context.CancelFunc
	ctx                context.Context
	sendChan           chan *sdkapi.GameServer
	setCondition       f
}

// NewController returns a new GameServer crd controller
func NewController(
	namespace, name string,
	kubeClient kubernetes.Interface,
	carrierClient versioned.Interface,
	gameServerInformer v1alpha1.GameServerInformer,
	sendChan chan *sdkapi.GameServer) *Controller {

	c := &Controller{
		namespace:        namespace,
		name:             name,
		gameServerLister: gameServerInformer.Lister(),
		gameServerSynced: gameServerInformer.Informer().HasSynced,
		kubeClient:       kubeClient,
		carrierClient:    carrierClient,
		sendChan:         sendChan,
	}

	s := scheme.Scheme
	// Register operator types with the runtime scheme.
	s.AddKnownTypes(carrierv1alpha1.SchemeGroupVersion, &carrierv1alpha1.GameServer{})

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface:
		kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(s, corev1.EventSource{Component: "gameserver-controller"})

	gameServerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.notify,
	})
	return c
}

// SetNotifyFunc set the notification func name.
// This should be called when notify func is ready.
func (c *Controller) SetNotifyFunc(setCondition f) {
	c.setCondition = setCondition
}

func (c *Controller) notify(oldObj, newObj interface{}) {
	klog.V(5).Infof("Sending GameServer Event to connectedStreams")
	if c.setCondition == nil {
		return
	}

	old := oldObj.(*carrierv1alpha1.GameServer)
	gs := newObj.(*carrierv1alpha1.GameServer)
	// inplace update
	constraintChanged := !reflect.DeepEqual(old.Spec.Constraints, gs.Spec.Constraints) && len(gs.Spec.Constraints) != 0
	klog.V(5).Infof("Constraints exist: %v", constraintChanged)
	oldContainer := util.GetContainer(&old.Spec.Template.Spec)
	newContainer := util.GetContainer(&gs.Spec.Template.Spec)
	if oldContainer != nil && newContainer != nil &&
		oldContainer.Image != newContainer.Image {
		if c.cancel != nil {
			c.cancel()
		}
		klog.V(5).Infof("Image changed to %v", newContainer.Image)
		c.ctx, c.cancel = context.WithCancel(context.Background())
		if err := c.sendWebhookRequests(gs, c.setCondition); err != nil {
			klog.Error(err)
		}
	}

	if constraintChanged && c.webhookConfig != nil {
		klog.Info("sending game server notification")
		startHookConstraintNotify(c.ctx, gs, c.webhookConfig.Webhooks, c.setCondition)
	}
	c.sendChan <- util.Convert_Carrier_To_GRPC(gs)
}

// Run starts the controller.
func (c *Controller) Run(stop <-chan struct{}) error {
	gs, err := c.gameServerLister.GameServers(c.namespace).Get(c.name)
	if err != nil {
		return err
	}
	if c.cancel == nil {
		c.ctx, c.cancel = context.WithCancel(context.Background())
	}
	if err = c.sendWebhookRequests(gs, c.setCondition); err != nil {
		return err
	}
	<-stop
	return nil
}

func startHookConstraintNotify(ctx context.Context, gs *carrierv1alpha1.GameServer,
	cfgs []carrierv1alpha1.Configurations, fun f) {
	var wg sync.WaitGroup
	for _, cfg := range cfgs {
		if cfg.Type == nil || *cfg.Type != util.ConstraintWebhook {
			continue
		}
		wg.Add(1)
		// if psuh success, wg Done
		// if Once, go routine exit; Otherwise, continue notification util context done.
		go func(cfg carrierv1alpha1.Configurations) {
			w := client.NewWebhookClient(&cfg, *cfg.Name)
			var notified bool
			err := wait.PollImmediateUntil(w.Period(), func() (bool, error) {
				_, err := w.Push(gs)
				if err != nil {
					klog.Error(err)
					return false, nil
				}
				if !notified {
					wg.Done()
					notified = true
				}
				// if once return true, otherwise return false, always retry
				if cfg.RequestPolicy == carrierv1alpha1.RequestPolicyOnce {
					return true, nil
				}
				return false, nil
			}, ctx.Done())
			if err != nil {
				klog.Errorf("Send constraint: %v to %v error: %v", gs.Spec.Constraints, cfg.Name, err)
				return
			}
		}(cfg)
	}
	wg.Wait()
	// wait all notified, then send deletable.
	startHooks(ctx, gs, cfgs, util.DeletableWebhook, fun)
}

func (c *Controller) sendWebhookRequests(gs *carrierv1alpha1.GameServer, fun f) error {
	if len(gs.Spec.DeletableGates) != 0 || len(gs.Spec.ReadinessGates) != 0 {
		// webconfig
		wbName := util.GetWebhookConfigName(gs)
		if len(wbName) != 0 {
			wb, err := c.carrierClient.CarrierV1alpha1().
				WebhookConfigurations(gs.Namespace).Get(wbName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			c.webhookConfig = wb
			startHooks(c.ctx, gs, wb.Webhooks, util.ReadinessWebhook, fun)
		}
	}
	return nil
}

func startHooks(ctx context.Context, gs *carrierv1alpha1.GameServer,
	cfgs []carrierv1alpha1.Configurations, hookType string, fun f) {
	for i := range cfgs {
		if cfgs[i].Type == nil {
			continue
		}
		if *cfgs[i].Type != hookType {
			continue
		}

		go func(i int) {
			startHook(ctx, gs, cfgs[i], fun)
		}(i)
	}
}

func startHook(ctx context.Context, gs *carrierv1alpha1.GameServer, cfg carrierv1alpha1.Configurations, fun f) {
	w := client.NewWebhookClient(&cfg, *cfg.Name)
	err := wait.PollImmediateUntil(w.Period(), func() (done bool, err error) {
		res, err := w.Check(gs)
		if err != nil {
			klog.Error(err)
			return false, nil
		}
		if _, err := fun(ctx, &sdkapi.KeyValue{
			Key:   w.Name(),
			Value: util.Bool2Str(res.Allowed),
		}); err != nil {
			klog.Error(err)
			return false, nil
		}
		if !res.Allowed {
			return false, nil
		}
		// if once return true, otherwise return false, always retry
		if cfg.RequestPolicy == carrierv1alpha1.RequestPolicyOnce {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
	if err != nil {
		klog.Errorf("send request failed: %v", err)
	}
}
