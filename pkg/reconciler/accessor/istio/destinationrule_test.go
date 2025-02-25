/*
Copyright 2021 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

https://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package istio

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	istioclientset "knative.dev/net-istio/pkg/client/istio/clientset/versioned"
	fakeistioclient "knative.dev/net-istio/pkg/client/istio/injection/client/fake"
	fakedrinformer "knative.dev/net-istio/pkg/client/istio/injection/informers/networking/v1alpha3/destinationrule/fake"
	istiolisters "knative.dev/net-istio/pkg/client/istio/listers/networking/v1alpha3"

	. "knative.dev/pkg/reconciler/testing"
)

var (
	originDR = &v1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "dr",
			Namespace:       "default",
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: istiov1alpha3.DestinationRule{
			Host: "origin.example.com",
		},
	}

	desiredDR = &v1alpha3.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "dr",
			Namespace:       "default",
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: istiov1alpha3.DestinationRule{
			Host: "desired.example.com",
		},
	}
)

type FakeDestinatioRuleAccessor struct {
	client   istioclientset.Interface
	drLister istiolisters.DestinationRuleLister
}

func (f *FakeDestinatioRuleAccessor) GetIstioClient() istioclientset.Interface {
	return f.client
}

func (f *FakeDestinatioRuleAccessor) GetDestinationRuleLister() istiolisters.DestinationRuleLister {
	return f.drLister
}

func TestReconcileDestinationRule_Create(t *testing.T) {
	ctx, cancel, informers := SetupFakeContextWithCancel(t)

	istio := fakeistioclient.Get(ctx)
	drInformer := fakedrinformer.Get(ctx)

	waitInformers, err := RunAndSyncInformers(ctx, informers...)
	if err != nil {
		t.Fatal("Failed to start informers")
	}
	defer func() {
		cancel()
		waitInformers()
	}()

	accessor := &FakeDestinatioRuleAccessor{
		client:   istio,
		drLister: drInformer.Lister(),
	}

	h := NewHooks()
	h.OnCreate(&istio.Fake, "destinationrules", func(obj runtime.Object) HookResult {
		got := obj.(*v1alpha3.DestinationRule)
		if diff := cmp.Diff(got, desiredDR); diff != "" {
			t.Log("Unexpected DestinationRule (-want, +got):", diff)
			return HookIncomplete
		}
		return HookComplete
	})

	ReconcileDestinationRule(ctx, ownerObj, desiredDR, accessor)

	if err := h.WaitForHooks(3 * time.Second); err != nil {
		t.Error("Failed to Reconcile DestinationRule:", err)
	}
}

func TestReconcileDestinationRule_Update(t *testing.T) {
	ctx, cancel, informers := SetupFakeContextWithCancel(t)

	istio := fakeistioclient.Get(ctx)
	drInformer := fakedrinformer.Get(ctx)

	waitInformers, err := RunAndSyncInformers(ctx, informers...)
	if err != nil {
		t.Fatal("Failed to start informers")
	}
	defer func() {
		cancel()
		waitInformers()
	}()

	accessor := &FakeDestinatioRuleAccessor{
		client:   istio,
		drLister: drInformer.Lister(),
	}

	istio.NetworkingV1alpha3().DestinationRules(origin.Namespace).Create(ctx, originDR, metav1.CreateOptions{})
	drInformer.Informer().GetIndexer().Add(originDR)

	h := NewHooks()
	h.OnUpdate(&istio.Fake, "destinationrules", func(obj runtime.Object) HookResult {
		got := obj.(*v1alpha3.DestinationRule)
		if diff := cmp.Diff(got, desiredDR); diff != "" {
			t.Log("Unexpected DestinationRule (-want, +got):", diff)
			return HookIncomplete
		}
		return HookComplete
	})

	ReconcileDestinationRule(ctx, ownerObj, desiredDR, accessor)
	if err := h.WaitForHooks(3 * time.Second); err != nil {
		t.Error("Failed to Reconcile DestinationRule:", err)
	}
}
