package ufw

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type Form struct {
	Action     string
	Direction  string
	Interface  string
	FromIP     string
	ToIP       string
	Port       string
	Protocol   string
	AppProfile string
}

func (f *Form) ParseForm() (string, error) {
	b := strings.Builder{}
	b.WriteString("ufw ")

	if f.AppProfile != "" {
		if f.Action != "allow" && f.Action != "deny" {
			return "", errors.New("action must be either 'allow' or 'deny'")
		}
		_, err := fmt.Fprintf(&b, "%s \"%s\"", f.Action, f.AppProfile)
		if err != nil {
			return "", errors.New("unable to parse app profile")
		}
		return b.String(), nil
	}

	if f.Action != "allow" && f.Action != "deny" {
		return "", errors.New("action must be either 'allow' or 'deny'")
	}

	b.WriteString(f.Action)

	if f.Direction != "" {
		_, err := fmt.Fprintf(&b, " %s", f.Direction)
		if err != nil {
			return "", errors.New("unable to parse direction")
		}

	}

	if f.Interface != "" {
		_, err := fmt.Fprintf(&b, " on %s", f.Interface)
		if err != nil {
			return "", errors.New("unable to parse interface")
		}

	}

	if f.FromIP != "" {
		if !validIpv4(f.FromIP) {
			return "", errors.New("invalid source IP address")
		}
		_, err := fmt.Fprintf(&b, " from %s", f.FromIP)
		if err != nil {
			return "", errors.New("unable to parse source IP")
		}

	}

	if f.ToIP != "" {
		if !validIpv4(f.FromIP) {
			return "", errors.New("invalid source IP address")
		}
		_, err := fmt.Fprintf(&b, " to %s", f.ToIP)
		if err != nil {
			return "", errors.New("unable to parse destination IP")
		}
	} else if f.Port != "" || f.Protocol != "" {
		//Assume that if ToIP is empty but Port or Protocol is set, the user wants to specify "to any"
		b.WriteString(" to any")
	}

	if f.Port != "" {
		_, err := fmt.Fprintf(&b, " port %s", f.Port)
		if err != nil {
			return "", errors.New("unable to parse port(s)")
		}

	}

	if f.Protocol != "" {
		_, err := fmt.Fprintf(&b, " proto %s", f.Protocol)
		if err != nil {
			return "", errors.New("unable to parse protocol")
		}

	}

	return b.String(), nil
}

func validIpv4(ip string) bool {
	goodIP := net.ParseIP(ip)
	return goodIP != nil && goodIP.To4() != nil
}
