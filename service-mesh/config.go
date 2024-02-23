package main

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	envoyConfigBootstrapV3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoyConfigClusterV3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoyCoreV3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointV3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoyConfigListenerV3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyConfigTCPProxyV3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoyTLSV3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

var loopbackCIDR = netip.MustParsePrefix("127.0.0.1/8")

// ProxyConfig represents the configuration for the proxy.
type ProxyConfig []configEntry

type configEntry struct {
	name         string
	listenAddr   netip.Addr
	listenPort   uint16
	remoteDomain string
	remotePort   uint16
}

// ParseProxyConfig parses the proxy configuration from the given string.
// The configuration is expected to be in the following format:
//
//	<name>#<listen-address>:<port>#<remote-domain>:<port>##<name>#<listen-address>:<port>#<remote-domain>:<port>...
//
// Example:
//
//	emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080
func ParseProxyConfig(data string) (ProxyConfig, error) {
	entries := strings.Split(data, "##")
	var cfg ProxyConfig
	for _, entry := range entries {
		parts := strings.Split(entry, "#")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid entry: %s", entry)
		}
		listenAddrPort, err := netip.ParseAddrPort(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid listen address: %s", parts[1])
		}

		if !loopbackCIDR.Contains(listenAddrPort.Addr()) {
			return nil, fmt.Errorf("listen address %s is not in local CIDR %s", listenAddrPort.Addr(), loopbackCIDR)
		}
		remoteDomain := parts[2]
		remoteDomain, remotePort, err := net.SplitHostPort(remoteDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid remote domain: %s", remoteDomain)
		}
		remotePortInt, err := strconv.Atoi(remotePort)
		if err != nil {
			return nil, fmt.Errorf("invalid remote port: %s", remotePort)
		}
		cfg = append(cfg, configEntry{
			name:         parts[0],
			listenAddr:   listenAddrPort.Addr(),
			listenPort:   listenAddrPort.Port(),
			remotePort:   uint16(remotePortInt),
			remoteDomain: remoteDomain,
		})
	}
	return cfg, nil
}

// ToEnvoyConfig converts the proxy configuration to an Envoy configuration.
// Reference: https://github.com/solo-io/envoy-operator/blob/master/pkg/kube/config.go
func (c ProxyConfig) ToEnvoyConfig() ([]byte, error) {
	config := &envoyConfigBootstrapV3.Bootstrap{
		StaticResources: &envoyConfigBootstrapV3.Bootstrap_StaticResources{},
	}
	listeners := make([]*envoyConfigListenerV3.Listener, 0, len(c))
	clusters := make([]*envoyConfigClusterV3.Cluster, 0, len(c))
	for _, entry := range c {
		listener, err := listener(entry)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
		cluster, err := cluster(entry)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}
	config.StaticResources.Listeners = listeners
	config.StaticResources.Clusters = clusters

	if err := config.ValidateAll(); err != nil {
		return nil, err
	}

	configBytes, err := protojson.Marshal(config)
	if err != nil {
		return nil, err
	}

	return configBytes, nil
}

func listener(entry configEntry) (*envoyConfigListenerV3.Listener, error) {
	proxy := &envoyConfigTCPProxyV3.TcpProxy{
		StatPrefix: entry.name,
		ClusterSpecifier: &envoyConfigTCPProxyV3.TcpProxy_Cluster{
			Cluster: entry.name,
		},
	}

	proxyAny, err := anypb.New(proxy)
	if err != nil {
		return nil, err
	}

	return &envoyConfigListenerV3.Listener{
		Name: entry.name,
		Address: &envoyCoreV3.Address{
			Address: &envoyCoreV3.Address_SocketAddress{
				SocketAddress: &envoyCoreV3.SocketAddress{
					Address: entry.listenAddr.String(),
					PortSpecifier: &envoyCoreV3.SocketAddress_PortValue{
						PortValue: uint32(entry.listenPort),
					},
				},
			},
		},
		FilterChains: []*envoyConfigListenerV3.FilterChain{
			{
				Filters: []*envoyConfigListenerV3.Filter{
					{
						Name: "envoy.filters.network.tcp_proxy",
						ConfigType: &envoyConfigListenerV3.Filter_TypedConfig{
							TypedConfig: proxyAny,
						},
					},
				},
			},
		},
	}, nil
}

func cluster(entry configEntry) (*envoyConfigClusterV3.Cluster, error) {
	socket, err := tlsTransportSocket()
	if err != nil {
		return nil, err
	}

	return &envoyConfigClusterV3.Cluster{
		Name: entry.name,
		ClusterDiscoveryType: &envoyConfigClusterV3.Cluster_Type{
			Type: envoyConfigClusterV3.Cluster_LOGICAL_DNS,
		},
		DnsLookupFamily: envoyConfigClusterV3.Cluster_V4_ONLY,
		LoadAssignment: &endpointV3.ClusterLoadAssignment{
			ClusterName: entry.name,
			Endpoints: []*endpointV3.LocalityLbEndpoints{
				{
					LbEndpoints: []*endpointV3.LbEndpoint{
						{
							HostIdentifier: &endpointV3.LbEndpoint_Endpoint{
								Endpoint: &endpointV3.Endpoint{
									Address: &envoyCoreV3.Address{
										Address: &envoyCoreV3.Address_SocketAddress{
											SocketAddress: &envoyCoreV3.SocketAddress{
												Address: entry.remoteDomain,
												PortSpecifier: &envoyCoreV3.SocketAddress_PortValue{
													PortValue: uint32(entry.remotePort),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		TransportSocket: socket,
	}, nil
}

func tlsTransportSocket() (*envoyCoreV3.TransportSocket, error) {
	tls := &envoyTLSV3.UpstreamTlsContext{
		CommonTlsContext: &envoyTLSV3.CommonTlsContext{
			TlsCertificates: []*envoyTLSV3.TlsCertificate{
				{
					PrivateKey: &envoyCoreV3.DataSource{
						Specifier: &envoyCoreV3.DataSource_Filename{
							Filename: "/tls-config/key.pem",
						},
					},
					CertificateChain: &envoyCoreV3.DataSource{
						Specifier: &envoyCoreV3.DataSource_Filename{
							Filename: "/tls-config/certChain.pem",
						},
					},
				},
			},
			ValidationContextType: &envoyTLSV3.CommonTlsContext_ValidationContext{
				ValidationContext: &envoyTLSV3.CertificateValidationContext{
					TrustedCa: &envoyCoreV3.DataSource{
						Specifier: &envoyCoreV3.DataSource_Filename{
							Filename: "/tls-config/MeshCACert.pem",
						},
					},
				},
			},
		},
	}
	tlsAny, err := anypb.New(tls)
	if err != nil {
		return nil, err
	}

	return &envoyCoreV3.TransportSocket{
		Name: "envoy.transport_sockets.tls",
		ConfigType: &envoyCoreV3.TransportSocket_TypedConfig{
			TypedConfig: tlsAny,
		},
	}, nil
}
