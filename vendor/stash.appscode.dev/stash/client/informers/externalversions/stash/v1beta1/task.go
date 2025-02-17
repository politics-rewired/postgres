/*
Copyright 2019 The Stash Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1beta1

import (
	time "time"

	stashv1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	versioned "stash.appscode.dev/stash/client/clientset/versioned"
	internalinterfaces "stash.appscode.dev/stash/client/informers/externalversions/internalinterfaces"
	v1beta1 "stash.appscode.dev/stash/client/listers/stash/v1beta1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// TaskInformer provides access to a shared informer and lister for
// Tasks.
type TaskInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta1.TaskLister
}

type taskInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewTaskInformer constructs a new informer for Task type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewTaskInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredTaskInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredTaskInformer constructs a new informer for Task type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredTaskInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.StashV1beta1().Tasks().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.StashV1beta1().Tasks().Watch(options)
			},
		},
		&stashv1beta1.Task{},
		resyncPeriod,
		indexers,
	)
}

func (f *taskInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredTaskInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *taskInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&stashv1beta1.Task{}, f.defaultInformer)
}

func (f *taskInformer) Lister() v1beta1.TaskLister {
	return v1beta1.NewTaskLister(f.Informer().GetIndexer())
}
