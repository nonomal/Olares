package upgradeserver

import (
	"github.com/caddyserver/caddy/v2"
	"k8s.io/klog/v2"
)

type upgradeServer struct {
	config *caddy.Config
}

func NewUpgradeServer() (*upgradeServer, error) {
	config, err := newReverseProxy()
	if err != nil {
		klog.Errorf("failed to create reverse proxy config: %v", err)
		return nil, err
	}

	server := &upgradeServer{
		config: config,
	}

	caddy.TrapSignals()

	err = caddy.Run(server.config)
	if err != nil {
		klog.Errorf("failed to run upgrade server: %v", err)
		return nil, err
	}

	klog.Info("upgrade server started successfully")
	return server, nil
}

func (s *upgradeServer) Stop() error {
	err := caddy.Stop()
	if err != nil {
		klog.Errorf("failed to stop upgrade server: %v", err)
		return err
	}
	klog.Info("upgrade server stopped successfully")

	return nil
}
