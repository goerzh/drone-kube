package item

import "k8s.io/client-go/kubernetes"

type Item interface {
	Apply(client *kubernetes.Clientset) error
}
