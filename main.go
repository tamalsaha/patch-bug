package main

import (
	"context"
	"fmt"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	core_util "kmodules.xyz/client-go/core/v1"
	coreutil "kmodules.xyz/client-go/core/v1"
	policy_util "kmodules.xyz/client-go/policy"
	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	psapi "kubeops.dev/petset/apis/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var ps = `apiVersion: apps.k8s.appscode.com/v1
kind: PetSet
metadata:
  creationTimestamp: "2024-05-06T18:40:09Z"
  generation: 1
  name: fer
  namespace: default
  resourceVersion: "2069"
  uid: 9fe3728d-914d-47cd-b62d-701e8806a6cc
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: fer
      app.kubernetes.io/managed-by: kubedb.com
      app.kubernetes.io/name: ferretdbs.kubedb.com
  serviceName: ""
  template:
    metadata:
      labels:
        app.kubernetes.io/component: database
        app.kubernetes.io/instance: fer
        app.kubernetes.io/managed-by: kubedb.com
        app.kubernetes.io/name: ferretdbs.kubedb.com
    spec:
      containers:
      - image: ghcr.io/appscode-images/ferretdb:1.18.0
        imagePullPolicy: IfNotPresent
        name: ferretdb
        ports:
        - containerPort: 27017
          name: db
          protocol: TCP
        resources: {}
  updateStrategy: {}`

func main() {
	var cur psapi.PetSet
	err := yaml.Unmarshal([]byte(ps), &cur)
	if err != nil {
		panic(err)
	}

	transform := func(obj client.Object, createOp bool) client.Object {
		in := obj.(*psapi.PetSet)

		c := core.Container{
			Name:  api.FerretDBContainerName,
			Image: "ghcr.io/appscode-images/ferretdb:1.18.0",
			Ports: []core.ContainerPort{
				{
					Name:          "db",
					ContainerPort: api.FerretDBDefaultPort,
					Protocol:      core.ProtocolTCP, // fixes unnecessary patching
				},
			},
			ImagePullPolicy: core.PullIfNotPresent,
		}
		in.Spec.Template.Spec.Containers = coreutil.UpsertContainers(in.Spec.Template.Spec.Containers, []core.Container{c})
		//in.Spec.Selector = &meta.LabelSelector{
		//	//in.Labels = r.db.PodControllerLabels(opts.controllerLabels, opts.labels)
		//	MatchLabels: opts.selectors,
		//	//copyFromPodTemplate(in, *opts.podTemplate)
		//	//in.Spec.Template.Spec.Containers = coreutil.UpsertContainers(in.Spec.Template.Spec.Containers, containers)
		//}
		// in.Spec.Template.Labels = r.db.PodLabels(opts.labels)
		//in.Spec.Replicas = opts.replicas
		//in.Spec.Template.Spec.Volumes = opts.volumes
		//// PetSet update strategy is set default to "OnDelete".
		//in.Spec.UpdateStrategy = apps.StatefulSetUpdateStrategy{
		//	Type: apps.OnDeleteStatefulSetStrategyType,
		//}
		//in.Spec.PodPlacementPolicy = r.db.Spec.PodPlacementPolicy
		//if createOp {
		//	coreutil.EnsureOwnerReference(&in.ObjectMeta, r.db.AsOwner())
		//}

		return in
	}

	patch := client.MergeFrom(&cur)
	mod := transform(cur.DeepCopyObject().(client.Object), false)

	data, err := patch.Data(mod)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	curJson, _ := json.Marshal(cur)
	modJson, _ := json.Marshal(mod)
	d2, err := jsonpatch.CreatePatch(curJson, modJson)
	if err != nil {
		panic(err)
	}
	d2Json, _ := json.Marshal(d2)
	fmt.Println("---------------------------------")
	fmt.Println(string(d2Json))
}

func main_() {
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
