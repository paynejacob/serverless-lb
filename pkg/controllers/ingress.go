package controllers

import (
	"context"
	"serverless-lb/pkg/resolver"

	v1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngressReconciler struct {
	client.Client

	Resolver            *resolver.Resolver
	DefaultIngressClass string

	IngressHosts map[string][]string
}

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ingress v1.Ingress

	if err := r.Get(ctx, req.NamespacedName, &ingress); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if ingress.DeletionTimestamp != nil {
		return r.onDelete(ctx, &ingress)
	}

	return r.onChange(ctx, &ingress)
}

func (r *IngressReconciler) onDelete(_ context.Context, ingress *v1.Ingress) (ctrl.Result, error) {
	key := getIngressKey(ingress)

	for _, rule := range ingress.Spec.Rules {
		r.Resolver.RemoveHost(rule.Host)
	}

	for _, host := range r.IngressHosts[key] {
		r.Resolver.RemoveHost(host)
	}

	delete(r.IngressHosts, key)

	return ctrl.Result{}, nil
}

func (r *IngressReconciler) onChange(_ context.Context, ingress *v1.Ingress) (ctrl.Result, error) {
	key := getIngressKey(ingress)

	for _, host := range r.IngressHosts[key] {
		r.Resolver.RemoveHost(host)
	}

	pool := ingress.Annotations[RoutingClassAnnotation]
	if pool == "" {
		pool = r.DefaultIngressClass
	}
	if pool == "" {
		return ctrl.Result{}, nil
	}

	for _, rule := range ingress.Spec.Rules {
		r.IngressHosts[key] = append(r.IngressHosts[key], rule.Host)
	}

	for _, host := range r.IngressHosts[key] {
		r.Resolver.AddHost(host, pool)
	}

	return ctrl.Result{}, nil
}

func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Ingress{}).
		Complete(r)
}

func getIngressKey(ingress *v1.Ingress) string {
	return ingress.Namespace + ":" + ingress.Name
}
