package kubecontext

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type kubernetesConfigView struct {
	APIVersion string `yaml:"apiVersion"`
	Contexts   []struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
	Kind           string `yaml:"kind"`
	Preferences    struct {
	} `yaml:"preferences"`
	ClusterNameList     []string
	SelectedClusterName string
	SelectedNameSpace   string
}

func NewKubeConfigView() *kubernetesConfigView {
	return &kubernetesConfigView{}
}

func (c *kubernetesConfigView) GetKubectlConfig() {
	cmd := exec.Command("kubectl", "config", "view")

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	if len(strings.Split(out.String(), "kind")[0]) <= 0 {
		err = errors.New("error fetching data from your context")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err = yaml.Unmarshal([]byte(out.String()), c)
		if err != nil {
			log.Fatal(err)
		}
	}

	c.UpdateClusterList()
}

func (c *kubernetesConfigView) UpdateClusterList() {
	var clusterList []string
	for _, cluster := range c.Contexts {
		clusterList = append(clusterList, cluster.Name)
	}
	c.ClusterNameList = clusterList
}

func (c *kubernetesConfigView) SearchContext(ctxName string) bool {
	for _, s := range c.ClusterNameList {
		if s == c.CurrentContext {
			return true
		}
	}

	return false
}

func (c *kubernetesConfigView) SwitchContext(ctxName string) error {
	if !(c.SearchContext(ctxName)) {
		return fmt.Errorf("requested context %s not found in kubeconfig", ctxName)
	}

	c.SelectedClusterName = ctxName
	c.SetKubeConfig()

	return nil
}

func (c *kubernetesConfigView) SwitchNamespace(nsName string) error {
	// if !(c.SearchContext(ctxName)) {
	// 	return fmt.Errorf("requested context %s not found in kubeconfig", ctxName)
	// }

	c.SelectedNameSpace = nsName
	c.SetContextDefaultNs()

	return nil
}

func (c *kubernetesConfigView) PrintListOfClusters() {
	for index, clusterName := range c.ClusterNameList {
		fmt.Printf("(%d) %s \n", index, clusterName)
	}
}

func (c *kubernetesConfigView) SetKubeConfig() {
	cmd := exec.Command("kubectl", "config", "use-context", c.SelectedClusterName)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(out.String())
	}
}

func (c *kubernetesConfigView) SetContextDefaultNs() {
	cmd := exec.Command("kubectl", "config", "set-context", "--current", "--namespace", c.SelectedNameSpace)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(out.String())
	}
}
