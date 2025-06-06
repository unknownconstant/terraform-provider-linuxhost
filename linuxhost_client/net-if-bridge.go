package linuxhost_client

import "fmt"

type IfBridge struct {
	IfCommon
}

var _ IsIf = &IfBridge{}

func (m *IfBridge) GetCommon() *IfCommon {
	return &m.IfCommon
}

func CreateIfBridge(connectedClient *SSHClientContext, iface *IfBridge) (*IfBridge, error) {
	cmd := fmt.Sprintf("sudo ip link add %s type bridge", iface.Name)
	result, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		return nil, err
	}
	fmt.Println(result)
	if err := IfSetState(connectedClient, iface); err != nil {
		return nil, *err
	}
	return iface, nil
}
