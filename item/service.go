package item

import (
	"bytes"
	"encoding/json"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	"io"
	coreV1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"log"
)

type Service struct {
	Data   []coreV1.Service
	Patch  string
	Config util.Config
}

func NewService(patch string, cfg util.Config) (*Service, error) {
	svc := &Service{
		Patch:  patch,
		Config: cfg,
	}
	dc := utilyaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(svc.Patch)))
	for {
		ext := runtime.RawExtension{}
		if err := dc.Decode(&ext); err != nil {
			if err == io.EOF {
				return svc, nil
			}
			return nil, errors.WithStack(err)
		}
		service := coreV1.Service{}
		err := json.Unmarshal(ext.Raw, &service)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		svc.Data = append(svc.Data, service)
	}
}

func (s *Service) Apply(client *kubernetes.Clientset) error {
	for _, service := range s.Data {
		// check and see if there is a deployment already.  If there is, update it.
		origin, err := s.findOrigin(service.ObjectMeta.Name, service.ObjectMeta.Namespace, client)
		if err != nil {
			return errors.WithStack(err)
		}

		if origin == nil {
			// create the new service since this never existed.
			_, err = client.CoreV1().Services(s.Config.Namespace).Create(&service)
			if err != nil {
				return errors.WithStack(err)
			}
			log.Println("create service " + service.ObjectMeta.Name)
		}

		// TODO update service
	}

	return nil
}

func (s *Service) findOrigin(svcName string, namespace string, c *kubernetes.Clientset) (*coreV1.Service, error) {
	if namespace == "" {
		namespace = "default"
	}
	record, err := c.CoreV1().Services(namespace).Get(svcName, metaV1.GetOptions{})
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.WithStack(err)
	}
	return record, nil
}
