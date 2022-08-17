package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/euchch/circleci-insights-prometheus-exporter/local/kubecontext"
	"github.com/magefile/mage/sh"
)

const (
	sourceControl       = "github.com"
	organization        = "euchch"
	reposiotry          = "circleci-insights-prometheus-exporter"
	remoteCluster       = "local-poos" // "arn:aws:eks:us-east-1:738515142885:cluster/hc-rc-general"
	remoteNameSpace     = "circleci-exporter"
	remoteTokenSecret   = "cci-reader"
	dockerImageName     = "euchch/cci-insights-exporter"
	tokenPath           = "./init/token-secret.yaml"
	kubectlVersion      = "1.24.3"
	kustomizeVersion    = "4.5.7"
	kustomizeImage      = "line/kubectl-kustomize"
	kustomizeBaseDir    = "./manifest_staging"
	kustomizeDeployFile = "./manifest_staging/cci-exporter-deploy.yaml"
)

var Aliases = map[string]interface{}{
	"bootstrap": Init,
	"run":       LocalRun,
	"build":     LocalBuild,
}

func K8sClusterValidate() error {
	contexts := kubecontext.NewKubeConfigView()
	contexts.GetKubectlConfig()
	if contexts.CurrentContext != remoteCluster {
		err := contexts.SwitchContext(remoteCluster)
		if err != nil {
			return err
		}
	}
	contexts.GetKubectlConfig()
	if contexts.CurrentContext != remoteCluster {
		return fmt.Errorf("%s context not found or could not be activated", remoteCluster)
	}

	err := contexts.SwitchNamespace(remoteNameSpace)
	if err != nil {
		return err
	}

	outp, err := sh.Output("kubectl", "get", "secret", remoteTokenSecret)
	if err != nil {
		return err
	}
	if strings.Contains(outp, "Error from server (NotFound)") {
		return fmt.Errorf(outp)
	}
	fmt.Println("We're good!")

	return nil
}

func Init() error {
	_, err := os.Stat(tokenPath)
	if err == nil {
		return nil
	}

	var user, token string
	fmt.Println("Enter CCI username: ")
	fmt.Scanln(&user)
	fmt.Println("Enter CCI token for", user, ": ")
	fmt.Scanln(&token)

	t := NewTokenSecret()
	t.TokenInit(user, token)
	t.MetadataInit(remoteTokenSecret, remoteNameSpace)
	t.WriteYaml(tokenPath)

	return nil
}

func LocalRun() error {
	K8sClusterValidate()
	err := sh.Run("go", "run", "main.go")
	if err != nil {
		return err
	}

	return err
}

func LocalBuild() error {
	fmt.Println("Building binaries...")
	err := sh.Run("docker", "build", ".", "-f", "./Dockerfile", "-t", dockerImageName)
	if err != nil {
		return err
	}

	fmt.Println("Building manifests...")
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Println("Override existing manifest")
	// Override existing manifest
	_, err = os.Stat(kustomizeDeployFile)
	if err == nil {
		os.RemoveAll(kustomizeBaseDir)
	}

	fmt.Println("Create folder if not exist")
	//Create folder if not exist
	_, err = os.Stat(kustomizeBaseDir)
	if err != nil {
		os.MkdirAll(kustomizeBaseDir, os.ModePerm)
	}

	fmt.Println("Creating manifest...")
	err = sh.Run("docker", "run", "--rm",
		"-v", pwd+":/cciex", "--entrypoint", "/usr/local/bin/kustomize",
		kustomizeImage+":"+kubectlVersion+"-"+kustomizeVersion,
		"build", "/cciex/config/default", "-o", "/cciex/"+kustomizeBaseDir+"/cci-exporter-deploy.yaml")
	if err != nil {
		return err
	}

	return nil
}
