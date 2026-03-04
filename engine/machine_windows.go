//go:build windows

package engine

import (
	"golang.org/x/sys/windows/registry"
)

func platformMachineID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.READ|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}
	defer k.Close()

	val, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", err
	}
	return val, nil
}
