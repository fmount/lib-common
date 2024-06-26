/*
Copyright 2023 Red Hat
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

package helpers

import (
	t "github.com/onsi/gomega"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
)

// ExpectCondition -
func (tc *TestHelper) ExpectCondition(
	name types.NamespacedName,
	getter conditionsGetter,
	conditionType condition.Type,
	expectedStatus corev1.ConditionStatus,
) {
	tc.logger.Info("ExpectCondition", "type", conditionType, "expected status", expectedStatus, "on", name)
	t.Eventually(func(g t.Gomega) {
		conditions := getter.GetConditions(name)
		g.Expect(conditions).NotTo(
			t.BeNil(), "Conditions in nil")
		g.Expect(conditions.Has(conditionType)).To(
			t.BeTrue(), "Does not have condition type %s", conditionType)
		actual := conditions.Get(conditionType).Status
		g.Expect(actual).To(
			t.Equal(expectedStatus),
			"%s condition is in an unexpected state. Expected: %s, Actual: %s, instance name: %s, Conditions: %v",
			conditionType, expectedStatus, actual, name, conditions)
	}, tc.timeout, tc.interval).Should(t.Succeed())
	tc.logger.Info("ExpectCondition succeeded", "type", conditionType, "expected status", expectedStatus, "on", name)
}

// ExpectConditionWithDetails -
func (tc *TestHelper) ExpectConditionWithDetails(
	name types.NamespacedName,
	getter conditionsGetter,
	conditionType condition.Type,
	expectedStatus corev1.ConditionStatus,
	expectedReason condition.Reason,
	expecteMessage string,
) {
	tc.logger.Info("ExpectConditionWithDetails", "type", conditionType, "expected status", expectedStatus, "on", name)
	t.Eventually(func(g t.Gomega) {
		conditions := getter.GetConditions(name)
		g.Expect(conditions).NotTo(
			t.BeNil(), "Status.Conditions in nil")
		g.Expect(conditions.Has(conditionType)).To(
			t.BeTrue(), "Condition type is not in Status.Conditions %s", conditionType)
		actualCondition := conditions.Get(conditionType)
		g.Expect(actualCondition.Status).To(
			t.Equal(expectedStatus),
			"%s condition is in an unexpected state. Expected: %s, Actual: %s",
			conditionType, expectedStatus, actualCondition.Status)
		g.Expect(actualCondition.Reason).To(
			t.Equal(expectedReason),
			"%s condition has a different reason. Actual condition: %v", conditionType, actualCondition)
		g.Expect(actualCondition.Message).To(
			t.Equal(expecteMessage),
			"%s condition has a different message. Actual condition: %v", conditionType, actualCondition)
	}, tc.timeout, tc.interval).Should(t.Succeed())

	tc.logger.Info("ExpectConditionWithDetails succeeded", "type", conditionType, "expected status", expectedStatus, "on", name)
}
