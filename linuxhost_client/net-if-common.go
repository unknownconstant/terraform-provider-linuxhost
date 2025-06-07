package linuxhost_client

import (
	"fmt"
)

func IfSetCommon(connectedClient *SSHClientContext, ifaceX IsIf) *error {
	steps := []func(*SSHClientContext, IsIf) *error{
		IfSetState,
		IfSetBridgeMaster,
	}
	for _, step := range steps {
		if err := step(connectedClient, ifaceX); err != nil {
			return err
		}
	}
	return nil
}

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

func IfSetBridgeMaster(connectedClient *SSHClientContext, ifaceX IsIf) *error {
	iface := ifaceX.GetCommon()
	var cmd string
	
	if iface.BridgeMember == nil {
		cmd = fmt.Sprintf("sudo ip link set %s nomaster", iface.Name)
	} else {
		cmd = fmt.Sprintf("sudo ip link set %s master %s", iface.Name, iface.BridgeMember.Name)
	}
	result, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		return &err
	}
	fmt.Println(result)
	return nil
}
