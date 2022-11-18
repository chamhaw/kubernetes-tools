package v1

import (
	"context"
	"github.com/chamhaw/go-tools/signals"
	"github.com/chamhaw/go-tools/utils"
	defaults "github.com/chamhaw/kubernetes-tools/util/default"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
	"log"
	"os"
	"time"
)

const (
	// DefaultLeaderElect is the default true leader election should be enabled
	DefaultLeaderElect = true

	// DefaultLeaderElectionLeaseDuration is the default time in seconds that non-leader candidates will wait to force acquire leadership
	DefaultLeaderElectionLeaseDuration = 15 * time.Second

	// DefaultLeaderElectionRenewDeadline is the default time in seconds that the acting master will retry refreshing leadership before giving up
	DefaultLeaderElectionRenewDeadline = 10 * time.Second

	// DefaultLeaderElectionRetryPeriod is the default time in seconds that the leader election clients should wait between tries of actions
	DefaultLeaderElectionRetryPeriod = 2 * time.Second

	defaultLeaderElectionLeaseLockName = "default-kubernetes-lock"
)

type LeaderCallback interface {
	OnStartedLeading(ctx context.Context, clusterCode, namespace, labelSelector string, kubeClientSet clientset.Interface) error
	OnStoppedLeading(ctx context.Context)
}

type LeaderElector struct {
	KubeClientSet clientset.Interface
}

type LeaderElectionOptions struct {
	LeaderElect                 bool
	LeaderElectionNamespace     string
	LeaderElectionLeaseDuration time.Duration
	LeaderElectionRenewDeadline time.Duration
	LeaderElectionRetryPeriod   time.Duration
	LeaderElectionLeaseLockName string
	ClusterCode                 string
	Namespace                   string
	LabelSelector               string
}

func (s LeaderElector) Run(ctx context.Context, electOpts LeaderElectionOptions, callback LeaderCallback, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	signals.SetupSignalHandler(cancel)
	if !electOpts.LeaderElect {
		// 传空，则认为是单机，调试用
		klog.Info("Leader election is turned off. Running in single-instance mode")
		go callback.OnStartedLeading(ctx, electOpts.ClusterCode, electOpts.Namespace, electOpts.LabelSelector, s.KubeClientSet)
	} else {

		id, err := os.Hostname()
		if err != nil {
			log.Fatalf("Error getting hostname for leader election %v", err)
		}

		klog.Infof("Leaderelection get id %s, lock key is %s", id, electOpts.LeaderElectionLeaseLockName)
		go leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock: &resourcelock.LeaseLock{
				LeaseMeta: metav1.ObjectMeta{Name: utils.GetOrDefault(electOpts.LeaderElectionLeaseLockName, defaultLeaderElectionLeaseLockName),
					Namespace: utils.GetOrDefault(electOpts.LeaderElectionNamespace, defaults.Namespace())},
				Client:     s.KubeClientSet.CoordinationV1(),
				LockConfig: resourcelock.ResourceLockConfig{Identity: id},
			},
			ReleaseOnCancel: true,
			LeaseDuration:   utils.GetOrDefault(electOpts.LeaderElectionLeaseDuration, DefaultLeaderElectionLeaseDuration),
			RenewDeadline:   utils.GetOrDefault(electOpts.LeaderElectionRenewDeadline, DefaultLeaderElectionRenewDeadline),
			RetryPeriod:     utils.GetOrDefault(electOpts.LeaderElectionRetryPeriod, DefaultLeaderElectionRetryPeriod),
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					err := callback.OnStartedLeading(ctx, electOpts.ClusterCode, electOpts.Namespace, electOpts.LabelSelector, s.KubeClientSet)
					if err != nil {
						cancel()
						os.Exit(2)
					}
				},
				OnStoppedLeading: func() {
					klog.Errorf("stopped leading: %s", id)
					cancel()
					// 理论上不会到这里，意外放弃了leader， 则进程结束，用于触发新一轮选举watch
					// 如果因为下线走到这一步，理论上也不影响进程退出
					os.Exit(3)
				},
				OnNewLeader: func(identity string) {
					// we're notified when new leader elected
					if identity == id {
						// I just got the lock
						return
					}
					klog.Infof("new leader: %s", identity)
				},
			},
		})
	}
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}
