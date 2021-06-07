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

package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/ocgi/carrier-sdk/api/webhook"
	carrierv1 "github.com/ocgi/carrier/pkg/apis/carrier/v1alpha1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var defaultTimeout = 15 * time.Second
var defaultPeriodTime = 5 * time.Second

// Options defines a func whose parameter is WebhookReview
type Options func(*webhook.WebhookReview)

// WebhookClient defines webhook client to query custom servers
type WebhookClient struct {
	config *carrierv1.Configurations
	name   string
	period time.Duration
	client http.Client
}

// NewWebhookClient creates a new client
func NewWebhookClient(config *carrierv1.Configurations, name string) *WebhookClient {
	timeout := defaultTimeout
	period := defaultPeriodTime
	if config == nil {
		return nil
	}
	if config.TimeoutSeconds != nil && *config.TimeoutSeconds != 0 {
		timeout = time.Duration(*config.TimeoutSeconds) * time.Second
	}
	if config.PeriodSeconds != nil && *config.PeriodSeconds != 0 && *config.PeriodSeconds >= 1 {
		period = time.Duration(*config.PeriodSeconds) * time.Second
	}
	return &WebhookClient{
		config: config,
		name:   name,
		client: http.Client{
			Timeout: timeout,
		},
		period: period,
	}
}

// Request send request to the custom webhook
func (s *WebhookClient) Request(gs *carrierv1.GameServer, options ...Options) (*webhook.WebhookResponse, error) {
	u, err := s.buildURLFromWebhookPolicy()
	if err != nil {
		return nil, err
	}

	wr := webhook.WebhookReview{
		Request: &webhook.WebhookRequest{
			UID:       uuid.NewUUID(),
			Name:      gs.Name,
			Namespace: gs.Namespace,
		},
		Response: nil,
	}
	for _, opt := range options {
		opt(&wr)
	}
	b, err := json.Marshal(wr)
	if err != nil {
		return nil, err
	}
	res, err := s.client.Post(
		u.String(),
		"application/json",
		strings.NewReader(string(b)),
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := res.Body.Close(); cerr != nil {
			if err != nil {
				err = errors.Wrap(err, cerr.Error())
			} else {
				err = cerr
			}
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code %d from the server: %s", res.StatusCode, u.String())
	}
	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var newWebhookReview webhook.WebhookReview
	err = json.Unmarshal(result, &newWebhookReview)
	if err != nil {
		return nil, err
	}
	return newWebhookReview.Response, nil
}

// Check checks the response
func (s *WebhookClient) Check(gs *carrierv1.GameServer) (*webhook.WebhookResponse, error) {
	return s.Request(gs)
}

// Push informs the webhook about constraints.
func (s *WebhookClient) Push(gs *carrierv1.GameServer) (*webhook.WebhookResponse, error) {
	return s.Request(gs, func(review *webhook.WebhookReview) {
		review.Request.Constraints = convertConstraints(gs.Spec.Constraints)
	})
}

// Period return request period
func (s *WebhookClient) Period() time.Duration {
	return s.period
}

// Name return webhook name
func (s *WebhookClient) Name() string {
	return s.name
}

// buildURLFromWebhookPolicy - build URL for Webhook and set CARoot for client Transport
func (s *WebhookClient) buildURLFromWebhookPolicy() (u *url.URL, err error) {
	w := s.config.ClientConfig
	if w.URL != nil && w.Service != nil {
		return nil, errors.New("service and URL cannot be used simultaneously")
	}

	scheme := "http"
	if w.CABundle != nil {
		scheme = "https"

		if err := s.setCABundle(w.CABundle); err != nil {
			return nil, err
		}
	}

	if w.URL != nil {
		if *w.URL == "" {
			return nil, errors.New("URL was not provided")
		}

		return url.ParseRequestURI(*w.URL)
	}

	if w.Service == nil {
		return nil, errors.New("service was not provided, either URL or Service must be provided")
	}

	if w.Service.Name == "" {
		return nil, errors.New("service name was not provided")
	}

	if w.Service.Path == nil {
		empty := ""
		w.Service.Path = &empty
	}

	if w.Service.Namespace == "" {
		w.Service.Namespace = "default"
	}

	return s.createURL(scheme, w.Service.Name, w.Service.Namespace, *w.Service.Path, w.Service.Port), nil
}

// moved to a separate method to cover it with unit tests and check that URL corresponds to a proper pattern
func (s *WebhookClient) createURL(scheme, name, namespace, path string, port *int32) *url.URL {
	var hostPort int32 = 8000
	if port != nil {
		hostPort = *port
	}

	return &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s.%s.svc:%d", name, namespace, hostPort),
		Path:   path,
	}
}

func (s *WebhookClient) setCABundle(caBundle []byte) error {
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(caBundle); !ok {
		return errors.New("no certs were appended from caBundle")
	}
	s.client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: rootCAs,
		},
	}
	return nil
}

func convertConstraints(constraints []carrierv1.Constraint) []webhook.Constraint {
	var cons []webhook.Constraint
	for _, constraint := range constraints {
		con := webhook.Constraint{
			Type:      string(constraint.Type),
			Effective: constraint.Effective,
			Message:   constraint.Message,
			TimeAdded: &constraint.TimeAdded.Time,
		}
		cons = append(cons, con)
	}
	return cons
}
