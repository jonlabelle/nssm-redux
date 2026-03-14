//go:build windows

package scm

import (
	"fmt"
	"strings"
)

const (
	localSystemAccount    = "LocalSystem"
	localServiceAccount   = `NT AUTHORITY\LocalService`
	networkServiceAccount = `NT AUTHORITY\NetworkService`
	virtualAccountPrefix  = `NT Service\`
)

func GetObjectName(name string) (string, error) {
	manager, serviceHandle, cfg, err := openService(name)
	if err != nil {
		return "", err
	}
	defer manager.Disconnect()
	defer serviceHandle.Close()

	if strings.TrimSpace(cfg.ServiceStartName) == "" {
		return localSystemAccount, nil
	}
	return cfg.ServiceStartName, nil
}

func SetObjectName(name, username, password string) error {
	manager, serviceHandle, cfg, err := openService(name)
	if err != nil {
		return err
	}
	defer manager.Disconnect()
	defer serviceHandle.Close()

	account, secret, err := normalizeObjectName(name, username, password)
	if err != nil {
		return err
	}

	cfg.ServiceStartName = account
	cfg.Password = secret
	if err := serviceHandle.UpdateConfig(cfg); err != nil {
		return fmt.Errorf("update service account: %w", err)
	}
	return nil
}

func ResetObjectName(name string) error {
	return SetObjectName(name, localSystemAccount, "")
}

func normalizeObjectName(serviceName, username, password string) (string, string, error) {
	user := strings.TrimSpace(username)
	switch {
	case user == "":
		return "", "", fmt.Errorf("ObjectName requires a username")
	case strings.EqualFold(user, "localsystem"),
		strings.EqualFold(user, "system"),
		strings.EqualFold(user, `NT AUTHORITY\System`):
		return localSystemAccount, "", nil
	case strings.EqualFold(user, "localservice"),
		strings.EqualFold(user, "local service"),
		strings.EqualFold(user, `NT AUTHORITY\LocalService`),
		strings.EqualFold(user, `NT AUTHORITY\Local Service`):
		return localServiceAccount, "", nil
	case strings.EqualFold(user, "networkservice"),
		strings.EqualFold(user, "network service"),
		strings.EqualFold(user, `NT AUTHORITY\NetworkService`),
		strings.EqualFold(user, `NT AUTHORITY\Network Service`):
		return networkServiceAccount, "", nil
	case strings.EqualFold(user, virtualAccountPrefix+serviceName):
		return virtualAccountPrefix + serviceName, "", nil
	case strings.TrimSpace(password) == "":
		return "", "", fmt.Errorf("ObjectName for %q requires a password", user)
	default:
		return user, password, nil
	}
}
