package main

import (
	"bytes"
	"encoding/base64"
	"github.com/goerzh/drone-kube/item"
	"github.com/goerzh/drone-kube/util"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"log"
)

type (
	Repo struct {
		Owner string
		Name  string
	}

	Build struct {
		Tag     string
		Event   string
		Number  int
		Commit  string
		Ref     string
		Branch  string
		Author  string
		Status  string
		Link    string
		Started int64
		Created int64
	}

	Job struct {
		Started int64
	}

	Plugin struct {
		Repo   Repo
		Build  Build
		Config util.Config
		Job    Job
	}
)

func (p *Plugin) Exec() error {

	if p.Config.Server == "" {
		log.Fatal("KUBE_SERVER is not defined")
	}
	if p.Config.Token == "" {
		log.Fatal("KUBE_TOKEN is not defined")
	}
	if p.Config.Ca == "" {
		log.Fatal("KUBE_CA is not defined")
	}
	if p.Config.Namespace == "" {
		p.Config.Namespace = "default"
	}
	if p.Config.Template == "" {
		log.Fatal("KUBE_TEMPLATE or template must be defined")
	}

	// connect to Kubernetes
	clientset, err := p.createKubeClient()
	if err != nil {
		log.Fatal(err.Error())
	}

	patch, err := openAndSub(p.Config.Template, p)
	if err != nil {
		return errors.WithStack(err)
	}
	dep, err := item.NewDeployment(patch, p.Config)
	if err != nil {
		return errors.WithStack(err)
	}
	if err = dep.Apply(clientset); err != nil {
		return errors.WithStack(err)
	}

	// apply service
	if p.Config.Service != "" {
		patch, err = openAndSub(p.Config.Service, p)
		if err != nil {
			return errors.WithStack(err)
		}
		svc, err := item.NewService(patch, p.Config)
		if err != nil {
			return errors.WithStack(err)
		}

		err = svc.Apply(clientset)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	//apply ingress
	if p.Config.Ingress != "" {
		patch, err = openAndSub(p.Config.Ingress, p)
		if err != nil {
			return errors.WithStack(err)
		}
		svc, err := item.NewIngress(patch, p.Config)
		if err != nil {
			return errors.WithStack(err)
		}

		err = svc.Apply(clientset)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return err
}

func (p *Plugin) decodeYamlToObjects(fName string, objects ...interface{}) error {
	// parse the template file and do substitutions
	txt, err := openAndSub(fName, p)
	if err != nil {
		return errors.WithStack(err)
	}
	dc := yaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(txt)))
	for _, o := range objects {
		err = dc.Decode(&o)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// open up the template and then sub variables in. Handlebar stuff.
func openAndSub(templateFile string, p *Plugin) (string, error) {
	t, err := ioutil.ReadFile(templateFile)
	if err != nil {
		return "", err
	}
	//potty humor!  Render trim toilet paper!  Ha ha, so funny.
	return util.RenderTrim(string(t), p)
}

// create the connection to kubernetes based on parameters passed in.
// the kubernetes/client-go project is really hard to understand.
func (p Plugin) createKubeClient() (*kubernetes.Clientset, error) {

	ca, err := base64.StdEncoding.DecodeString(p.Config.Ca)
	config := clientcmdapi.NewConfig()
	config.Clusters["drone"] = &clientcmdapi.Cluster{
		Server:                   p.Config.Server,
		CertificateAuthorityData: ca,
	}
	config.AuthInfos["drone"] = &clientcmdapi.AuthInfo{
		Token: p.Config.Token,
	}

	config.Contexts["drone"] = &clientcmdapi.Context{
		Cluster:  "drone",
		AuthInfo: "drone",
	}
	//config.Clusters["drone"].CertificateAuthorityData = ca
	config.CurrentContext = "drone"

	clientBuilder := clientcmd.NewNonInteractiveClientConfig(*config, "drone", &clientcmd.ConfigOverrides{}, nil)
	actualCfg, err := clientBuilder.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}

	return kubernetes.NewForConfig(actualCfg)
}
