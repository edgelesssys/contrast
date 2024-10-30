#!/usr/bin/env bash
# This script sets up NAT for IMDS on Azure/AWS
# This is required for IMDS to work on Azure/AWS
# This script is executed as a oneshot systemd service
# during first boot

set -euo pipefail

IMDS_IP="169.254.169.254"
DUMMY_IP="169.254.99.99"

# trap errors
trap 'echo "Error: $0:$LINENO stopped"; exit 1' ERR INT

# Function to setup veth pair
function setup_proxy_arp() {
  local pod_ip
  pod_ip=$(ip netns exec podns ip route get "$IMDS_IP" | awk '{ for(i=1; i<=NF; i++) { if($i == "src") { print $(i+1); break; } } }')

  ip link add veth2 type veth peer name veth1
  # Proxy arp does not get enabled when no IP address is assigned
  ip address add "$DUMMY_IP/32" dev veth1
  ip link set up dev veth1

  sysctl -w net.ipv4.ip_forward=1
  sysctl -w net.ipv4.conf.veth1.proxy_arp=1
  sysctl -w net.ipv4.neigh.veth1.proxy_delay=0

  ip link set veth2 netns podns
  ip netns exec podns ip link set up dev veth2
  ip netns exec podns ip route add "$IMDS_IP/32" dev veth2

  ip route add "$pod_ip/32" dev veth1

  local hwaddr
  hwaddr=$(ip netns exec podns ip -br link show veth2 | awk 'NR==1 { print $3 }')
  ip neigh replace "$pod_ip" dev veth1 lladdr "$hwaddr"

  iptables -t nat -A POSTROUTING -s "$pod_ip/32" -d "$IMDS_IP/32" -j MASQUERADE
}

# Execute functions
setup_proxy_arp
