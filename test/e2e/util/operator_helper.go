package util

import (
	"bufio"
	"github.com/jboss-dockerfiles/infinispan-server-operator/pkg/launcher"
	osv1 "github.com/openshift/api/authorization/v1"
	"k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"strings"
)

// Provides methods to install and uninstall the operator and dependencies.
func RunOperator(okd *ExternalOKD, ns, config string) chan struct{} {
	installRBAC(ns, okd)
	installCRD(okd)
	stopCh := make(chan struct{})
	go runOperatorLocally(ns, config, stopCh)
	return stopCh
}

func Cleanup(client ExternalOKD, namespace string, stopCh chan struct{}) {
	close(stopCh)
	client.DeleteProject(namespace)
	client.DeleteCRD("infinispans.infinispan.org")
}

func InstallConfigMap(ns string, name string, okd *ExternalOKD) {
	file := GetFilePath("../../config/cloud-ephemeral.xml")
	okd.CreateOrUpdateConfigMap(name, file, ns)
}

// Install resources from rbac.yaml required by the Infinispan operator
func installRBAC(ns string, okd *ExternalOKD) {
	filename := GetFilePath("../../deploy/rbac.yaml")
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	yamlReader := yaml.NewYAMLReader(bufio.NewReader(f))

	read, _ := yamlReader.Read()
	role := osv1.Role{}
	err = yaml.NewYAMLToJSONDecoder(strings.NewReader(string(read))).Decode(&role)
	okd.CreateOrUpdateRole(&role, ns)

	read2, _ := yamlReader.Read()
	sa := v1.ServiceAccount{}
	err = yaml.NewYAMLToJSONDecoder(strings.NewReader(string(read2))).Decode(&sa)
	okd.CreateOrUpdateSa(&sa, ns)

	read3, _ := yamlReader.Read()
	binding := osv1.RoleBinding{}
	err = yaml.NewYAMLToJSONDecoder(strings.NewReader(string(read3))).Decode(&binding)
	okd.CreateOrUpdateRoleBinding(&binding, ns)
}

func installCRD(okd *ExternalOKD) {
	filename := GetFilePath("../../deploy/crd.yaml")
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	yamlReader := yaml.NewYAMLReader(bufio.NewReader(f))

	read, _ := yamlReader.Read()
	crd := apiextv1beta1.CustomResourceDefinition{}
	err = yaml.NewYAMLToJSONDecoder(strings.NewReader(string(read))).Decode(&crd)
	okd.CreateAndWaitForCRD(&crd)
}

// Run the operator locally
func runOperatorLocally(ns string, configLocation string, stopCh chan struct{}) {
	_ = os.Setenv("WATCH_NAMESPACE", ns)
	_ = os.Setenv("KUBECONFIG", configLocation)
	launcher.Launch(launcher.Parameters{StopChannel: stopCh})
}

func GetFilePath(path string) string {
	dir, _ := os.Getwd()
	absPath, _ := filepath.Abs(dir + "/" + path)
	return absPath
}