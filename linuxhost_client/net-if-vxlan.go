package linuxhost_client

import "fmt"

type IfVxlan struct {
	IfCommon
	Vni  uint32
	Port uint32
}

var _ IsIf = &IfVxlan{}

func (m *IfVxlan) GetCommon() *IfCommon {
	return &m.IfCommon
}

func CreateIfVXLAN(connectedClient *SSHClientContext, iface *IfVxlan) (*IfVxlan, error) {
	cmd := fmt.Sprintf("sudo ip link add %s type vxlan id %d dstport %d", iface.Name, iface.Vni, iface.Port)
	fmt.Println("DO CMD: " + cmd)
	result, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		fmt.Println("Error!!!" + err.Error())
		return nil, err
	}
	fmt.Println(result)
	if err := IfSetCommon(connectedClient, iface); err != nil {
		return nil, *err
	}

	return iface, nil
}
