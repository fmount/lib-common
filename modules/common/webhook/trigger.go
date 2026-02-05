/*
Copyright 2024 Red Hat

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

// Package webhook provides validation utilities for admission webhooks and RFC compliance
package webhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrWebhookTriggerTimeout is returned when the webhook trigger annotation
// remains on the object longer than the configured timeout, indicating the
// webhook failed to process or remove it
var ErrWebhookTriggerTimeout = errors.New("webhook failed to process trigger annotation within timeout")

// EnsureWebhookTrigger ensures the reconcile trigger annotation is set to trigger UPDATE webhook
// The annotation value is set to the current timestamp to ensure it triggers updates
// Returns ctrl.Result{Requeue: true}, nil to requeue and let the defer block save the annotation
// reason is logged to help identify why the trigger was added (e.g., "Service name caching")
// timeout specifies how long to wait before considering the annotation stale (defaults to 5 minutes if 0)
// If the annotation exists for more than the timeout, it's considered stale and removed with an error
//
// example usage:
//
//	result, err := webhook.EnsureWebhookTrigger(
//		ctx,
//		instance,
//		"openstack.org/reconcile-trigger",
//		"Cinder service name caching",
//		log,
//		0,  // use default 5 minute timeout
//	)
func EnsureWebhookTrigger(
	ctx context.Context,
	obj client.Object,
	annotationKey string,
	reason string,
	log logr.Logger,
	timeout time.Duration,
) (ctrl.Result, error) {
	// Default to 5 minutes if timeout not provided
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if val, exists := annotations[annotationKey]; exists {
		// Annotation exists - check if it's stale (webhook may have failed)
		if timestamp, err := time.Parse(time.RFC3339, val); err == nil {
			if time.Since(timestamp) > timeout {
				// Annotation is stale - webhook failed to process or remove it
				log.Error(nil, "Stale reconcile-trigger annotation detected, removing", "reason", reason, "age", time.Since(timestamp))
				delete(annotations, annotationKey)
				obj.SetAnnotations(annotations)
				return ctrl.Result{}, fmt.Errorf("%w: %s after %v", ErrWebhookTriggerTimeout, reason, timeout)
			}
		}
		// Annotation exists but not stale yet - webhook is still processing
		log.Info(fmt.Sprintf("Waiting for webhook to process: %s", reason))
		return ctrl.Result{Requeue: true}, nil
	}

	// Add annotation with timestamp to trigger webhook
	// The defer block will save this change, which triggers the UPDATE webhook
	annotations[annotationKey] = metav1.Now().Format(time.RFC3339)
	obj.SetAnnotations(annotations)
	log.Info(fmt.Sprintf("Adding reconcile-trigger annotation: %s", reason))

	// Requeue - defer will save annotation, which triggers webhook
	return ctrl.Result{Requeue: true}, nil
}
