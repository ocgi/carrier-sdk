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

package util

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	carrierv1 "github.com/ocgi/carrier/pkg/apis/carrier/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	WebhookAnn        = "carrier.ocgi.dev/webhook-config-name"
	ReadinessWebhook  = "ReadinessWebhook"
	DeletableWebhook  = "DeletableWebhook"
	ConstraintWebhook = "ConstraintWebhook"
)

var onlyOneSignalHandler = make(chan struct{})
var shutdownHandler chan os.Signal
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// SetupSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupSignalHandler() <-chan struct{} {
	close(onlyOneSignalHandler) // panics when called twice

	shutdownHandler = make(chan os.Signal, 2)

	stop := make(chan struct{})
	signal.Notify(shutdownHandler, shutdownSignals...)
	go func() {
		<-shutdownHandler
		close(stop)
		<-shutdownHandler
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

func newGameServerCondition(condType carrierv1.GameServerConditionType,
	status carrierv1.ConditionStatus) *carrierv1.GameServerCondition {
	return &carrierv1.GameServerCondition{
		Type:               condType,
		Status:             status,
		LastProbeTime:      metav1.Now(),
		LastTransitionTime: metav1.Now(),
	}
}

// SetRequestToGameServerCondition convert request to GameServer condition
func SetRequestToGameServerCondition(req SetRequest) (*carrierv1.GameServerCondition, error) {
	var (
		status   carrierv1.ConditionStatus
		condType carrierv1.GameServerConditionType
	)

	v := req.KeyValue
	if v == nil {
		return nil, errors.New("condition value is nil")
	}

	if v.Value == "True" {
		status = carrierv1.ConditionTrue
	} else {
		status = carrierv1.ConditionFalse
	}

	switch req.Type {
	case SetCondition:
		condType = carrierv1.GameServerConditionType(req.KeyValue.Key)
	default:
		return nil, errors.Errorf("invalid condition type: %v", req.Type)
	}

	return newGameServerCondition(condType, status), nil
}

// GetGameServerCondition returns the condition with the provided type.
func GetGameServerCondition(status carrierv1.GameServerStatus,
	condType carrierv1.GameServerConditionType) *carrierv1.GameServerCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetGameServerCondition updates the GameServer to include the provided condition. If the condition that
// we are about to add already exists and has the same status, then we are not going to update.
func SetGameServerCondition(status *carrierv1.GameServerStatus, condition carrierv1.GameServerCondition) {
	currentCond := GetGameServerCondition(*status, condition.Type)

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	// currentCond is nil or condition status is changed.
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of GameServer conditions without conditions with the provided type.
func filterOutCondition(conditions []carrierv1.GameServerCondition,
	condType carrierv1.GameServerConditionType) []carrierv1.GameServerCondition {
	var newConditions []carrierv1.GameServerCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// GetWebhookConfigName returns webhook name according to annotations.
func GetWebhookConfigName(gs *carrierv1.GameServer) string {
	if gs == nil {
		return ""
	}
	if len(gs.Annotations) == 0 {
		return ""
	}
	return gs.Annotations[WebhookAnn]
}

// Bool2Str converts bool to string
func Bool2Str(state bool) string {
	switch state {
	case true:
		return "True"
	case false:
		return "False"
	}
	return "False"
}

// GetContainer returns the container name
func GetContainer(spec *corev1.PodSpec) *corev1.Container {
	for _, container := range spec.Containers {
		if container.Name != "server" {
			continue
		}
		return &container
	}
	return nil
}

// WithExponentialBackoff will retry webhookFn up to 5 times with exponentially increasing backoff when
// it returns an error。
func WithExponentialBackoff(initialBackoff time.Duration, webhookFn func() error) error {
	backoff := wait.Backoff{
		Duration: initialBackoff,
		Factor:   1.5,
		Jitter:   0,
		Steps:    5,
	}

	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := webhookFn()
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}
