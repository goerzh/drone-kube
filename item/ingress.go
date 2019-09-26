package item

import (
	"bytes"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	extendV1beta1 "k8s.io/api/extensions/v1beta1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

type Ingress struct {
	Data   extendV1beta1.Ingress
	Patch  string
	Config util.Config
}

func NewIngress(patch string, cfg util.Config) (*Ingress, error) {
	ig := &Ingress{
		Patch:  patch,
		Config: cfg,
	}
	dc := utilyaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(ig.Patch)))
	err := dc.Decode(&ig.Data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return ig, nil
}

func (ig *Ingress) Apply(client *kubernetes.Clientset) error {
	// check and see if there is a deployment already.  If there is, update it.
	origin, err := ig.findOrigin(ig.Data.Name, ig.Data.Namespace, client)
	if err != nil {
		return errors.WithStack(err)
	}
	if origin != nil {
		// update the existing deployment, ignore the deployment that it comes back with
		_, err = client.ExtensionsV1beta1().Ingresses(ig.Config.Namespace).Update(&ig.Data)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		// create the new deployment since this never existed.
		_, err = client.ExtensionsV1beta1().Ingresses(ig.Config.Namespace).Create(&ig.Data)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (ig *Ingress) findOrigin(igName string, namespace string, c *kubernetes.Clientset) (*extendV1beta1.Ingress, error) {
	if namespace == "" {
		namespace = "default"
	}

	record, err := c.ExtensionsV1beta1().Ingresses(namespace).Get(igName, v1.GetOptions{})
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.WithStack(err)
	}
	return record, nil
}
