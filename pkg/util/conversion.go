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
	sdkapi "github.com/ocgi/carrier-sdk/sdks/sdkgo/api/v1alpha1"
	carrierv1 "github.com/ocgi/carrier/pkg/apis/carrier/v1alpha1"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
)

// Convert_Carrier_To_GRPC converts a K8s GameServer object, into a gRPC SDK GameServer object
func Convert_Carrier_To_GRPC(gs *carrierv1.GameServer) *sdkapi.GameServer {
	meta := gs.ObjectMeta
	status := gs.Status
	result := &sdkapi.GameServer{
		ObjectMeta: &sdkapi.GameServer_ObjectMeta{
			Name:              meta.Name,
			Namespace:         meta.Namespace,
			Uid:               string(meta.UID),
			ResourceVersion:   meta.ResourceVersion,
			Generation:        meta.Generation,
			CreationTimestamp: meta.CreationTimestamp.Unix(),
			Annotations:       meta.Annotations,
			Labels:            meta.Labels,
		},
		Spec: &sdkapi.GameServer_Spec{
			ReadinessGates: gs.Spec.ReadinessGates,
			DeletableGates: gs.Spec.DeletableGates,
		},
		Status: &sdkapi.GameServer_Status{
			State:    string(status.State),
			Address:  status.Address,
			NodeName: status.NodeName,
		},
	}
	if meta.DeletionTimestamp != nil {
		result.ObjectMeta.DeletionTimestamp = meta.DeletionTimestamp.Unix()
	}

	// loop around and add all the load-balancer ingress
	lbStatus := status.LoadBalancerStatus
	if lbStatus != nil {
		var lbIngress []*sdkapi.GameServer_Status_LoadBalancerStatus_LoadBalancerIngress
		for _, i := range lbStatus.Ingress {
			var lbPorts []*sdkapi.GameServer_Status_LoadBalancerStatus_LoadBalancerIngress_LoadBalancerPort
			for _, p := range i.Ports {
				port := &sdkapi.GameServer_Status_LoadBalancerStatus_LoadBalancerIngress_LoadBalancerPort{}
				if p.ContainerPort != nil && p.ExternalPort != nil {
					port.ContainerPort = wrappers.Int32(*p.ContainerPort)
					port.ExternalPort = wrappers.Int32(*p.ExternalPort)
				}
				if p.ContainerPortRange != nil {
					port.ContainerPortRange = &sdkapi.GameServer_Status_LoadBalancerStatus_LoadBalancerIngress_LoadBalancerPort_PortRange{
						MinPort: p.ContainerPortRange.MinPort,
						MaxPort: p.ContainerPortRange.MaxPort,
					}
				}
				if p.ExternalPortRange != nil {
					port.ExternalPortRange = &sdkapi.GameServer_Status_LoadBalancerStatus_LoadBalancerIngress_LoadBalancerPort_PortRange{
						MinPort: p.ExternalPortRange.MinPort,
						MaxPort: p.ExternalPortRange.MaxPort,
					}
				}
				port.Protocal = string(p.Protocol)
				lbPorts = append(lbPorts, port)
			}
			ing := &sdkapi.GameServer_Status_LoadBalancerStatus_LoadBalancerIngress{
				Ip:    i.IP,
				Ports: lbPorts,
			}
			lbIngress = append(lbIngress, ing)
		}
		result.Status.LoadBalancerStatus = &sdkapi.GameServer_Status_LoadBalancerStatus{
			Ingress: lbIngress,
		}
	}

	for _, constraint := range gs.Spec.Constraints {
		sdkConstraint := &sdkapi.GameServer_Spec_Constraint{
			Type:    string(constraint.Type),
			Message: constraint.Message,
		}

		if constraint.Effective != nil {
			sdkConstraint.Effective = *constraint.Effective
		}
		if constraint.TimeAdded != nil {
			sdkConstraint.TimeAdded = constraint.TimeAdded.Unix()
		}
		result.Spec.Constraints = append(result.Spec.Constraints, sdkConstraint)
	}
	return result
}
