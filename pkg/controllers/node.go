package controllers

import (
	"context"
	"serverless-lb/pkg/resolver"

	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NodeReconciler struct {
	client.Client

	Resolver         *resolver.Resolver
	DefaultNodeClass string
}

func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var node v1.Node

	if err := r.Get(ctx, req.NamespacedName, &node); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	pool := node.Annotations[RoutingClassAnnotation]
	if pool == "" {
		pool = r.DefaultNodeClass
	}
	if pool == "" {
		return ctrl.Result{}, nil
	}

	if node.DeletionTimestamp != nil || node.Spec.Unschedulable {
		for _, address := range node.Status.Addresses {
			switch address.Type {
			case v1.NodeExternalIP:
				r.Resolver.RemoveAddress(pool, address.Address)
				break
			case v1.NodeInternalIP:
				r.Resolver.RemoveAddress(pool, address.Address)
			}
		}

		return ctrl.Result{}, nil
	}

	for _, address := range node.Status.Addresses {
		switch address.Type {
		case v1.NodeExternalIP:
			r.Resolver.AddAddress(pool, address.Address)
			break
		case v1.NodeInternalIP:
			r.Resolver.AddAddress(pool, address.Address)
		}
	}

	return ctrl.Result{}, nil
}

func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Node{}).
		Complete(r)
}
