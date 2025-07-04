package upgradeserver

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/headers"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/proxyprotocol"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	"go.uber.org/zap"
)

/**
{
  "apps": {
    "http": {
      "servers": {
        "https_server": {
          "listen": [":443"],
          "routes": [
            {
              "handle": [
                {
                  "handler": "reverse_proxy",
                  "upstreams": [{"dial": "backend:443"}]
                }
              ]
            }
          ]
        },
        "proxyproto_server": {
          "listen": [":8443"],
          "listener_wrappers": [
            {"wrapper": "proxy_protocol"}
          ],
          "routes": [
            {
              "handle": [
                {
                  "handler": "reverse_proxy",
                  "upstreams": [{"dial": "backend:8080"}]
                }
              ]
            }
          ]
        }
      }
    }
  }
}
*/

var debug = true

func newReverseProxy() (*caddy.Config, error) {

	// set up the downstream address; assume missing information from given parts
	fromAddr, err := httpcaddyfile.ParseAddress("https://127.0.0.1:49443")
	if err != nil {
		return nil, fmt.Errorf("invalid downstream address for https: %v", err)
	}

	fromProxyAddr, err := httpcaddyfile.ParseAddress("https://127.0.0.1:49444")
	if err != nil {
		return nil, fmt.Errorf("invalid downstream address for proxy protocol: %v", err)
	}

	// https reverse proxy
	upstreamHttpsPool := reverseproxy.UpstreamPool{
		&reverseproxy.Upstream{
			Dial: caddy.JoinNetworkAddress("", "127.0.0.1", "443"),
		},
	}

	httpsTransport := reverseproxy.HTTPTransport{TLS: &reverseproxy.TLSConfig{}}

	httpsHandler := reverseproxy.Handler{
		TransportRaw: caddyconfig.JSONModuleObject(httpsTransport, "protocol", "http", nil),
		Upstreams:    upstreamHttpsPool,
		Headers: &headers.Handler{
			Request: &headers.HeaderOps{
				Set: map[string][]string{
					"Host": {"{http.reverse_proxy.upstream.hostport}"},
				},
			},
		},
	}

	httpsRoute := caddyhttp.Route{
		HandlersRaw: []json.RawMessage{
			caddyconfig.JSONModuleObject(httpsHandler, "handler", "reverse_proxy", nil),
		},
	}

	httpsServer := &caddyhttp.Server{
		Routes: caddyhttp.RouteList{httpsRoute},
		Listen: []string{":" + fromAddr.Port},
		Logs:   &caddyhttp.ServerLogConfig{},
	}

	// proxy protocol reverse proxy
	upstreamProxyProtocolPool := reverseproxy.UpstreamPool{
		&reverseproxy.Upstream{
			Dial: caddy.JoinNetworkAddress("", "127.0.0.1", "444"),
		},
	}

	proxyProtocolTransport := reverseproxy.HTTPTransport{TLS: &reverseproxy.TLSConfig{}, ProxyProtocol: "on"}

	proxyProtocolHandler := reverseproxy.Handler{
		TransportRaw: caddyconfig.JSONModuleObject(proxyProtocolTransport, "protocol", "http", nil),
		Upstreams:    upstreamProxyProtocolPool,
		Headers: &headers.Handler{
			Request: &headers.HeaderOps{
				Set: map[string][]string{
					"Host": {"{http.reverse_proxy.upstream.hostport}"},
				},
			},
		},
	}

	proxyProtocolRoute := caddyhttp.Route{
		HandlersRaw: []json.RawMessage{
			caddyconfig.JSONModuleObject(proxyProtocolHandler, "handler", "reverse_proxy", nil),
		},
	}

	wrapper := proxyprotocol.ListenerWrapper{}

	proxyProtocolServer := &caddyhttp.Server{
		Routes: caddyhttp.RouteList{proxyProtocolRoute},
		Listen: []string{":" + fromProxyAddr.Port},
		ListenerWrappersRaw: []json.RawMessage{
			caddyconfig.JSONModuleObject(wrapper, "wrapper", "proxy_protocol", nil),
		},
		Logs: &caddyhttp.ServerLogConfig{},
	}

	httpApp := caddyhttp.App{
		Servers: map[string]*caddyhttp.Server{
			"reverse_proxy":  httpsServer,
			"proxy_protocol": proxyProtocolServer,
		},
	}

	appsRaw := caddy.ModuleMap{
		"http": caddyconfig.JSON(httpApp, nil),
	}

	var false bool
	cfg := &caddy.Config{
		Admin: &caddy.AdminConfig{
			Disabled: true,
			Config: &caddy.ConfigSettings{
				Persist: &false,
			},
		},
		AppsRaw: appsRaw,
	}

	if debug {
		cfg.Logging = &caddy.Logging{
			Logs: map[string]*caddy.CustomLog{
				"default": {BaseLog: caddy.BaseLog{Level: zap.DebugLevel.CapitalString()}},
			},
		}
	}

	return cfg, nil
}
