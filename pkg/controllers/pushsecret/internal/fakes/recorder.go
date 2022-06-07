package fakes

import "k8s.io/apimachinery/pkg/runtime"

type FakeRecorder struct {
	EventCallCounter int
}

func (r *FakeRecorder) Event(object runtime.Object, eventtype string, reason string, message string) {
	r.EventCallCounter += 1
}
func (r FakeRecorder) EventCallCount() int { return r.EventCallCounter }
func (r FakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
}
func (r FakeRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
}
