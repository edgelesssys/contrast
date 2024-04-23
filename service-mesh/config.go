// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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
	envoyOrigDstV3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/original_dst/v3"
	envoyConfigTCPProxyV3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoyTLSV3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var loopbackCIDR = netip.MustParsePrefix("127.0.0.1/8")

// ProxyConfig represents the configuration for the proxy.
type ProxyConfig struct {
	egress  []egressConfigEntry
	ingress []ingressConfigEntry
}
type egressConfigEntry struct {
	name         string
	clusterName  string
	listenAddr   netip.Addr
	listenPort   uint16
	remoteDomain string
	remotePort   uint16
}
type ingressConfigEntry struct {
	name       string
	listenPort uint16
	disableTLS bool
}

// ParseProxyConfig parses the proxy configuration from the given string.
// The configuration is expected to be in the following format:
//
//	<name>#<listen-address>:<port>#<remote-domain>:<port>##<name>#<listen-address>:<port>#<remote-domain>:<port>...
//
// Example:
//
//	emoji#127.137.0.1:8081#emoji-svc:8080##voting#127.137.0.2:8081#voting-svc:8080
func ParseProxyConfig(ingressConfig, egressConfig string) (ProxyConfig, error) {
	if ingressConfig == "" && egressConfig == "" {
		return ProxyConfig{}, nil
	}

	entries := strings.Split(egressConfig, "##")
	var cfg ProxyConfig
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "#")
		if len(parts) != 3 {
			return ProxyConfig{}, fmt.Errorf("invalid entry: %s", entry)
		}
		listenAddrPort, err := netip.ParseAddrPort(parts[1])
		if err != nil {
			return ProxyConfig{}, fmt.Errorf("invalid listen address: %s", parts[1])
		}

		if !loopbackCIDR.Contains(listenAddrPort.Addr()) {
			return ProxyConfig{}, fmt.Errorf("listen address %s is not in local CIDR %s", listenAddrPort.Addr(), loopbackCIDR)
		}
		remoteDomain := parts[2]
		remoteDomain, remotePort, err := net.SplitHostPort(remoteDomain)
		if err != nil {
			return ProxyConfig{}, fmt.Errorf("invalid remote domain: %s", remoteDomain)
		}
		remotePortInt, err := strconv.Atoi(remotePort)
		if err != nil {
			return ProxyConfig{}, fmt.Errorf("invalid remote port: %s", remotePort)
		}
		cfg.egress = append(cfg.egress, egressConfigEntry{
			name:         parts[0],
			clusterName:  parts[0],
			listenAddr:   listenAddrPort.Addr(),
			listenPort:   listenAddrPort.Port(),
			remotePort:   uint16(remotePortInt),
			remoteDomain: remoteDomain,
		})
	}

	for _, entry := range strings.Split(ingressConfig, "##") {
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "#")
		if len(parts) != 3 {
			return ProxyConfig{}, fmt.Errorf("invalid entry: %s", entry)
		}
		listenPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return ProxyConfig{}, fmt.Errorf("invalid listen port: %s", parts[1])
		}
		disableTLS, err := strconv.ParseBool(parts[2])
		if err != nil {
			return ProxyConfig{}, fmt.Errorf("invalid disable TLS: %s", parts[2])
		}
		cfg.ingress = append(cfg.ingress, ingressConfigEntry{
			name:       parts[0],
			listenPort: uint16(listenPort),
			disableTLS: disableTLS,
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
	listeners := make([]*envoyConfigListenerV3.Listener, 0)
	clusters := make([]*envoyConfigClusterV3.Cluster, 0)

	// Create listeners and clusters for egress traffic.
	for _, entry := range c.egress {
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

	// Create listeners and clusters for ingress traffic.
	ingrListenerClientAuth, err := ingressListener("ingress", 15006, true)
	if err != nil {
		return nil, err
	}
	ingrListenerNoClientAuth, err := ingressListener("ingressWithoutClientAuth", 15007, false)
	if err != nil {
		return nil, err
	}

	ingressCluster := &envoyConfigClusterV3.Cluster{
		Name:                 "ingress",
		ClusterDiscoveryType: &envoyConfigClusterV3.Cluster_Type{Type: envoyConfigClusterV3.Cluster_ORIGINAL_DST},
		DnsLookupFamily:      envoyConfigClusterV3.Cluster_V4_ONLY,
		LbPolicy:             envoyConfigClusterV3.Cluster_CLUSTER_PROVIDED,
	}

	listeners = append(listeners, ingrListenerClientAuth)
	listeners = append(listeners, ingrListenerNoClientAuth)
	clusters = append(clusters, ingressCluster)

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

func listener(entry egressConfigEntry) (*envoyConfigListenerV3.Listener, error) {
	proxy := &envoyConfigTCPProxyV3.TcpProxy{
		StatPrefix: entry.name,
		ClusterSpecifier: &envoyConfigTCPProxyV3.TcpProxy_Cluster{
			Cluster: entry.clusterName,
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

func cluster(entry egressConfigEntry) (*envoyConfigClusterV3.Cluster, error) {
	socket, err := upstreamTLSTransportSocket()
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

func ingressListener(name string, listenPort uint16, requireClientCertificate bool) (*envoyConfigListenerV3.Listener, error) {
	ingressListener, err := listener(egressConfigEntry{
		name:        name,
		clusterName: "ingress",
		listenAddr:  netip.MustParseAddr("0.0.0.0"),
		listenPort:  listenPort,
	})
	if err != nil {
		return nil, err
	}
	ingressListener.Transparent = &wrapperspb.BoolValue{Value: true}
	originalDstConfig := &envoyOrigDstV3.OriginalDst{}
	originalDstAny, err := anypb.New(originalDstConfig)
	if err != nil {
		return nil, err
	}
	ingressListener.ListenerFilters = []*envoyConfigListenerV3.ListenerFilter{
		{
			Name:       "envoy.filters.listener.original_dst",
			ConfigType: &envoyConfigListenerV3.ListenerFilter_TypedConfig{TypedConfig: originalDstAny},
		},
	}
	tlsSock, err := downstreamTLSTransportSocket(requireClientCertificate)
	if err != nil {
		return nil, err
	}
	ingressListener.FilterChains[0].TransportSocket = tlsSock
	return ingressListener, nil
}

func upstreamTLSTransportSocket() (*envoyCoreV3.TransportSocket, error) {
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

func downstreamTLSTransportSocket(requireClientCertificate bool) (*envoyCoreV3.TransportSocket, error) {
	tls := &envoyTLSV3.DownstreamTlsContext{
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
		RequireClientCertificate: &wrapperspb.BoolValue{Value: requireClientCertificate},
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
