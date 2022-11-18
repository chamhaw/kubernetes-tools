package v1

import (
	"context"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher interface {
	DoWatch(ctx context.Context) error
}

type WatchEventHandler interface {
	Handle(ctx context.Context, clusterCode string, event watch.Event) error
}
