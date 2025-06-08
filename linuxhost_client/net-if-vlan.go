package linuxhost_client

import "fmt"

type IfVlan struct {
	IfCommon
	Vid        uint32
	Parent string
}

var _ IsIf = &IfVlan{}

func (m *IfVlan) GetCommon() *IfCommon {
	return &m.IfCommon
}

func CreateIfVlan(connectedClient *SSHClientContext, iface *IfVlan) (*IfVlan, error) {
	cmd := fmt.Sprintf("sudo ip link add link %s name %s type vlan id %d", iface.Parent, iface.Name, iface.Vid)
	result, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		return nil, err
	}
	fmt.Println(result)
	if err := IfSetCommon(connectedClient, iface); err != nil {
		return nil, *err
	}

	return iface, nil
}
