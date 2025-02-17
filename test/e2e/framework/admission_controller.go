/*
Copyright The KubeDB Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package framework

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	catlog "kubedb.dev/apimachinery/apis/catalog/v1alpha1"
	"kubedb.dev/postgres/pkg/cmds/server"

	"github.com/appscode/go/log"
	shell "github.com/codeskyblue/go-sh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	kApi "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	kutil "kmodules.xyz/client-go"
	admsn_kutil "kmodules.xyz/client-go/admissionregistration/v1beta1"
	apiext_util "kmodules.xyz/client-go/apiextensions/v1beta1"
	meta_util "kmodules.xyz/client-go/meta"
)

func (f *Framework) isApiSvcReady(apiSvcName string) error {
	apiSvc, err := f.kaClient.ApiregistrationV1beta1().APIServices().Get(apiSvcName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for _, cond := range apiSvc.Status.Conditions {
		if cond.Type == kApi.Available && cond.Status == kApi.ConditionTrue {
			log.Infof("APIService %v status is true", apiSvcName)
			return nil
		}
	}
	log.Errorf("APIService %v not ready yet", apiSvcName)
	return fmt.Errorf("APIService %v not ready yet", apiSvcName)
}

func (f *Framework) EventuallyAPIServiceReady() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			if err := f.isApiSvcReady("v1alpha1.mutators.kubedb.com"); err != nil {
				return err
			}
			if err := f.isApiSvcReady("v1alpha1.validators.kubedb.com"); err != nil {
				return err
			}
			time.Sleep(time.Second * 5) // let the resource become available

			// Check if the annotations of validating webhook is updated by operator/controller
			apiSvc, err := f.kaClient.ApiregistrationV1beta1().APIServices().Get("v1alpha1.validators.kubedb.com", metav1.GetOptions{})
			if err != nil {
				return err
			}

			if _, err := meta_util.GetString(apiSvc.Annotations, admsn_kutil.KeyAdmissionWebhookActive); err == kutil.ErrNotFound {
				log.Errorf("APIService v1alpha1.validators.kubedb.com not ready yet")
				return err
			}
			return nil
		},
		time.Minute*2,
		time.Second*5,
	)
}

func (f *Framework) RunOperatorAndServer(config *restclient.Config, kubeconfigPath string, stopCh <-chan struct{}) {
	defer GinkgoRecover()

	// ensure crds. Mainly for catalogVersions CRD.
	log.Infoln("Ensuring CustomResourceDefinition...")
	crds := []*crd_api.CustomResourceDefinition{
		catlog.PostgresVersion{}.CustomResourceDefinition(),
	}
	err := apiext_util.RegisterCRDs(f.apiExtKubeClient, crds)
	Expect(err).NotTo(HaveOccurred())

	sh := shell.NewSession()
	args := []interface{}{"--minikube", fmt.Sprintf("--docker-registry=%v", DockerRegistry)}
	SetupServer := filepath.Join("..", "..", "hack", "deploy", "setup.sh")

	By("Creating API server and webhook stuffs")
	cmd := sh.Command(SetupServer, args...)
	err = cmd.Run()
	Expect(err).ShouldNot(HaveOccurred())

	By("Starting Server and Operator")
	serverOpt := server.NewPostgresServerOptions(os.Stdout, os.Stderr)

	serverOpt.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath = kubeconfigPath
	serverOpt.RecommendedOptions.SecureServing.BindPort = 8443
	serverOpt.RecommendedOptions.SecureServing.BindAddress = net.ParseIP("127.0.0.1")
	serverOpt.RecommendedOptions.Authorization.RemoteKubeConfigFile = kubeconfigPath
	serverOpt.RecommendedOptions.Authentication.RemoteKubeConfigFile = kubeconfigPath

	serverOpt.ExtraOptions.EnableRBAC = true
	serverOpt.ExtraOptions.EnableMutatingWebhook = true
	serverOpt.ExtraOptions.EnableValidatingWebhook = true

	err = serverOpt.Run(stopCh)
	Expect(err).NotTo(HaveOccurred())
}

func (f *Framework) CleanAdmissionConfigs() {
	// delete validating Webhook
	if err := f.kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().DeleteCollection(deleteInForeground(), metav1.ListOptions{
		LabelSelector: "app=kubedb",
	}); err != nil && !kerr.IsNotFound(err) {
		fmt.Printf("error in deletion of Validating Webhook. Error: %v", err)
	}

	// delete mutating Webhook
	if err := f.kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().DeleteCollection(deleteInForeground(), metav1.ListOptions{
		LabelSelector: "app=kubedb",
	}); err != nil && !kerr.IsNotFound(err) {
		fmt.Printf("error in deletion of Mutating Webhook. Error: %v", err)
	}

	// Delete APIService
	if err := f.kaClient.ApiregistrationV1beta1().APIServices().DeleteCollection(deleteInForeground(), metav1.ListOptions{
		LabelSelector: "app=kubedb",
	}); err != nil && !kerr.IsNotFound(err) {
		fmt.Printf("error in deletion of APIService. Error: %v", err)
	}

	// Delete Service
	if err := f.kubeClient.CoreV1().Services("kube-system").Delete("kubedb-operator", &metav1.DeleteOptions{}); err != nil && !kerr.IsNotFound(err) {
		fmt.Printf("error in deletion of Service. Error: %v", err)
	}

	// Delete EndPoints
	if err := f.kubeClient.CoreV1().Endpoints("kube-system").DeleteCollection(deleteInForeground(), metav1.ListOptions{
		LabelSelector: "app=kubedb",
	}); err != nil && !kerr.IsNotFound(err) {
		fmt.Printf("error in deletion of Endpoints. Error: %v", err)
	}

	time.Sleep(time.Second * 1) // let the kube-server know it!!
}
