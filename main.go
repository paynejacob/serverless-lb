package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"serverless-lb/pkg/controllers"
	"serverless-lb/pkg/resolver"
	"serverless-lb/pkg/server"

	"github.com/miekg/dns"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var err error
	var wg errgroup.Group

	var address string
	var defaultNodeClass string
	var defaultIngressClass string

	resolv := resolver.NewResolver()
	ctx, cancel := context.WithCancel(context.Background())
	opts := zap.Options{}

	flag.StringVar(&address, "address", ":5353", "The address the dns service binds to.")
	flag.StringVar(&defaultNodeClass, "default-node-class", "", "default routing class for nodes")
	flag.StringVar(&defaultIngressClass, "default-ingress-class", "default", "default routing class for ingresses")
	opts.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// get k8s client
	scheme := runtime.NewScheme()
	_ = corev1.SchemeBuilder.AddToScheme(scheme)
	_ = networkingv1.SchemeBuilder.AddToScheme(scheme)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})

	ingressController := &controllers.IngressReconciler{
		Client:              mgr.GetClient(),
		Resolver:            resolv,
		DefaultIngressClass: defaultIngressClass,
		IngressHosts:        map[string][]string{},
	}

	nodeController := &controllers.NodeReconciler{
		Client:           mgr.GetClient(),
		Resolver:         resolv,
		DefaultNodeClass: defaultNodeClass,
	}

	svr := &server.Server{
		Server: &dns.Server{
			Addr: address,
			Net:  "udp",
		},
		Resolver: resolv,
	}

	if err = ingressController.SetupWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "failed to configure ingress controller")
	}

	if err = nodeController.SetupWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "failed to configure node controller")
	}

	// watch for shutdown signals
	go signalListener(cancel)

	wg.Go(func() error {
		return svr.Run(ctx)
	})

	wg.Go(func() error {
		return mgr.Start(ctx)
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	if err = wg.Wait(); err != nil {
		ctrl.Log.Error(err, "operator encountered an error")
	}
}

func signalListener(cancel func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	cancel()
}
