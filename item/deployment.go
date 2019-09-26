package item

import (
	"bytes"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	"k8s.io/api/apps/v1beta1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

type Deployment struct {
	Data   v1beta1.Deployment
	Patch  string
	Config util.Config
}

func NewDeployment(patch string, cfg util.Config) (*Deployment, error) {
	dep := &Deployment{
		Patch:  patch,
		Config: cfg,
	}
	dc := utilyaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(dep.Patch)))
	err := dc.Decode(&dep.Data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dep, nil
}

func (dm *Deployment) Apply(client *kubernetes.Clientset) error {
	// check and see if there is a deployment already.  If there is, update it.
	origin, err := dm.findOrigin(dm.Data.ObjectMeta.Name, dm.Data.ObjectMeta.Namespace, client)
	if err != nil {
		return errors.WithStack(err)
	}

	if origin != nil {
		// update the existing deployment, ignore the deployment that it comes back with
		_, err = client.AppsV1beta1().Deployments(dm.Config.Namespace).Update(&dm.Data)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		// create the new deployment since this never existed.
		_, err = client.AppsV1beta1().Deployments(dm.Config.Namespace).Create(&dm.Data)
		if err != nil {
			return errors.WithStack(err)
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
