// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package workflow

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type TestSuite struct {
	suite.Suite
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func (s *TestSuite) SetupTest() {
	s.scheme = runtime.NewScheme()
	workflows.AddToScheme(s.scheme)
	s.recorder = record.NewFakeRecorder(10)
}
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

type expect struct {
	result any
	err    error
}
type args struct {
	objsForClient       []client.Object
	workflowRunTemplate *workflows.WorkflowRunTemplate
	workflowRuns        []workflows.WorkflowRun
	limit               int
	totalToDelete       int
	revision            string
	runTime             time.Time
}
type testCase struct {
	name   string
	expect expect
	args   args
}

func (s *TestSuite) TestNeedStatusUpdate() {
	conditionReady := metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue}
	conditionNotReady := metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse}

	testCases := []testCase{
		{
			name: "no existing statuses, no new runs, should not need update",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Status: workflows.WorkflowRunTemplateStatus{RunStatuses: []workflows.NamedWorkflowRunStatus{}},
				},
				workflowRuns: []workflows.WorkflowRun{},
			},
			expect: expect{result: false},
		},
		{
			name: "no existing statuses, new run, should need update",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Status: workflows.WorkflowRunTemplateStatus{RunStatuses: []workflows.NamedWorkflowRunStatus{}},
				},
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{Name: "run-1"}, Status: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "existing statuses, same runs and conditions, should not need update",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Status: workflows.WorkflowRunTemplateStatus{
						RunStatuses: []workflows.NamedWorkflowRunStatus{
							{RunName: "run-1", WorkflowRunStatus: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
						},
					},
				},
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{Name: "run-1"}, Status: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
				},
			},
			expect: expect{result: false},
		},
		{
			name: "existing statuses, run with different conditions, should need update",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Status: workflows.WorkflowRunTemplateStatus{
						RunStatuses: []workflows.NamedWorkflowRunStatus{
							{RunName: "run-1", WorkflowRunStatus: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
						},
					},
				},
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{Name: "run-1"}, Status: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionNotReady}}},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "existing statuses, different number of runs, should need update",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Status: workflows.WorkflowRunTemplateStatus{
						RunStatuses: []workflows.NamedWorkflowRunStatus{
							{RunName: "run-1", WorkflowRunStatus: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
						},
					},
				},
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{Name: "run-1"}, Status: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
					{ObjectMeta: metav1.ObjectMeta{Name: "run-2"}, Status: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "existing statuses, fewer runs than before, should need update",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Status: workflows.WorkflowRunTemplateStatus{
						RunStatuses: []workflows.NamedWorkflowRunStatus{
							{RunName: "run-1", WorkflowRunStatus: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
							{RunName: "run-2", WorkflowRunStatus: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
						},
					},
				},
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{Name: "run-1"}, Status: workflows.WorkflowRunStatus{Conditions: []metav1.Condition{conditionReady}}},
				},
			},
			expect: expect{result: true},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			reconciler := &RunTemplateReconciler{}
			needsUpdate := reconciler.needsStatusUpdate(tc.args.workflowRunTemplate, tc.args.workflowRuns)
			assert.Equal(s.T(), tc.expect.result, needsUpdate)
		})
	}
}

func (s *TestSuite) TestShouldReconcile() {
	now := metav1.Now()
	tenMinutesAgo := metav1.NewTime(now.Time.Add(-10 * time.Minute))

	testCases := []testCase{
		{
			name: "Once: should reconcile if SyncedResourceVersion is empty",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec:   workflows.WorkflowRunTemplateSpec{RunPolicy: workflows.RunPolicy{Once: &workflows.RunPolicyOnce{}}},
					Status: workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: ""},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "Once: should not reconcile if SyncedResourceVersion is not empty",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec:   workflows.WorkflowRunTemplateSpec{RunPolicy: workflows.RunPolicy{Once: &workflows.RunPolicyOnce{}}},
					Status: workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: "1"},
				},
			},
			expect: expect{result: false},
		},
		{
			name: "OnChange: should reconcile if SyncedResourceVersion is different",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					ObjectMeta: metav1.ObjectMeta{ResourceVersion: "2"},
					Spec:       workflows.WorkflowRunTemplateSpec{RunPolicy: workflows.RunPolicy{OnChange: &workflows.RunPolicyOnChange{}}},
					Status:     workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: "1"},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "Scheduled: should reconcile if syncedResourceVersion is Nil",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{Every: &metav1.Duration{Duration: 5 * time.Minute}},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{LastRunTime: &tenMinutesAgo},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "Scheduled: should reconcile if syncedResourceVersion isnt matching",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{Every: &metav1.Duration{Duration: 5 * time.Minute}},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: "1", LastRunTime: &tenMinutesAgo},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "Scheduled: should reconcile if no LastRunTime ",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{Every: &metav1.Duration{Duration: 5 * time.Minute}},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: "0-f47fe3c0b255b6dd8047cdffa772587bb829efe7a1cb70febeda2eb2"},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "Scheduled/Every: should reconcile if time has passed",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{Every: &metav1.Duration{Duration: 5 * time.Minute}},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: "0-f47fe3c0b255b6dd8047cdffa772587bb829efe7a1cb70febeda2eb2", LastRunTime: &tenMinutesAgo},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "Scheduled/Cron: should not reconcile if next run is in the future",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{Cron: stringPtr("*/20 * * * *")},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: "0-f47fe3c0b255b6dd8047cdffa772587bb829efe7a1cb70febeda2eb2", LastRunTime: &now},
				},
			},
			expect: expect{result: false},
		},
		{
			name: "Scheduled/Cron: should reconcile if next run is in the past",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{Cron: stringPtr("*/5 * * * *")},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: "0-f47fe3c0b255b6dd8047cdffa772587bb829efe7a1cb70febeda2eb2", LastRunTime: &tenMinutesAgo},
				},
			},
			expect: expect{result: true},
		},
		{
			name: "Scheduled/Cron: should not reconcile if cron is invalid",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{Cron: stringPtr("badcron")},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{SyncedResourceVersion: "0-f47fe3c0b255b6dd8047cdffa772587bb829efe7a1cb70febeda2eb2", LastRunTime: &tenMinutesAgo},
				},
			},
			expect: expect{result: false},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			reconciler := &RunTemplateReconciler{Log: logr.Discard()}
			should := reconciler.shouldReconcile(tc.args.workflowRunTemplate)
			assert.Equal(s.T(), tc.expect.result, should)
		})
	}
}

func (s *TestSuite) TestRequeueAfter() {
	now := metav1.Now()
	tenMinutesAgo := metav1.NewTime(now.Time.Add(-10 * time.Minute))

	testCases := []testCase{
		{
			name: "should not requeue for Once policy",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{Once: &workflows.RunPolicyOnce{}},
					},
				},
			},
			expect: expect{
				result: ctrl.Result{},
			},
		},
		{
			name: "should not requeue for OnChange policy",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{OnChange: &workflows.RunPolicyOnChange{}},
					},
				},
			},
			expect: expect{
				result: ctrl.Result{},
			},
		},
		{
			name: "should requeue immediately if scheduled time has passed",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{
								Every: &metav1.Duration{Duration: 5 * time.Minute},
							},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{
						LastRunTime: &tenMinutesAgo,
					},
				},
			},
			expect: expect{
				result: ctrl.Result{Requeue: true},
			},
		},
		{
			name: "should requeue after remaining time",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{
								Every: &metav1.Duration{Duration: 15 * time.Minute},
							},
						},
					},
					Status: workflows.WorkflowRunTemplateStatus{
						LastRunTime: &tenMinutesAgo,
					},
				},
			},
			expect: expect{
				result: ctrl.Result{RequeueAfter: 5 * time.Minute},
			},
		},
		// TODO[gusfcarvalho]
		// We need to improve cron tests as they add a lot of noise to the requeueAfter
		// This forces us to have a bigger delta in the test case checks;
		{
			name: "should requeue based on cron schedule",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					Status: workflows.WorkflowRunTemplateStatus{
						LastRunTime: &tenMinutesAgo,
					},
					Spec: workflows.WorkflowRunTemplateSpec{
						RunPolicy: workflows.RunPolicy{
							Scheduled: &workflows.RunPolicyScheduled{
								Cron: stringPtr("*/10 * * * *"),
							},
						},
					},
				},
			},
			expect: expect{
				result: ctrl.Result{RequeueAfter: 10 * time.Minute},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			reconciler := &RunTemplateReconciler{Log: logr.Discard()}
			result, err := reconciler.requeueAfter(tc.args.workflowRunTemplate)

			require.NoError(s.T(), err)
			if tc.expect.result.(ctrl.Result).RequeueAfter > 0 {
				assert.InDelta(s.T(), tc.expect.result.(ctrl.Result).RequeueAfter, result.RequeueAfter, float64(10*time.Minute))
			} else {
				assert.Equal(s.T(), tc.expect.result, result)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func (s *TestSuite) TestGenerateRevision() {
	testCases := []testCase{
		{
			name: "should return 1 if no workflow runs",
			args: args{
				workflowRuns: []workflows.WorkflowRun{},
			},
			expect: expect{
				result: "1",
			},
		},
		{
			name: "should return highest revision + 1",
			args: args{
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "1"}}},
					{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "5"}}},
					{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "2"}}},
				},
			},
			expect: expect{
				result: "6",
			},
		},
		{
			name: "should return error if revision annotation is missing",
			args: args{
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{Name: "run-no-rev"}},
				},
			},
			expect: expect{
				err: assert.AnError,
			},
		},
		{
			name: "should return error if revision is not an integer",
			args: args{
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{Name: "run-bad-rev", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "not-a-number"}}},
				},
			},
			expect: expect{
				err: assert.AnError,
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			reconciler := &RunTemplateReconciler{}
			revision, err := reconciler.generateRevision(tc.args.workflowRuns)

			if tc.expect.err != nil {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				assert.Equal(s.T(), tc.expect.result, revision)
			}
		})
	}
}

func (s *TestSuite) TestCreateWorkflowRun() {
	testCases := []testCase{
		{
			name: "should create a new workflowrun",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-wrt",
						Namespace: "test-ns",
					},
					Spec: workflows.WorkflowRunTemplateSpec{
						RunSpec: workflows.WorkflowRunSpec{
							TemplateRef: workflows.TemplateRef{
								Name: "test-template",
							},
						},
					},
				},
				revision: "1",
			},
			expect: expect{
				result: &workflows.WorkflowRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
						Annotations: map[string]string{
							"workflowruntemplate.external-secrets.io/revision": "1",
						},
						Labels: map[string]string{
							"workflowruntemplate.external-secrets.io/owner": "test-wrt",
						},
					},
					Spec: workflows.WorkflowRunSpec{
						TemplateRef: workflows.TemplateRef{
							Name: "test-template",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cl := fake.NewClientBuilder().WithScheme(s.scheme).Build()
			reconciler := &RunTemplateReconciler{
				Client:   cl,
				Log:      logr.Discard(),
				Scheme:   s.scheme,
				Recorder: s.recorder,
			}

			createdRun, err := reconciler.createWorkflowRun(context.TODO(), tc.args.workflowRunTemplate, tc.args.revision)
			require.NoError(s.T(), err)

			var workflowRuns workflows.WorkflowRunList
			err = cl.List(context.TODO(), &workflowRuns)
			require.NoError(s.T(), err)
			require.Len(s.T(), workflowRuns.Items, 1)

			run := workflowRuns.Items[0]
			expectedRun := tc.expect.result.(*workflows.WorkflowRun)

			assert.Contains(s.T(), run.Name, "test-wrt-")
			assert.Equal(s.T(), expectedRun.Namespace, run.Namespace)
			assert.Equal(s.T(), expectedRun.Annotations, run.Annotations)
			assert.Equal(s.T(), expectedRun.Labels, run.Labels)
			assert.Equal(s.T(), expectedRun.Spec, run.Spec)
			assert.NotEmpty(s.T(), createdRun.OwnerReferences)
		})
	}
}

func (s *TestSuite) TestUpdateWorkflowRunTemplate() {
	now := time.Now().Round(time.Second)
	testCases := []testCase{
		{
			name: "should update the workflowruntemplate status",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test",
						Namespace:       "test",
						ResourceVersion: "1",
					},
				},
				objsForClient: []client.Object{
					&workflows.WorkflowRunTemplate{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "workflows.external-secrets.io/v1alpha1",
							Kind:       "WorkflowRunTemplate",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test",
							Namespace:       "test",
							ResourceVersion: "1",
						},
					},
				},
				workflowRuns: []workflows.WorkflowRun{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "run-1"},
						Status:     workflows.WorkflowRunStatus{Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}},
					},
				},
				runTime: now,
			},
			expect: expect{
				result: &workflows.WorkflowRunTemplate{
					Status: workflows.WorkflowRunTemplateStatus{
						RunStatuses: []workflows.NamedWorkflowRunStatus{
							{
								RunName: "run-1",
								WorkflowRunStatus: workflows.WorkflowRunStatus{
									Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
								},
							},
						},
						LastRunTime:           &metav1.Time{Time: now},
						SyncedResourceVersion: "0-f47fe3c0b255b6dd8047cdffa772587bb829efe7a1cb70febeda2eb2",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cl := fake.NewClientBuilder().WithScheme(s.scheme).WithObjects(tc.args.objsForClient...).WithStatusSubresource(tc.args.objsForClient...).Build()
			reconciler := &RunTemplateReconciler{
				Client:   cl,
				Log:      logr.Discard(),
				Scheme:   s.scheme,
				Recorder: s.recorder,
			}

			err := reconciler.updateWorkflowRunTemplate(context.TODO(), tc.args.workflowRunTemplate, tc.args.workflowRuns, tc.args.runTime)
			require.NoError(s.T(), err)

			var updatedWrt workflows.WorkflowRunTemplate
			err = cl.Get(context.TODO(), types.NamespacedName{Name: "test", Namespace: "test"}, &updatedWrt)
			require.NoError(s.T(), err)

			expectedWrt := tc.expect.result.(*workflows.WorkflowRunTemplate)

			assert.Equal(s.T(), expectedWrt.Status.RunStatuses, updatedWrt.Status.RunStatuses)
			fmt.Printf("now: %s\nlast run: %s", now.String(), updatedWrt.Status.LastRunTime.Time.String())
			assert.True(s.T(), now.Equal(updatedWrt.Status.LastRunTime.Time))
			assert.Equal(s.T(), expectedWrt.Status.SyncedResourceVersion, updatedWrt.Status.SyncedResourceVersion)
		})
	}
}

func (s *TestSuite) TestCleanup() {
	testCases := []testCase{
		{
			name: "should not delete anything if under limit",
			args: args{
				objsForClient: []client.Object{
					&workflows.WorkflowRun{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1", Name: "run-1", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "1"}}},
					&workflows.WorkflowRun{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1", Name: "run-2", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "2"}}},
				},
				limit: 3,
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1", Name: "run-1", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "1"}}},
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1", Name: "run-2", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "2"}}},
				},
			},
			expect: expect{
				result: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1", Name: "run-1", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "1"}}},
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1", Name: "run-2", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "2"}}},
				},
			},
		},
		{
			name: "should delete oldest revision",
			args: args{
				objsForClient: []client.Object{
					&workflows.WorkflowRun{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "4", Name: "run-4", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "1"}}},
					&workflows.WorkflowRun{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "5", Name: "run-5", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "2"}}},
					&workflows.WorkflowRun{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "6", Name: "run-6", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "3"}}},
				},
				limit: 3,
				workflowRuns: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "4", Name: "run-4", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "1"}}},
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "5", Name: "run-5", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "2"}}},
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "6", Name: "run-6", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "3"}}},
				},
			},
			expect: expect{
				result: []workflows.WorkflowRun{
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "5", Name: "run-5", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "2"}}},
					{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "6", Name: "run-6", Annotations: map[string]string{"workflowruntemplate.external-secrets.io/revision": "3"}}},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			client := fake.NewClientBuilder().WithScheme(s.scheme).WithObjects(tc.args.objsForClient...).Build()
			reconciler := &RunTemplateReconciler{
				Client:   client,
				Log:      logr.Discard(),
				Scheme:   s.scheme,
				Recorder: s.recorder,
			}

			result, err := reconciler.cleanup(context.Background(), tc.args.workflowRuns, tc.args.limit)

			if tc.expect.err != nil {
				require.Error(s.T(), err)
				assert.ErrorContains(s.T(), err, tc.expect.err.Error())
			} else {
				require.NoError(s.T(), err)
			}

			if tc.expect.result != nil {
				expected := tc.expect.result.([]workflows.WorkflowRun)
				sort.Slice(result, func(i, j int) bool {
					return result[i].Name < result[j].Name
				})
				sort.Slice(expected, func(i, j int) bool {
					return expected[i].Name < expected[j].Name
				})
				assert.Equal(s.T(), expected, result)
			}

			// Check if the correct workflow was deleted
			if len(tc.args.workflowRuns) > tc.args.limit {
				deletedRunName := "run-1"
				wfr := &workflows.WorkflowRun{}
				err := reconciler.Get(context.Background(), types.NamespacedName{Name: deletedRunName}, wfr)
				assert.True(s.T(), apierrors.IsNotFound(err))
			}
		})
	}
}

func (s *TestSuite) TestSortWorkflowsByRevision() {
	testCases := []testCase{
		{
			name: "should sort by revision",
			args: args{
				totalToDelete: 1,
				workflowRuns: []workflows.WorkflowRun{
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"workflowruntemplate.external-secrets.io/revision": "1",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"workflowruntemplate.external-secrets.io/revision": "2",
							},
						},
					},
				},
			},
			expect: expect{
				result: []any{
					[]workflows.WorkflowRun{
						{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									"workflowruntemplate.external-secrets.io/revision": "2",
								},
							},
						},
					},
					[]workflows.WorkflowRun{
						{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: map[string]string{
									"workflowruntemplate.external-secrets.io/revision": "1",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should return error on missing annotation",
			args: args{
				totalToDelete: 1,
				workflowRuns: []workflows.WorkflowRun{
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{},
						},
					},
				},
			},
			expect: expect{
				err: assert.AnError,
			},
		},
		{
			name: "should return error on revision not being an integer",
			args: args{
				totalToDelete: 1,
				workflowRuns: []workflows.WorkflowRun{
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"workflowruntemplate.external-secrets.io/revision": "not-an-integer",
							},
						},
					},
				},
			},
			expect: expect{
				err: assert.AnError,
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			active, toBeDeleted, err := sortWorkflowsByRevision(tc.args.workflowRuns, tc.args.totalToDelete)

			if tc.expect.err != nil {
				assert.Error(s.T(), err)
			} else {
				assert.NoError(s.T(), err)
			}

			if tc.expect.result != nil {
				expectedResult := tc.expect.result.([]any)
				assert.Equal(s.T(), expectedResult[0], active)
				assert.Equal(s.T(), expectedResult[1], toBeDeleted)
			}
		})
	}
}

func (s *TestSuite) TestGetChildrenFor() {
	testCases := []testCase{
		{
			name: "no objects",
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
		},
		{
			name: "three objects",
			expect: expect{
				result: []workflows.WorkflowRun{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-1",
							Namespace:       "test",
							ResourceVersion: "999",
							Labels: map[string]string{
								"workflowruntemplate.external-secrets.io/owner": "test",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-2",
							Namespace:       "test",
							ResourceVersion: "999",
							Labels: map[string]string{
								"workflowruntemplate.external-secrets.io/owner": "test",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-3",
							Namespace:       "test",
							ResourceVersion: "999",
							Labels: map[string]string{
								"workflowruntemplate.external-secrets.io/owner": "test",
							},
						},
					},
				},
			},
			args: args{
				workflowRunTemplate: &workflows.WorkflowRunTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
				objsForClient: []client.Object{
					&workflows.WorkflowRun{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-1",
							Namespace:       "test",
							ResourceVersion: "999",
							Labels: map[string]string{
								"workflowruntemplate.external-secrets.io/owner": "test",
							},
						},
					},
					&workflows.WorkflowRun{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-2",
							Namespace:       "test",
							ResourceVersion: "999",
							Labels: map[string]string{
								"workflowruntemplate.external-secrets.io/owner": "test",
							},
						},
					},
					&workflows.WorkflowRun{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-3",
							Namespace:       "test",
							ResourceVersion: "999",
							Labels: map[string]string{
								"workflowruntemplate.external-secrets.io/owner": "test",
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			client := fake.NewClientBuilder().WithScheme(s.scheme).WithObjects(tc.args.objsForClient...).Build()
			reconciler := &RunTemplateReconciler{
				Client:   client,
				Log:      logr.Discard(),
				Scheme:   s.scheme,
				Recorder: s.recorder,
			}
			result, err := reconciler.getChildrenFor(context.TODO(), tc.args.workflowRunTemplate)
			if tc.expect.err != nil {
				require.Error(s.T(), err)
				assert.ErrorContains(s.T(), err, tc.expect.err.Error())
			}
			if tc.expect.result != nil {
				require.IsType(s.T(), result, tc.expect.result)
				assert.Equal(s.T(), result, tc.expect.result)
			}
		})
	}
}
