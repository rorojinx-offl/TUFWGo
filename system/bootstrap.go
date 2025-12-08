package system

import (
	"TUFWGo/alert"
	"TUFWGo/auth"
	"TUFWGo/binaries"
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"TUFWGo/tui"
	"TUFWGo/ufw"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joho/godotenv"
)

var skipTermCheck = flag.Bool("skip-term-check", false, "Skip the terminal size check")
var sshMode = flag.Bool("ssh", false, "Run in SSH mode")
var copilotStp = flag.Bool("copilot-setup", false, "Setup copilot mode")
var email = flag.Bool("email", false, "Edit mailing list")
var ansibleConfig = flag.Bool("ansible-config", false, "Edit Ansible config")
var ansibleInv = flag.Bool("ansible-inv", false, "Edit Ansible inventory")
var spp = flag.Bool("ansible-sppb", false, "Edit the send_profile Ansible playbook")
var dpp = flag.Bool("ansible-dppb", false, "Edit the deploy_profile Ansible playbook")
var mailersend = flag.Bool("mailersend", false, "Add/Edit MailerSend Email API Key")
var help = flag.Bool("help", false, "Show help")
var emailTest = flag.Bool("emailtest", false, "Test if emailing works")
var version = flag.Bool("version", false, "Show version")

func RunTUIMode() {
	flag.Parse()
	local.InitPaths()

	if *email {
		if err := local.EditEmailList(); err != nil {
			fmt.Println(err)
			return
		}
		return
	}
	if *ansibleConfig {
		if err := local.EditAnsibleCfg(); err != nil {
			fmt.Println(err)
			return
		}
		return
	}
	if *ansibleInv {
		if err := local.EditAnsibleInventory(); err != nil {
			fmt.Println(err)
			return
		}
		return
	}
	if *spp {
		if err := local.EditSendProfilePlaybook(); err != nil {
			fmt.Println(err)
			return
		}
		return
	}
	if *dpp {
		if err := local.EditDeployPlaybook(); err != nil {
			fmt.Println(err)
			return
		}
		return
	}
	if *mailersend {
		if err := local.EditMailersendAPI(); err != nil {
			fmt.Println(err)
			return
		}
		return
	}
	if *help {
		flag.PrintDefaults()
		return
	}
	if *emailTest {
		if err := testEmail(); err != nil {
			fmt.Println(err)
			return
		}
		return
	}
	if *version {
		fmt.Printf(binaries.Version)
		return
	}

	initSetup()
	signal, err := checkUpdates()
	if err != nil {
		fmt.Println(err)
		fmt.Println("FATAL ERROR: Failed to check for updates. You cannot continue using TUFWGo without being fully updated - this is for your security!")
		os.Exit(1)
	}
	if signal != "0" {
		fmt.Println(signal)
		fmt.Println("Run \"sudo tufwgo-update\" to perform updates. You must update TUFWGo to continue using it!")
		os.Exit(0)
	}
	fmt.Println("TUFWGo is up to date!")

	if *sshMode {
		if !*skipTermCheck && !local.TermCheck() {
			return
		}
		client, err := ssh.InputSSH()
		if err != nil {
			fmt.Println("SSH Connection Failed:", err)
			return
		}

		label, err := local.RunCommand("uname -snrm")
		if err != nil {
			_ = fmt.Errorf("unable to get system name to generate controller ID: %w", err)
			return
		}
		clientID, pubB64, priv, created, err := auth.EnsureControllerKey(label)
		if err != nil {
			fmt.Println("Failed to load or create controller key:", err)
			return
		}
		if created {
			fmt.Println("Controller ID:", clientID)
			fmt.Println("Public Key:", pubB64)
			out, err := ssh.CommandStream(fmt.Sprintf("%s add-controller --pub %q --label %q", "/usr/bin/tufwgo-auth", pubB64, label))
			if err != nil {
				fmt.Println("Failed to add new controller to allowlist:", err, "\n", out)
				return
			}
			fmt.Println("New controller key created.")
		}
		err = auth.AuthenticateOverSSH(client, clientID, "1.0", "tufwgo-auth", priv)
		if err != nil {
			fmt.Println("Authentication Failed:", err)
			return
		}

		ssh.SetSSHStatus(true)
		tui.RunTUI()
		defer client.Close()
		return
	}
	if !*skipTermCheck && !local.TermCheck() {
		return
	}
	if *copilotStp {
		copilotSetup()
		return
	}
	tui.RunTUI()
}

func initSetup() {
	initDone := false
	cfgDir := local.GlobalUserCfgDir

	baseCfgPath := filepath.Join(cfgDir, "tufwgo")
	authController := filepath.Join(baseCfgPath, "authorised_controllers.json")
	emailList := filepath.Join(baseCfgPath, "emails.txt")
	pdcDir := filepath.Join(baseCfgPath, "pdc")
	pdcLogs := filepath.Join(pdcDir, "logs")
	pdcBin := filepath.Join(pdcDir, "tufwgo-deploy")
	profilesDir := filepath.Join(baseCfgPath, "profiles")
	authBin := "/usr/bin/tufwgo-auth"
	infraDir := filepath.Join(pdcDir, "infra")
	infraInventory := filepath.Join(infraDir, "inventory.ini")
	ansibleCfg := filepath.Join(infraDir, "ansible.cfg")
	playbooksDir := filepath.Join(infraDir, "playbooks")
	sendPlaybook := filepath.Join(playbooksDir, "send_profile.yml")
	deployPlaybook := filepath.Join(playbooksDir, "deploy_profile.yml")
	auditDir := filepath.Join(baseCfgPath, "audit")
	varDir := filepath.Join(baseCfgPath, "vars")
	msEnv := filepath.Join(varDir, "mailersend.env")
	auditKeyEnv := filepath.Join(varDir, "auditkey.env")

	var err error

	if _, err = os.Stat(baseCfgPath); err != nil {
		fmt.Println("TUFWGo config not found, creating config folder...")
		err = os.MkdirAll(baseCfgPath, 0700)
		if err != nil {
			fmt.Println("Failed to create config dir:", err)
			return
		}
		fmt.Printf("Config folder created at %s\n\n", baseCfgPath)
	}

	if _, err = os.Stat(authController); err != nil {
		fmt.Println("Authorised controllers file not found, creating...")
		_, err = os.Create(authController)
		if err != nil {
			fmt.Println("Failed to create authorised controllers file:", err)
			return
		}
		fmt.Printf("Authorised controllers file created at %s\n\n", authController)
	}

	if _, err = os.Stat(emailList); err != nil {
		fmt.Println("Email list file not found, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/misc/emails.txt", emailList, "8922be73ea4f18847da317ff1373b71ad2f16a963c131e7b420c8a0199c95277")
		if err != nil {
			fmt.Println("Failed to download email list file:", err)
			return
		}
		fmt.Printf("Email list file downloaded at %s\n\n", emailList)
	}

	if _, err = os.Stat(pdcDir); err != nil {
		fmt.Println("Profile Distribution Center directory not found, creating...")
		err = os.MkdirAll(pdcDir, 0700)
		if err != nil {
			fmt.Println("Failed to create PDC directory:", err)
			return
		}
		fmt.Printf("PDC directory created at %s\n\n", pdcDir)
	}

	if _, err = os.Stat(pdcLogs); err != nil {
		fmt.Println("PDC logs directory not found, creating...")
		err = os.MkdirAll(pdcLogs, 0700)
		if err != nil {
			fmt.Println("Failed to create PDC logs directory:", err)
			return
		}
		fmt.Printf("PDC logs directory created at %s\n\n", pdcLogs)
	}

	if _, err = os.Stat(pdcBin); err != nil {
		fmt.Println("PDC binary not found, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/binaries/tufwgo-deploy", pdcBin, "2cc0d076b24b354fcb451e94d5b6aaf040f267719d6926426a2b7c462d96ea97")
		if err != nil {
			fmt.Println("Failed to download PDC binary:", err)
			return
		}
		fmt.Printf("PDC binary downloaded at %s\n\n", pdcBin)
	}

	if _, err = os.Stat(profilesDir); err != nil {
		fmt.Println("Profiles directory not found, creating...")
		err = os.MkdirAll(profilesDir, 0700)
		if err != nil {
			fmt.Println("Failed to create profiles directory:", err)
			return
		}
		fmt.Printf("Profiles directory created at %s\n\n", profilesDir)
	}

	if _, err = os.Stat(authBin); err != nil {
		fmt.Println("Auth binary not found at /usr/bin/tufwgo-auth, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/binaries/tufwgo-auth", authBin, "15361a36a6b533e990cbbb99c631dc2a645bfe553a4e25226a2b8a40c4b5c6b9")
		if err != nil {
			fmt.Println("Failed to download auth binary:", err)
			return
		}
		fmt.Printf("Auth binary downloaded at %s\n\n", authBin)
	}

	if _, err = os.Stat(infraDir); err != nil {
		fmt.Println("IaC directory not found, creating...")
		err = os.MkdirAll(infraDir, 0700)
		if err != nil {
			fmt.Println("Failed to create infrastructure directory:", err)
			return
		}
		fmt.Printf("IaC directory created at %s\n\n", infraDir)
	}

	if _, err = os.Stat(infraInventory); err != nil {
		fmt.Println("Ansible inventory file not found, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/infra/inventory.ini", infraInventory, "25b8c96e713334c8717656369e25de8dc86841f7af85571dd4f8887d350dde7e")
		if err != nil {
			fmt.Println("Failed to download Ansible inventory file:", err)
			return
		}
		fmt.Printf("Ansible inventory file downloaded at %s\n\n", infraInventory)
	}

	if _, err = os.Stat(ansibleCfg); err != nil {
		fmt.Println("Ansible config file not found, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/infra/ansible.cfg", ansibleCfg, "bc25eb04b07fa67e6d13e7265fd26f49ba2080c70b10ba7d19da3a7fefdc22a8")
		if err != nil {
			fmt.Println("Failed to download Ansible config file:", err)
			return
		}
		fmt.Printf("Ansible config file downloaded at %s\n\n", ansibleCfg)
	}

	if _, err = os.Stat(playbooksDir); err != nil {
		fmt.Println("Playbooks directory not found, creating...")
		err = os.MkdirAll(playbooksDir, 0700)
		if err != nil {
			fmt.Println("Failed to create playbooks directory:", err)
			return
		}
		fmt.Printf("Playbooks directory created at %s\n\n", playbooksDir)
	}

	if _, err = os.Stat(sendPlaybook); err != nil {
		fmt.Println("Profile flight playbook not found, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/infra/playbooks/send_profile.yml", sendPlaybook, "658de44a4212e55569c6f540274f890c66796145da2c22ab20796cfe98ee4691")
		if err != nil {
			fmt.Println("Failed to download profile flight playbook:", err)
			return
		}
		fmt.Printf("Profile flight playbook downloaded at %s\n\n", sendPlaybook)
	}

	if _, err = os.Stat(deployPlaybook); err != nil {
		fmt.Println("Profile deployment playbook not found, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/infra/playbooks/deploy_profile.yml", deployPlaybook, "35d2bdc976f1ad5dc79d1975bfd952b0c5be723bbe45734245c9f05c8b64576e")
		if err != nil {
			fmt.Println("Failed to download profile deployment playbook:", err)
			return
		}
		fmt.Printf("Profile deployment playbook downloaded at %s\n\n", deployPlaybook)
	}

	if _, err = os.Stat(auditDir); err != nil {
		fmt.Println("Audit directory not found, creating...")
		err = os.MkdirAll(auditDir, 0700)
		if err != nil {
			fmt.Println("Failed to create audit directory:", err)
			return
		}
		fmt.Printf("Audit directory created at %s\n\n", auditDir)
	}

	if _, err = os.Stat(varDir); err != nil {
		fmt.Println("Environment Variables directory not found, creating...")
		err = os.MkdirAll(varDir, 0700)
		if err != nil {
			fmt.Println("Failed to create environment variables directory:", err)
			return
		}
		fmt.Printf("Environment Variables directory created at %s\n\n", varDir)
	}

	if _, err = os.Stat(msEnv); err != nil {
		fmt.Println("SendGrid environment file not found, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/vars/mailersend.env", msEnv, "865b3acad81f3ae55d5763d94b434b3674ae49bb1c75dda5fe270bc06a4bb1f8")
		if err != nil {
			fmt.Println("Failed to download SendGrid environment file:", err)
			return
		}
		fmt.Printf("SendGrid environment file downloaded at %s\n\n", msEnv)
	}

	if _, err = os.Stat(auditKeyEnv); err != nil {
		fmt.Println("Audit key environment file not found, downloading...")
		err = local.DownloadFile("https://dl.tufwgo.store/vars/auditkey.env", auditKeyEnv, "c20b42e3705265b35ae46c9d27714572eb0f8b0229eed2f3b68f072871bdc895")
		if err != nil {
			fmt.Println("Failed to download audit key environment file:", err)
			return
		}
		fmt.Printf("Audit key environment file downloaded at %s\n\n", auditKeyEnv)

		fmt.Println("Generating new audit key...")
		key, err := local.RunCommand("openssl rand -base64 32")
		if err != nil {
			fmt.Println("Failed to generate audit key:", err)
			return
		}
		err = local.EditEnv(auditKeyEnv, "TUFWGO_AUDIT_KEY", key)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Print("Saved audit key to env file\n\n")
	}

	err = godotenv.Load(auditKeyEnv)
	if err != nil {
		fmt.Println("Failed to load audit key:", err)
		return
	}

	err = godotenv.Load(msEnv)
	if err != nil {
		fmt.Println("Failed to load SendGrid env file:", err)
		return
	}
	if os.Getenv("MAILERSEND_API_KEY") == "" {
		fmt.Print("$MAILERSEND_API_KEY not set in mailersend.env. Please edit the file and add your MailerSend API key to enable email notifications.\n\n")
	} else {
		err = godotenv.Load(msEnv)
		if err != nil {
			fmt.Println("Failed to load MailerSend API key:", err)
			return
		}
		initDone = true
	}

	if initDone {
		fmt.Println("Initial setup completed.")
	}

	//runTUIMode()
}

func copilotSetup() {
	initdone := false
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Failed to get user config dir:", err)
		return
	}

	baseCfgPath := filepath.Join(cfgDir, "tufwgo")
	copilotDir := filepath.Join(baseCfgPath, "copilot")
	ggufModel := filepath.Join(copilotDir, "mistral-7b-instruct-v0.2.Q4_K_M.gguf")

	if _, err = os.Stat(copilotDir); err != nil {
		fmt.Println("Copilot directory not found, creating...")
		err = os.MkdirAll(copilotDir, 0700)
		if err != nil {
			fmt.Println("Failed to create copilot directory:", err)
			return
		}
		fmt.Printf("Copilot directory created at %s\n\n", copilotDir)
	}

	if _, err = exec.LookPath("ollama"); err != nil {
		fmt.Println("Ollama not found, installing...")
		err = local.CommandLiveOutput("curl -fsSL https://ollama.com/install.sh | sh")
		if err != nil {
			fmt.Println("Failed to install ollama:", err)
			return
		}
		fmt.Print("Ollama installed!\n\n")
	}

	if _, err = os.Stat(ggufModel); err != nil {
		fmt.Println("LM not found, downloading...")
		err = local.DownloadFile("https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.2-GGUF/resolve/main/mistral-7b-instruct-v0.2.Q4_K_M.gguf?download=true", ggufModel, "3e0039fd0273fcbebb49228943b17831aadd55cbcbf56f0af00499be2040ccf9")
		if err != nil {
			fmt.Println("Failed to download LM:", err)
			return
		}
		fmt.Printf("LM downloaded at %s\n\n", ggufModel)
		initdone = true
	}

	if initdone {
		fmt.Println("Copilot setup completed.")
	} else {
		fmt.Println("Copilot already set up.")
	}
}

func testEmail() error {
	if os.Getenv("MAILERSEND_API_KEY") == "" {
		return errors.New("unable to find MailerSend API key")
	}

	var rule *ufw.Form
	rule = &ufw.Form{
		Action:    "Test Email",
		Direction: "Towards Rome",
		Interface: "NIC0",
		FromIP:    "127.0.0.1",
		ToIP:      "127.0.0.1",
		Port:      "69",
		Protocol:  "SMTP",
	}

	emailInfo := alert.EmailInfo{}
	emailInfo.SendMail("Test Email Sent", "tufwgo -emailtest", rule)
	return nil
}
