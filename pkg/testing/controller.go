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

package testing

import (
	"context"
	"net/http"
	"time"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	carrierfake "github.com/ocgi/carrier/pkg/client/clientset/versioned/fake"
	"github.com/ocgi/carrier/pkg/client/informers/externalversions"
)

// Handy tools for testing controllers

// Mocks is a holder for all my fakes and Mocks
type Mocks struct {
	KubeClient             *kubefake.Clientset
	KubeInformerFactory    informers.SharedInformerFactory
	ExtClient              *extfake.Clientset
	CarrierClient          *carrierfake.Clientset
	CarrierInformerFactory externalversions.SharedInformerFactory
	FakeRecorder           *record.FakeRecorder
	Mux                    *http.ServeMux
}

// NewMocks creates a new set of fakes and mocks.
func NewMocks() Mocks {
	kubeClient := &kubefake.Clientset{}
	carrierClient := &carrierfake.Clientset{}

	m := Mocks{
		KubeClient:             kubeClient,
		KubeInformerFactory:    informers.NewSharedInformerFactory(kubeClient, 30*time.Second),
		ExtClient:              &extfake.Clientset{},
		CarrierClient:          carrierClient,
		CarrierInformerFactory: externalversions.NewSharedInformerFactory(carrierClient, 30*time.Second),
		FakeRecorder:           record.NewFakeRecorder(100),
	}
	return m
}

// StartInformers starts new fake informers
func StartInformers(mocks Mocks, sync ...cache.InformerSynced) (<-chan struct{}, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	stop := ctx.Done()

	mocks.KubeInformerFactory.Start(stop)
	mocks.CarrierInformerFactory.Start(stop)

	if !cache.WaitForCacheSync(stop, sync...) {
		panic("Cache never synced")
	}

	return stop, cancel
}

// NewEstablishedCRD fakes CRD installation success.
func NewEstablishedCRD() *v1beta1.CustomResourceDefinition {
	return &v1beta1.CustomResourceDefinition{
		Status: v1beta1.CustomResourceDefinitionStatus{
			Conditions: []v1beta1.CustomResourceDefinitionCondition{{
				Type:   v1beta1.Established,
				Status: v1beta1.ConditionTrue,
			}},
		},
	}
}
