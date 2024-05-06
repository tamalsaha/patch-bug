package main

import (
	"context"
	"fmt"
	apps "k8s.io/api/apps/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	policy_util "kmodules.xyz/client-go/policy"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	if err := useGeneratedClient(); err != nil {
		panic(err)
	}
}

func useGeneratedClient() error {
	fmt.Println("Using Generated client")
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = 100
	cfg.Burst = 100

	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	sts, err := kc.AppsV1().StatefulSets("default").Get(context.TODO(), "ha-postgres", metav1.GetOptions{})
	if err != nil {
		return err
	}

	pdbRef := metav1.ObjectMeta{
		Name:      sts.Name,
		Namespace: sts.Namespace,
	}

	maxUnavailable := &intstr.IntOrString{IntVal: 1}
	matchLabelSelectors := map[string]string{
		"app.kubernetes.io/instance":   "ha-postgres",
		"app.kubernetes.io/managed-by": "kubedb.com",
		"app.kubernetes.io/name":       "postgreses.kubedb.com",
	}
	owner := metav1.NewControllerRef(sts, apps.SchemeGroupVersion.WithKind("StatefulSet"))
	_, _, err = policy_util.CreateOrPatchPodDisruptionBudget(context.TODO(), kc, pdbRef,
		func(in *policy.PodDisruptionBudget) *policy.PodDisruptionBudget {
			in.Labels = matchLabelSelectors
			core_util.EnsureOwnerReference(&in.ObjectMeta, owner)
			in.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: matchLabelSelectors,
			}
			in.Spec.MaxUnavailable = maxUnavailable
			in.Spec.MinAvailable = nil
			return in
		}, metav1.PatchOptions{})
	return err
}
