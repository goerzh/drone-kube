package obj

import (
	"github.com/goerzh/drone-kube/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/util/strategicpatch"
	utilyaml "k8s.io/kubernetes/pkg/util/yaml"
	"log"
)

type Service struct {
	Data   v1.Service
	Patch  string
	Config util.Config
}

func (s *Service) Apply(client *kubernetes.Clientset) error {
	// check and see if there is a deployment already.  If there is, update it.
	oldDep, err := s.find(s.Data.ObjectMeta.Name, s.Data.ObjectMeta.Namespace, client)
	if err != nil {
		return err
	}
	if oldDep.ObjectMeta.Name == s.Data.ObjectMeta.Name {
		originalJS, err := runtime.Encode(unstructured.UnstructuredJSONScheme, &oldDep)
		if err != nil {
			return err
		}

		patchJS, err := utilyaml.ToJSON([]byte(s.Patch))
		if err != nil {
			return err
		}
		scheme := runtime.NewScheme()
		obj, err := scheme.New(oldDep.GetObjectKind().GroupVersionKind())
		if err != nil {
			return err
		}

		patch, err := strategicpatch.StrategicMergePatch(originalJS, patchJS, obj)
		if err != nil {
			return err
		}
		log.Printf("%s\n", patch)

		// update the existing deployment, ignore the deployment that it comes back with
		_, err = client.CoreV1().Services(s.Config.Namespace).Patch(oldDep.Name, api.StrategicMergePatchType, patch)
		return err
	}
	// create the new deployment since this never existed.
	_, err = client.CoreV1().Services(s.Config.Namespace).Create(&s.Data)

	return err
}

func (s *Service) find(svcName string, namespace string, c *kubernetes.Clientset) (v1.Service, error) {
	if namespace == "" {
		namespace = "default"
	}
	var d v1.Service
	services, err := s.list(c, namespace)
	if err != nil {
		return d, err
	}
	for _, thisSvc := range services {
		if thisSvc.ObjectMeta.Name == svcName {
			return thisSvc, err
		}
	}
	return d, err
}

// List the services
func (s *Service) list(clientset *kubernetes.Clientset, namespace string) ([]v1.Service, error) {
	// docs on this:
	// https://github.com/kubernetes/client-go/blob/master/pkg/apis/extensions/types.go
	services, err := clientset.CoreV1().Services(namespace).List(v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}
	return services.Items, err
}
