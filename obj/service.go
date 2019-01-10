package obj

import (
	"bytes"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"log"
)

type Service struct {
	Data   coreV1.Service
	Patch  string
	Config util.Config
}

func NewService(patch string, cfg util.Config) (*Service, error) {
	svc := &Service{
		Patch:  patch,
		Config: cfg,
	}
	dc := utilyaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(svc.Patch)))
	err := dc.Decode(&svc.Data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return svc, nil
}

func (s *Service) Apply(client *kubernetes.Clientset) error {
	// check and see if there is a deployment already.  If there is, update it.
	old, err := s.find(s.Data.ObjectMeta.Name, s.Data.ObjectMeta.Namespace, client)
	if err != nil {
		return errors.WithStack(err)
	}
	if old == nil {
		// create the new service since this never existed.
		_, err = client.CoreV1().Services(s.Config.Namespace).Create(&s.Data)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// TODO update service

	return nil
}

func (s *Service) find(svcName string, namespace string, c *kubernetes.Clientset) (*coreV1.Service, error) {
	if namespace == "" {
		namespace = "default"
	}
	var d *coreV1.Service
	services, err := s.list(c, namespace)
	if err != nil {
		return d, errors.WithStack(err)
	}
	for _, thisSvc := range services {
		if thisSvc.ObjectMeta.Name == svcName {
			return &thisSvc, nil
		}
	}
	return d, nil
}

// List the services
func (s *Service) list(clientset *kubernetes.Clientset, namespace string) ([]coreV1.Service, error) {
	// docs on this:
	// https://github.com/kubernetes/client-go/blob/master/pkg/apis/extensions/types.go
	services, err := clientset.CoreV1().Services(namespace).List(metaV1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}
	return services.Items, nil
}
