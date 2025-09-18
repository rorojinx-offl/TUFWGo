package copilot

import (
	"TUFWGo/ufw"
	"encoding/json"
	"errors"
	"fmt"
)

type Payload struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	Action     string `json:"action"`
	Direction  string `json:"direction"`
	Interface  string `json:"interface"`
	From       string `json:"from"`
	To         string `json:"to"`
	Port       string `json:"port"`
	Protocol   string `json:"protocol"`
	AppProfile string `json:"app_profile"`
}

func CompileJSONToUFW(b []byte) ([]string, error) {
	var p Payload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if len(p.Rules) == 0 {
		return nil, errors.New("no rules provided")
	}

	cmds := make([]string, 0, len(p.Rules))
	for i, r := range p.Rules {
		// Cross-field rule: if app_profile is set, only allow|deny are valid.
		if r.AppProfile != "" && !(r.Action == "allow" || r.Action == "deny") {
			return nil, fmt.Errorf("rule %d: app_profile requires action 'allow' or 'deny'", i)
		}

		// Map into your Form and reuse your exact ParseForm logic/order.
		f := &ufw.Form{
			Action:     r.Action,
			Direction:  r.Direction,
			Interface:  r.Interface,
			FromIP:     r.From,
			ToIP:       r.To,
			Port:       r.Port,
			Protocol:   r.Protocol,
			AppProfile: r.AppProfile,
		}

		cmd, err := f.ParseForm()
		if err != nil {
			return nil, fmt.Errorf("rule %d invalid: %w", i, err)
		}
		cmds = append(cmds, cmd)
	}

	return cmds, nil
}
