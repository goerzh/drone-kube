package item

import (
	"bytes"
	"encoding/json"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	"io"
	"k8s.io/api/apps/v1beta1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"log"
)

type Deployment struct {
	Data   []v1beta1.Deployment
	Patch  string
	Config util.Config
}

func NewDeployment(patch string, cfg util.Config) (*Deployment, error) {
	dep := &Deployment{
		Patch:  patch,
		Config: cfg,
	}
	dc := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(dep.Patch)), 4096)
	for {
		ext := runtime.RawExtension{}
		if err := dc.Decode(&ext); err != nil {
			if err == io.EOF {
				return dep, nil
			}
			return nil, errors.WithStack(err)
		}
		deploy := v1beta1.Deployment{}
		err := json.Unmarshal(ext.Raw, &deploy)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		dep.Data = append(dep.Data, deploy)
	}
}

func (dm *Deployment) Apply(client *kubernetes.Clientset) error {
	for _, deploy := range dm.Data {
		// check and see if there is a deployment already.  If there is, update it.
		origin, err := dm.findOrigin(deploy.ObjectMeta.Name, deploy.Namespace, client)
		if err != nil {
			return errors.WithStack(err)
		}

		if origin != nil {
			// update the existing deployment, ignore the deployment that it comes back with
			_, err = client.AppsV1beta1().Deployments(dm.Config.Namespace).Update(&deploy)
			log.Println("update deployment " + deploy.ObjectMeta.Name)
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			// create the new deployment since this never existed.
			_, err = client.AppsV1beta1().Deployments(dm.Config.Namespace).Create(&deploy)
			log.Println("create deployment " + deploy.ObjectMeta.Name)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

func (dm *Deployment) findOrigin(depName string, namespace string, c *kubernetes.Clientset) (*v1beta1.Deployment, error) {
	if namespace == "" {
		namespace = "default"
	}

	record, err := c.AppsV1beta1().Deployments(namespace).Get(depName, v1.GetOptions{})
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.WithStack(err)
	}
	return record, nil
}
