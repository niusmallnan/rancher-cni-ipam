package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"

	"github.com/Sirupsen/logrus"
	"github.com/containernetworking/cni/pkg/types"
)

const (
	metadataCIDR = "169.254.169.250"
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
	// gw: bridge_ip Dst:169.254.169.250
	cmd := fmt.Sprintf("ip addr show %s | grep 'inet\\b' | awk '{print $2}'", n.Bridge)
	logrus.Debug(cmd)
	bridgeAddr, err := exec.Command(cmd).Output()
	logrus.Debug(bridgeAddr)
	if err != nil {
		logrus.Errorf("failed to get flat bridge:%s ip, %v", n.Bridge, err)
	}
	bridgeIP, _, err := net.ParseCIDR(string(bridgeAddr))
	logrus.Debug(bridgeIP)
	if err != nil {
		logrus.Errorf("failed to parse flat bridge:%s cidr, %v", n.Bridge, err)
	} else {
		_, metadataIPNet, err := net.ParseCIDR(metadataCIDR)
		if err != nil {
			logrus.Errorf("failed to parse metadataCIDR")
		} else {
			mdRoute := types.Route{Dst: *metadataIPNet, GW: bridgeIP}
			n.IPAM.Routes = append(n.IPAM.Routes, mdRoute)
		}
	}

	return n.IPAM, nil
}
