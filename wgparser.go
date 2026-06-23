package main

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

type WireGuardConfig struct {
	// Interface section
	PrivateKey string   `json:"privateKey"`
	Address    []string `json:"address"`
	DNS        []string `json:"dns"`
	MTU        int      `json:"mtu,omitempty"`

	// Peer section
	PublicKey    string   `json:"publicKey"`
	PresharedKey string   `json:"presharedKey,omitempty"`
	Endpoint     string   `json:"endpoint"`
	AllowedIPs   []string `json:"allowedIPs"`
}

func ParseWireGuardConfig(content string) (WireGuardConfig, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowBooleanKeys:    true,
		InsensitiveKeys:     true,
		InsensitiveSections: true,
	}, []byte(content))
	if err != nil {
		return WireGuardConfig{}, fmt.Errorf("parse ini: %w", err)
	}

	var wg WireGuardConfig

	iface, err := cfg.GetSection("Interface")
	if err != nil {
		return wg, fmt.Errorf("missing [Interface] section")
	}

	wg.PrivateKey = iface.Key("PrivateKey").String()
	if wg.PrivateKey == "" {
		return wg, fmt.Errorf("missing PrivateKey in [Interface]")
	}

	if addr := iface.Key("Address").String(); addr != "" {
		wg.Address = splitCSV(addr)
	}
	if dns := iface.Key("DNS").String(); dns != "" {
		wg.DNS = splitCSV(dns)
	}
	wg.MTU, _ = iface.Key("MTU").Int()

	peer, err := cfg.GetSection("Peer")
	if err != nil {
		return wg, fmt.Errorf("missing [Peer] section")
	}

	wg.PublicKey = peer.Key("PublicKey").String()
	if wg.PublicKey == "" {
		return wg, fmt.Errorf("missing PublicKey in [Peer]")
	}

	wg.PresharedKey = peer.Key("PresharedKey").String()
	wg.Endpoint = peer.Key("Endpoint").String()
	if ips := peer.Key("AllowedIPs").String(); ips != "" {
		wg.AllowedIPs = splitCSV(ips)
	}

	return wg, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
