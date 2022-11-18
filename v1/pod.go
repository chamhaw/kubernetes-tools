package v1

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
)

type Pod interface {
}

type PodClient struct {
	coreclientv1.PodInterface
	WatchEventHandler
	Namespace     string
	ClusterCode   string
	LabelSelector string
}

func NewPodClient(v1Client coreclientv1.PodInterface, clusterCode, namespace, labelSelector string, eventHandler WatchEventHandler) *PodClient {
	return &PodClient{
		PodInterface:      v1Client,
		WatchEventHandler: eventHandler,
		Namespace:         namespace,
		ClusterCode:       clusterCode,
		LabelSelector:     labelSelector,
	}
}

func (r *PodClient) DoWatch(ctx context.Context) error {
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
		klog.Infof("Pods watch chan: %s/%s was closed.", r.ClusterCode, r.Namespace)
	}()
	return nil
}

func PodConditionStatus(pod *v1.Pod) (v1.ConditionStatus, v1.ConditionStatus, v1.ConditionStatus, v1.ConditionStatus) {
	var initialized, ready, containersReady, podScheduled v1.ConditionStatus
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodInitialized {
			initialized = condition.Status
		}
		if condition.Type == v1.PodReady {
			ready = condition.Status
		}
		if condition.Type == v1.ContainersReady {
			containersReady = condition.Status
		}
		if condition.Type == v1.PodScheduled {
			podScheduled = condition.Status
		}
	}
	return initialized, ready, containersReady, podScheduled
}
