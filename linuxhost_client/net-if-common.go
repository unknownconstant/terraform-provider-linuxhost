package linuxhost_client

import (
	"fmt"
)

func IfSetState(connectedClient *SSHClientContext, ifaceX IsIf) *error {
	iface := ifaceX.GetCommon()
	if iface.State == "" {
		return nil
	}
	cmd := fmt.Sprintf("sudo ip link set %s %s", iface.Name, iface.State)
	result, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		return &err
	}
	fmt.Println(result)

	return nil
}
