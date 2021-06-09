/*
Copyright 2021 THL A29 Limited, a Tencent company.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// WebhookRequest defines the request to webhook endpoint
type WebhookRequest struct {
	// UID is used for tracing the request and response.
	UID types.UID `json:"uid"`
	// Name is the name of the GameServer
	Name string `json:"name"`
	// Namespace is the workload namespace
	Namespace string `json:"namespace"`
	// Constraints describes the constraints of game server.
	// This filed may be added or changed by controller or manually.
	// If anyone of them is `NotInService` and Effective is `True`,
	// We should react to the constraint.
	Constraints []Constraint `json:"constraints,omitempty"`
	// Status defines the GameServer status, Include: Created, InplaceUpdated.
	Status string `json:"status,omitempty"`
}

const (
	RequestReasonCreated        = "Created"
	RequestReasonInplaceUpdated = "InplaceUpdated"
)

// WebhookResponse defines the response of webhook server
type WebhookResponse struct {
	// UID is used for tracing the request and response.
	// It should be same as it in the request.
	UID types.UID `json:"uid"`
	// Set to false if should not allow
	Allowed bool `json:"allowed"`
	// Reason for this result.
	Reason string `json:"reason,omitempty"`
}

// Constraint describes the constraint info of game server.
type Constraint struct {
	// Type is the ConstraintType name, e.g. NotInService.
	Type string `json:"type"`
	// Effective describes whether the constraint is effective.
	Effective *bool `json:"effective,omitempty"`
	// Message explains why this constraint is added.
	Message string `json:"message,omitempty"`
	// TimeAdded describes when it is added.
	TimeAdded *time.Time `json:"timeAdded,omitempty"`
}

// WebhookReview is passed to the webhook with a populated Request value,
// and then returned with a populated Response.
type WebhookReview struct {
	Request  *WebhookRequest  `json:"request"`
	Response *WebhookResponse `json:"response"`
}
