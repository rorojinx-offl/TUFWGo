package alert

import (
	"TUFWGo/system/ssh"
	"TUFWGo/ufw"
	"fmt"
	"net"
	"os"
	"time"
)

type EmailInfo struct {
	Action     string
	Timestamp  string
	ExecutedBy string
	Hostname   string
	LocalIP    string
	Rule       *ufw.Form
	Command    string
}

var DeleteRule string

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr).String()
	return localAddr, nil
}

func (e *EmailInfo) prepareEmailInfo(action, cmd string, rule *ufw.Form) {
	e.Action = action
	e.Timestamp = time.Now().String()
	e.ExecutedBy = os.Getenv("USER")
	hostname, err := os.Hostname()
	if err != nil {
		e.Hostname = "unknown"
	} else {
		e.Hostname = hostname
	}
	localIP, err := getLocalIP()
	if err != nil {
		e.LocalIP = "unknown"
	} else {
		e.LocalIP = localIP
	}
	e.Rule = rule
	e.Command = cmd
}

func (e *EmailInfo) prepareMessage() string {

	var message string
	/*if e.Rule == nil {
			message = fmt.Sprintf(`
	Hello,
	An action was performed on your firewall via TUFWGo.
	📌 Action: %s
	📌 Timestamp: %s
	📌 Executed By: %s
	📌 Hostname: %s
	📌 Local IP: %s

	🏷️ Command Executed:
		%s

	TUFWGo Alert Manager
	`,
				e.Action,
				e.Timestamp,
				e.ExecutedBy,
				e.Hostname,
				e.LocalIP,
				e.Command)
			return message
		}*/

	var appProfile string
	var direction string
	var iface string
	var fromIP string
	var toIP string
	var port string
	var protocol string

	if e.Rule.AppProfile == "" {
		appProfile = "N/A"
	} else {
		appProfile = e.Rule.AppProfile
	}
	if e.Rule.Direction == "" {
		direction = "N/A"
	} else {
		direction = e.Rule.Direction
	}
	if e.Rule.Interface == "" {
		iface = "N/A"
	} else {
		iface = e.Rule.Interface
	}
	if e.Rule.FromIP == "" {
		fromIP = "N/A"
	} else {
		fromIP = e.Rule.FromIP
	}
	if e.Rule.ToIP == "" {
		toIP = "any"
	} else {
		toIP = e.Rule.ToIP
	}
	if e.Rule.Port == "" {
		port = "N/A"
	} else {
		port = e.Rule.Port
	}
	if e.Rule.Protocol == "" {
		protocol = "N/A"
	} else {
		protocol = e.Rule.Protocol
	}

	if e.Action == "Rule Added" {
		message = fmt.Sprintf(`
Hello,
An action was performed on your firewall via TUFWGo.
📌 Action: %s
📌 Timestamp: %s
📌 Executed By: %s
📌 Hostname: %s
📌 Local IP: %s
📌 Rule Details:
	- Action: %s
	- Direction: %s
	- Interface: %s
	- From: %s
	- To: %s
	- Port: %s
	- Protocol: %s
	- App Profile: %s

🏷️ Command Executed:
	%s

TUFWGo Alert Manager
`,
			e.Action,
			e.Timestamp,
			e.ExecutedBy,
			e.Hostname,
			e.LocalIP,
			e.Rule.Action,
			direction,
			iface,
			fromIP,
			toIP,
			port,
			protocol,
			appProfile,
			e.Command)
	} else if e.Action == "Rule Deleted" && e.Rule == nil {
		message = fmt.Sprintf(`
Hello,
An action was performed on your firewall via TUFWGo.
📌 Action: %s
📌 Timestamp: %s
📌 Executed By: %s
📌 Hostname: %s
📌 Local IP: %s
📌 Deleted Rule Details: %s

🏷️ Command Executed:
	%s

TUFWGo Alert Manager
`,
			e.Action,
			e.Timestamp,
			e.ExecutedBy,
			e.Hostname,
			e.LocalIP,
			DeleteRule,
			e.Command)
	}

	if ssh.GetSSHStatus() {
		remoteIP := ssh.GlobalHost
		remoteUser, err := ssh.CommandStream("whoami")
		remoteHostname, err := ssh.CommandStream("echo $hostname")
		if err != nil {
			fmt.Println("WARNING: Unable to get remote user or hostname:", err)
		}
		parsedSSH := fmt.Sprintf("%s@%s", remoteUser, remoteHostname)
		if e.Action == "Rule Added" {
			message = fmt.Sprintf(`
Hello,
An action was performed on your firewall via TUFWGo.
📌 Action: %s
📌 Timestamp: %s
📌 Executed By: %s
📌 Hostname: %s
📌 Local IP: %s
📌 Machine Affected by SSH: %s -> %s
📌 Rule Details:
	- Action: %s
	- Direction: %s
	- Interface: %s
	- From: %s
	- To: %s
	- Port: %s
	- Protocol: %s
	- App Profile: %s

🏷️ Command Executed:
	%s

TUFWGo Alert Manager
`,
				e.Action,
				e.Timestamp,
				e.ExecutedBy,
				e.Hostname,
				e.LocalIP,
				remoteIP,
				parsedSSH,
				e.Rule.Action,
				direction,
				iface,
				fromIP,
				toIP,
				port,
				protocol,
				appProfile,
				e.Command)
		} else {
			message = fmt.Sprintf(`
Hello,
An action was performed on your firewall via TUFWGo.
📌 Action: %s
📌 Timestamp: %s
📌 Executed By: %s
📌 Hostname: %s
📌 Local IP: %s
📌 Machine Affected by SSH: %s -> %s

🏷️ Command Executed:
	%s

TUFWGo Alert Manager
`,
				e.Action,
				e.Timestamp,
				e.ExecutedBy,
				e.Hostname,
				e.LocalIP,
				remoteIP,
				parsedSSH,
				e.Command)
		}
	}
	return message
}

func (e *EmailInfo) TestEmailData() {
	/*e.Timestamp = time.Now().String()
	e.ExecutedBy = os.Getenv("USER")
	e.Hostname = os.Getenv("hostname")
	localIP, err := getLocalIP()
	if err != nil {
		e.LocalIP = "unknown"
	} else {
		e.LocalIP = localIP
	}
	e.Rule = &ufw.Form{
		Action:     "allow",
		Direction:  "",
		Interface:  "eth0",
		FromIP:     "192.168.1.1",
		ToIP:       "",
		Port:       "22",
		Protocol:   "",
		AppProfile: "",
	}
	cmd, err := e.Rule.ParseForm()
	if err != nil {
		cmd = "N/A"
	}
	e.Command = cmd*/

	e.Timestamp = time.Now().String()
	e.ExecutedBy = os.Getenv("USER")
	e.Hostname = os.Getenv("hostname")
	localIP, err := getLocalIP()
	if err != nil {
		e.LocalIP = "unknown"
	} else {
		e.LocalIP = localIP
	}
	e.Rule = nil
	cmd := "ufw delete 3"

	e.SendMail("Rule Deleted", cmd, e.Rule)
}
