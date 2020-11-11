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

// Package examples is examples for webhook server

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"k8s.io/klog"

	"github.com/ocgi/carrier-sdk/api/webhook"
)

var port = flag.String("port", "8000", "The port to listen to TCP requests")

var cache = map[string]bool{}
var lock = sync.Mutex{}

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	http.HandleFunc("/ready", ready)
	http.HandleFunc("/set", set)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		klog.Fatalf("HTTP server failed to run: %v", err)
	}
}

// Let /ready return ready and status code 200
func ready(w http.ResponseWriter, r *http.Request) {
	var req webhook.WebhookReview
	res, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	err = json.Unmarshal(res, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	resp := webhook.WebhookResponse{
		UID:     req.Request.UID,
		Allowed: true,
	}

	lock.Lock()
	_, exist := cache[fmt.Sprintf("%s%s", req.Request.Namespace, req.Request.Name)]
	lock.Unlock()

	if !exist {
		resp.Allowed = false
		resp.Reason = "not allowed"
	}
	w.Header().Set("Content-Type", "application/json")
	review := &webhook.WebhookReview{
		Request:  req.Request,
		Response: &resp,
	}
	klog.V(5).Infof("Webhook Review: %v", review)
	result, _ := json.Marshal(&review)
	if _, err := w.Write(result); err != nil {
		klog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// Let /set insert status into map
func set(w http.ResponseWriter, r *http.Request) {
	var req webhook.WebhookRequest
	res, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	err = json.Unmarshal(res, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	lock.Lock()
	cache[fmt.Sprintf("%s%s", req.Namespace, req.Name)] = true
	lock.Unlock()
	ret := `{"status": "ok"}`
	w.Write([]byte(ret))
}
