package main

import (
	"TUFWGo/auth"
	"TUFWGo/system/local"
	"TUFWGo/system/ssh"
	"TUFWGo/tui"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var skipTermCheck = flag.Bool("skip-term-check", false, "Skip the terminal size check")
var sshMode = flag.Bool("ssh", false, "Run in SSH mode")

func main() {
	local.RequireRoot()
	//runTUIMode()
	//samples.Input()
	//tui.RunForm()
	//alert.SendSampleMail()
	//tui.RunCreateProfile()
	initSetup()
}

func runTUIMode() {
	flag.Parse()
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
	tui.RunTUI()
}

func initSetup() {
	initDone := false
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Failed to get user config dir:", err)
		return
	}
	baseCfgPath := filepath.Join(cfgDir, "tufwgo")
	authController := filepath.Join(baseCfgPath, "authorised_controllers.json")
	pdcDir := filepath.Join(baseCfgPath, "pdc")
	pdcLogs := filepath.Join(pdcDir, "logs")
	pdcBin := filepath.Join(pdcDir, "tufwgo-deploy")
	profilesDir := filepath.Join(baseCfgPath, "profiles")
	authBin := "/usr/bin/tufwgo-auth"

	if _, err = os.Stat(baseCfgPath); err != nil {
		fmt.Println("TUFWGo config not found, creating config folder...")
		err = os.MkdirAll(baseCfgPath, 0700)
		if err != nil {
			fmt.Println("Failed to create config dir:", err)
			return
		}
		fmt.Println("Config folder created at", baseCfgPath)
	}

	if _, err = os.Stat(authController); err != nil {
		fmt.Println("Authorised controllers file not found, creating...")
		_, err = os.Create(authController)
		if err != nil {
			fmt.Println("Failed to create authorised controllers file:", err)
			return
		}
		fmt.Println("Authorised controllers file created at", authController)
	}

	if _, err = os.Stat(pdcDir); err != nil {
		fmt.Println("Profile Distribution Center directory not found, creating...")
		err = os.MkdirAll(pdcDir, 0700)
		if err != nil {
			fmt.Println("Failed to create PDC directory:", err)
			return
		}
		fmt.Println("PDC directory created at", pdcDir)
	}

	if _, err = os.Stat(pdcLogs); err != nil {
		fmt.Println("PDC logs directory not found, creating...")
		err = os.MkdirAll(pdcLogs, 0700)
		if err != nil {
			fmt.Println("Failed to create PDC logs directory:", err)
			return
		}
		fmt.Println("PDC logs directory created at", pdcLogs)
	}

	if _, err = os.Stat(pdcBin); err != nil {
		fmt.Println("PDC binary not found, downloading...")
		err = local.DownloadFile("https://txrijwxmwfoempqmsuva.supabase.co/storage/v1/object/public/deploy%20binary/tufwgo-deploy", pdcBin, "2cc0d076b24b354fcb451e94d5b6aaf040f267719d6926426a2b7c462d96ea97")
		if err != nil {
			fmt.Println("Failed to download PDC binary:", err)
			return
		}
		fmt.Println("PDC binary downloaded at", pdcBin)
	}

	if _, err = os.Stat(profilesDir); err != nil {
		fmt.Println("Profiles directory not found, creating...")
		err = os.MkdirAll(profilesDir, 0700)
		if err != nil {
			fmt.Println("Failed to create profiles directory:", err)
			return
		}
		fmt.Println("Profiles directory created at", profilesDir)
	}

	if _, err = os.Stat(authBin); err != nil {
		fmt.Println("Auth binary not found at /usr/bin/tufwgo-auth, downloading...")
		err = local.DownloadFile("https://txrijwxmwfoempqmsuva.supabase.co/storage/v1/object/public/deploy%20binary/tufwgo-auth", authBin, "15361a36a6b533e990cbbb99c631dc2a645bfe553a4e25226a2b8a40c4b5c6b9")
		if err != nil {
			fmt.Println("Failed to download auth binary:", err)
			return
		}
		fmt.Println("Auth binary downloaded at", authBin)
		initDone = true
	}
	if initDone {
		fmt.Println("Initial setup completed.")
	}
	runTUIMode()
}
