package ufw

import (
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"errors"
	"fmt"
	"math"
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
		//fmt.Println("WARNING: Directly configuring an app profile will automatically add an IPv6 rule as well!")
		if err != nil {
			return "", errors.New("unable to parse app profile")
		}
		return b.String(), nil
	}

	if f.Action != "allow" && f.Action != "deny" && f.Action != "reject" && f.Action != "limit" {
		return "", errors.New("action must be either 'allow', 'deny', 'reject', or 'limit'")
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
		if f.Protocol != "tcp" && f.Protocol != "udp" && f.Protocol != "tcp/udp" && f.Protocol != "udp/tcp" && f.Protocol != "all" && f.Protocol != "esp" && f.Protocol != "ah" && f.Protocol != "gre" && f.Protocol != "icmp" && f.Protocol != "ipv6" {
			return "", errors.New("protocol must be either 'tcp', 'udp', or 'tcp/udp', 'all', 'esp', 'ah', 'gre', 'icmp', or 'ipv6'")
		}
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

func ParseRuleFromNumber(num int) (string, error) {
	digits := digitCount(num)

	cmd := fmt.Sprintf("ufw status numbered | grep '^\\[ *%d\\]' | sed -E 's/^\\[\\s*[0-9]{%d}+\\]\\s*//'", num, digits)

	if ssh.GetSSHStatus() {
		if err := ssh.Checkup(); err != nil {
			return "", err
		}
		out, err := ssh.CommandStream(cmd)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(out), nil
	} else {
		out, err := local.RunCommand(cmd)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(out), nil
	}
}

func digitCount(n int) int {
	if n == 0 {
		return 1
	}
	return int(math.Log10(float64(n))) + 1
}
