package item

import (
	"bytes"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
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
	origin, err := s.findOrigin(s.Data.ObjectMeta.Name, s.Data.ObjectMeta.Namespace, client)
	if err != nil {
		return errors.WithStack(err)
	}

	if origin == nil {
		// create the new service since this never existed.
		_, err = client.CoreV1().Services(s.Config.Namespace).Create(&s.Data)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// TODO update service

	return nil
}

func (s *Service) findOrigin(svcName string, namespace string, c *kubernetes.Clientset) (*coreV1.Service, error) {
	if namespace == "" {
		namespace = "default"
	}
	record, err := c.CoreV1().Services(namespace).Get(svcName, metaV1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return record, nil
}
