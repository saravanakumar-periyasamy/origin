package controller

import (
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	osclient "github.com/openshift/origin/pkg/client"
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	sdnplugin "github.com/openshift/origin/pkg/sdn/plugin"
	"github.com/openshift/origin/pkg/service/controller/ingressip"
)

type SDNControllerConfig struct {
	NetworkConfig configapi.MasterNetworkConfig
}

func (c *SDNControllerConfig) RunController(ctx ControllerContext) (bool, error) {
	// TODO: Switch SDN to use client.Interface
	osClientConfig, err := ctx.ClientBuilder.Config(bootstrappolicy.InfraSDNControllerServiceAccountName)
	if err != nil {
		return false, err
	}
	osClient, err := osclient.New(osClientConfig)
	if err != nil {
		return false, err
	}
	go func() {
		err := sdnplugin.StartMaster(
			c.NetworkConfig,
			osClient,
			ctx.ClientBuilder.KubeInternalClientOrDie(bootstrappolicy.InfraSDNControllerServiceAccountName),
			ctx.InternalKubeInformers,
		)
		if err != nil {
			glog.Errorf("failed to start SDN plugin controller: %v", err)
		}
	}()
	return true, nil
}

type IngressIPControllerConfig struct {
	IngressIPNetworkCIDR string
	IngressIPSyncPeriod  time.Duration
}

func (c *IngressIPControllerConfig) RunController(ctx ControllerContext) (bool, error) {
	if len(c.IngressIPNetworkCIDR) == 0 {
		return true, nil
	}

	_, ipNet, err := net.ParseCIDR(c.IngressIPNetworkCIDR)
	if err != nil {
		return false, fmt.Errorf("unable to start ingress IP controller: %v", err)
	}

	if ipNet.IP.IsUnspecified() {
		// TODO: Is this an error?
		return true, nil
	}

	ingressIPController := ingressip.NewIngressIPController(
		ctx.ExternalKubeInformers.Core().V1().Services().Informer(),
		ctx.ClientBuilder.ClientOrDie(bootstrappolicy.InfraServiceIngressIPControllerServiceAccountName),
		ipNet,
		c.IngressIPSyncPeriod,
	)
	go ingressIPController.Run(ctx.Stop)

	return true, nil
}
