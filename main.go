package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2/klogr"
	dbapi "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	kubedbclient "kubedb.dev/apimachinery/client/clientset/versioned"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	// NOTE: Register KubeDB api types
	_ = kubedbscheme.AddToScheme(scheme)

	ctrl.SetLogger(klogr.New())
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = 100
	cfg.Burst = 100

	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, hc)
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
		//Opts: client.WarningHandlerOptions{
		//	SuppressWarnings:   false,
		//	AllowDuplicateLogs: false,
		//},
	})
}

func main() {
	if err := useGeneratedClient(); err != nil {
		panic(err)
	}
	if err := useKubebuilderClient(); err != nil {
		panic(err)
	}
}

func useGeneratedClient() error {
	fmt.Println("Using Generated client")
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = 100
	cfg.Burst = 100

	kc, err := kubedbclient.NewForConfig(cfg)
	if err != nil {
		return err
	}

	var pglist *dbapi.PostgresList
	pglist, err = kc.KubedbV1alpha2().Postgreses(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, db := range pglist.Items {
		fmt.Println(client.ObjectKeyFromObject(&db))
	}
	return nil
}

func useKubebuilderClient() error {
	fmt.Println("Using kubebuilder client")
	kc, err := NewClient()
	if err != nil {
		return err
	}

	var pglist dbapi.PostgresList
	err = kc.List(context.TODO(), &pglist)
	if err != nil {
		return err
	}
	for _, db := range pglist.Items {
		fmt.Println(client.ObjectKeyFromObject(&db))
	}
	return nil
}
