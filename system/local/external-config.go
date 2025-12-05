package local

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

var (
	baseCfgPath  string
	pdcDir       string
	infraDir     string
	playbooksDir string
	varDir       string
)

func InitPaths() {
	baseCfgPath = filepath.Join(GlobalUserCfgDir, "tufwgo")
	pdcDir = filepath.Join(baseCfgPath, "pdc")
	infraDir = filepath.Join(pdcDir, "infra")
	playbooksDir = filepath.Join(infraDir, "playbooks")
	varDir = filepath.Join(baseCfgPath, "vars")
}

func EditEmailList() error {
	emailList := filepath.Join(baseCfgPath, "emails.txt")
	_, err := os.Stat(emailList)
	if err != nil {
		return fmt.Errorf("unable to find email list, run application normally (without flags) to redownload")
	}

	err = newEditor(emailList)
	if err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}
	return nil
}

func EditAnsibleCfg() error {
	ansibleCfg := filepath.Join(infraDir, "ansible.cfg")
	_, err := os.Stat(ansibleCfg)
	if err != nil {
		return fmt.Errorf("unable to find Ansible config file, run application normally (without flags) to redownload")
	}

	err = newEditor(ansibleCfg)
	if err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}
	return nil
}

func EditAnsibleInventory() error {
	ansibleInv := filepath.Join(infraDir, "inventory.ini")
	_, err := os.Stat(ansibleInv)
	if err != nil {
		return fmt.Errorf("unable to find Ansible inventory file, run application normally (without flags) to redownload")
	}

	err = newEditor(ansibleInv)
	if err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}
	return nil
}

func EditSendProfilePlaybook() error {
	spp := filepath.Join(playbooksDir, "send_profile.yml")
	_, err := os.Stat(spp)
	if err != nil {
		return fmt.Errorf("unable to find send_profile playbook, run application normally (without flags) to redownload")
	}

	err = newEditor(spp)
	if err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}
	return nil
}

func EditDeployPlaybook() error {
	dpp := filepath.Join(playbooksDir, "deploy_profile.yml")
	_, err := os.Stat(dpp)
	if err != nil {
		return fmt.Errorf("unable to find deploy_profile playbook, run application normally (without flags) to redownload")
	}

	err = newEditor(dpp)
	if err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}
	return nil
}

func EditSendgridAPI() error {
	sgEnv := filepath.Join(varDir, "sendgrid.env")
	if _, err := os.Stat(sgEnv); err != nil {
		return fmt.Errorf("unable to find deploy_profile playbook, run application normally (without flags) to redownload")
	}

	if err := newEditor(sgEnv); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	if err := godotenv.Load(sgEnv); err != nil {
		return fmt.Errorf("failed to load env vars: %w", err)
	}
	fmt.Println(os.Getenv("SENDGRID_API_KEY"))
	return nil
}
