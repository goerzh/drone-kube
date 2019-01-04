package obj

import (
	"bytes"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	utilyaml "k8s.io/kubernetes/pkg/util/yaml"
	"log"
)

type Service struct {
	Data   v1.Service
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
	if old != nil {
		// create the new deployment since this never existed.
		_, err = client.CoreV1().Services(s.Config.Namespace).Create(&s.Data)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// TODO update service

	return nil
}

func (s *Service) find(svcName string, namespace string, c *kubernetes.Clientset) (*v1.Service, error) {
	if namespace == "" {
		namespace = "default"
	}
	var d *v1.Service
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
func (s *Service) list(clientset *kubernetes.Clientset, namespace string) ([]v1.Service, error) {
	// docs on this:
	// https://github.com/kubernetes/client-go/blob/master/pkg/apis/extensions/types.go
	services, err := clientset.CoreV1().Services(namespace).List(v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}
	return services.Items, nil
}
