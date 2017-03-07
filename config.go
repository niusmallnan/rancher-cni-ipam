package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"

	"github.com/containernetworking/cni/pkg/types"
)

// IPAMConfig is used to load the options specified in the configuration file
type IPAMConfig struct {
	types.CommonArgs
	Type                 string        `json:"type"`
	LogToFile            string        `json:"logToFile"`
	IsDebugLevel         string        `json:"isDebugLevel"`
	SubnetPrefixSize     string        `json:"subnetPrefixSize"`
	Routes               []types.Route `json:"routes"`
	RancherContainerUUID types.UnmarshallableString
	MetadataRoute        types.Route
}

// Net loads the options of the CNI network configuration file
type Net struct {
	Name   string      `json:"name"`
	Bridge string      `json:"bridge"`
	IPAM   *IPAMConfig `json:"ipam"`
}

// LoadIPAMConfig loads the IPAM configuration from the given bytes
func LoadIPAMConfig(bytes []byte, args string) (*IPAMConfig, error) {
	n := Net{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}

	if n.IPAM == nil {
		return nil, fmt.Errorf("IPAM config missing 'ipam' key")
	}

	if err := types.LoadArgs(args, n.IPAM); err != nil {
		return nil, fmt.Errorf("failed to parse args %s: %v", args, err)
	}

	// ip addr show mpbr0 | grep "inet\b" | awk '{print $2}'
	cmd = fmt.Sprintf("ip addr show %s | grep 'inet\\b' | awk '{print $2}'", n.Name)
	bridgeAddr, err = exec.Command(cmd).Output()
	if err != nil {
		fmt.Errorf("failed to get flat bridge ip")
	}
	routegwv4, routev4, err := net.ParseCIDR(bridgeAddr)
	if err != nil {
		fmt.Errorf("failed to parse flat bridge cidr")
	} else {
		mdRoute = &types.Route{Dst: *routev4, GW: routegwv4}
		n.IPAM.MetadataRoute = mdRoute
	}

	return n.IPAM, nil
}
