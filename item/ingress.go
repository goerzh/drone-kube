package item

import (
	"bytes"
	"encoding/json"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	"io"
	extendV1beta1 "k8s.io/api/extensions/v1beta1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"log"
)

type Ingress struct {
	Data   []extendV1beta1.Ingress
	Patch  string
	Config util.Config
}

func NewIngress(patch string, cfg util.Config) (*Ingress, error) {
	ig := &Ingress{
		Patch:  patch,
		Config: cfg,
	}
	dc := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(ig.Patch)), 4096)
	for {
		ext := runtime.RawExtension{}
		if err := dc.Decode(&ext); err != nil {
			if err == io.EOF {
				return ig, nil
			}
			return nil, errors.WithStack(err)
		}
		ingress := extendV1beta1.Ingress{}
		err := json.Unmarshal(ext.Raw, &ingress)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		ig.Data = append(ig.Data, ingress)
	}
}

func (ig *Ingress) Apply(client *kubernetes.Clientset) error {
	for _, ingress := range ig.Data {
		// check and see if there is a deployment already.  If there is, update it.
		origin, err := ig.findOrigin(ingress.Name, ingress.Namespace, client)
		if err != nil {
			return errors.WithStack(err)
		}
		if origin != nil {
			// update the existing deployment, ignore the deployment that it comes back with
			_, err = client.ExtensionsV1beta1().Ingresses(ig.Config.Namespace).Update(&ingress)
			if err != nil {
				return errors.WithStack(err)
			}
			log.Println("update ingress " + ingress.Name)
		} else {
			// create the new deployment since this never existed.
			_, err = client.ExtensionsV1beta1().Ingresses(ig.Config.Namespace).Create(&ingress)
			if err != nil {
				return errors.WithStack(err)
			}
			log.Println("create ingress " + ingress.Name)
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
