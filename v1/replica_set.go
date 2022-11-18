package v1

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appclientv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/klog/v2"
)

type ReplicaSet interface {
	CreateOrReplace(ctx context.Context, rs *appsv1.ReplicaSet) (*appsv1.ReplicaSet, error)
}

type ReplicaSetClient struct {
	appclientv1.ReplicaSetInterface
	WatchEventHandler
	Namespace     string
	ClusterCode   string
	LabelSelector string
}

func NewReplicaSetClient(rsClient appclientv1.ReplicaSetInterface, clusterCode, namespace, labelSelector string, eventHandler WatchEventHandler) *ReplicaSetClient {
	return &ReplicaSetClient{
		ReplicaSetInterface: rsClient,
		WatchEventHandler:   eventHandler,
		Namespace:           namespace,
		ClusterCode:         clusterCode,
		LabelSelector:       labelSelector,
	}
}

func (r *ReplicaSetClient) CreateOrReplace(ctx context.Context, rs *appsv1.ReplicaSet) (*appsv1.ReplicaSet, error) {
	return r.Update(ctx, rs, metav1.UpdateOptions{})
}

func (r *ReplicaSetClient) DoWatch(ctx context.Context) error {
	w, err := r.Watch(ctx, metav1.ListOptions{
		LabelSelector: r.LabelSelector,
	})

	if err != nil {
		return err
	}

	go func() {
		select {
		case <-ctx.Done():
			w.Stop()
		}
	}()

	go func() {
		for event := range w.ResultChan() {
			err = r.Handle(ctx, r.ClusterCode, event)
			if err != nil {
				klog.Errorf("handle event error. Event: %s.", event.Type)
			}
		}
		klog.Infof("ReplicaSets watch chan of namespace: %s/%s was closed.", r.ClusterCode, r.Namespace)
	}()
	return nil
}
