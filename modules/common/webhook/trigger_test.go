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

package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestEnsureWebhookTrigger(t *testing.T) {
	const annotationKey = "test.org/reconcile-trigger"

	tests := []struct {
		name                string
		existingAnnotations map[string]string
		reason              string
		timeout             time.Duration
		wantRequeue         bool
		wantError           bool
	}{
		{
			name:                "no existing annotation - should add and requeue",
			existingAnnotations: nil,
			reason:              "test reason",
			timeout:             0, // default
			wantRequeue:         true,
			wantError:           false,
		},
		{
			name: "existing fresh annotation - should requeue",
			existingAnnotations: map[string]string{
				annotationKey: metav1.Now().Format(time.RFC3339),
			},
			reason:      "test reason",
			timeout:     0, // default
			wantRequeue: true,
			wantError:   false,
		},
		{
			name: "existing stale annotation with default timeout - should error",
			existingAnnotations: map[string]string{
				annotationKey: metav1.Time{Time: time.Now().Add(-10 * time.Minute)}.Format(time.RFC3339),
			},
			reason:      "test reason",
			timeout:     0, // default 5 minutes
			wantRequeue: false,
			wantError:   true,
		},
		{
			name: "existing annotation within custom timeout - should requeue",
			existingAnnotations: map[string]string{
				annotationKey: metav1.Time{Time: time.Now().Add(-2 * time.Minute)}.Format(time.RFC3339),
			},
			reason:      "test reason",
			timeout:     10 * time.Minute, // custom long timeout
			wantRequeue: true,
			wantError:   false,
		},
		{
			name: "existing annotation exceeds custom timeout - should error",
			existingAnnotations: map[string]string{
				annotationKey: metav1.Time{Time: time.Now().Add(-2 * time.Minute)}.Format(time.RFC3339),
			},
			reason:      "test reason",
			timeout:     1 * time.Minute, // custom short timeout
			wantRequeue: false,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := context.Background()

			// Create a test object (using ConfigMap as a simple client.Object)
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-obj",
					Namespace:   "test-ns",
					Annotations: tt.existingAnnotations,
				},
			}

			// Create a no-op logger for testing
			log := logr.Discard()

			result, err := EnsureWebhookTrigger(ctx, obj, annotationKey, tt.reason, log, tt.timeout)

			if tt.wantError {
				g.Expect(err).To(HaveOccurred())
				g.Expect(obj.GetAnnotations()).ToNot(HaveKey(annotationKey))
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(obj.GetAnnotations()).To(HaveKey(annotationKey))
				// Verify the annotation value is a valid RFC3339 timestamp
				timestamp := obj.GetAnnotations()[annotationKey]
				_, parseErr := time.Parse(time.RFC3339, timestamp)
				g.Expect(parseErr).ToNot(HaveOccurred())
			}

			if tt.wantRequeue {
				g.Expect(result).To(Equal(ctrl.Result{Requeue: true}))
			} else {
				g.Expect(result).To(Equal(ctrl.Result{}))
			}
		})
	}
}

func TestEnsureWebhookTrigger_AnnotationFormat(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	const annotationKey = "test.org/reconcile-trigger"

	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-obj",
			Namespace: "test-ns",
		},
	}

	log := logr.Discard()
	_, err := EnsureWebhookTrigger(ctx, obj, annotationKey, "test", log, 0)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(obj.GetAnnotations()).To(HaveKey(annotationKey))

	// Verify the timestamp format
	timestampStr := obj.GetAnnotations()[annotationKey]
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(timestamp).To(BeTemporally("~", time.Now(), time.Second))
}
