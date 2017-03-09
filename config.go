package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/containernetworking/cni/pkg/types"
)

const (
	metadataCIDR = "169.254.169.250/32"
)

// IPAMConfig is used to load the options specified in the configuration file
type IPAMConfig struct {
	types.CommonArgs
	Type                 string        `json:"type"`
	LogToFile            string        `json:"logToFile"`
	IsDebugLevel         string        `json:"isDebugLevel"`
	IsFlat               bool          `json:"isFlat"`
	SubnetPrefixSize     string        `json:"subnetPrefixSize"`
	Routes               []types.Route `json:"routes"`
	RancherContainerUUID types.UnmarshallableString
}

func (ipamConf *IPAMConfig) String() string {
	routes := make([]string, 3)
	for _, r := range ipamConf.Routes {
		b, _ := r.MarshalJSON()
		routes = append(routes, string(b))
	}
	return fmt.Sprintf("&IPAMConfig{CommonArgs:%#v, Type:%s, LogToFile:%s, IsDebugLevel:%s, SubnetPrefixSize:%s, RancherContainerUUID:%s, Routes:%s}",
		ipamConf.IgnoreUnknown, ipamConf.Type, ipamConf.LogToFile,
		ipamConf.IsDebugLevel, ipamConf.SubnetPrefixSize,
		ipamConf.RancherContainerUUID, routes)
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

	if n.IPAM.IsDebugLevel == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if n.IPAM.LogToFile != "" {
		f, err := os.OpenFile(n.IPAM.LogToFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err == nil && f != nil {
			logrus.SetOutput(f)
			defer f.Close()
		}
	}

	if n.IPAM.IsFlat == true {
		// ip addr show mpbr0 | grep "inet\b" | awk '{print $2}'
		// gw: bridge_ip Dst:169.254.169.250
		cmd := fmt.Sprintf("ip addr show %s | grep 'inet\\b' | awk '{print $2}'", n.Bridge)
		logrus.Debugf("rancher-cni-ipam: %s", cmd)
		bridgeCIDR, err := exec.Command("/bin/sh", "-c", cmd).Output()
		logrus.Debugf("rancher-cni-ipam: get bridgeCIDR %s", bridgeCIDR)
		if err != nil {
			logrus.Errorf("failed to get flat bridge: %s ip, %v", n.Bridge, err)
		}
		bridgeIP, _, err := net.ParseCIDR(strings.Replace(string(bridgeCIDR), "\n", "", -1))
		logrus.Debugf("rancher-cni-ipam: get bridgeIP %s", bridgeIP)
		if err != nil {
			logrus.Errorf("rancher-cni-ipam: failed to parse flat bridge:%s cidr, %v", n.Bridge, err)
		} else {
			_, metadataIPNet, err := net.ParseCIDR(metadataCIDR)
			if err != nil {
				logrus.Errorf("rancher-cni-ipam: failed to parse metadataCIDR, err: %v", err)
			} else {
				mdRoute := types.Route{Dst: *metadataIPNet, GW: bridgeIP}
				logrus.Debugf("rancher-cni-ipam: metadata route, dst: %s, gw: %s", metadataIPNet.String(), bridgeIP)
				n.IPAM.Routes = append(n.IPAM.Routes, mdRoute)
			}
		}
	}

	return n.IPAM, nil
}
