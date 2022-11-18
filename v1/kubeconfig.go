package v1

import (
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func RestConfig(kubeConfig string) (*restclient.Config, error) {
	config, err := clientcmd.NewClientConfigFromBytes([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}
	return restConfig, nil
}
