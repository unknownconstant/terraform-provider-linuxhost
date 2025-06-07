package linuxhost_client

import "fmt"

type IfVethPeer struct {
	IfCommon
}

var _ IsIf = &IfVethPeer{}

func (m *IfVethPeer) GetCommon() *IfCommon {
	return &m.IfCommon
}

type IfVethPair struct {
	Local IfVethPeer
	Peer  IfVethPeer
}

func CreateIfVeth(connectedClient *SSHClientContext, iface *IfVethPair) (*IfVethPair, error) {
	cmd := fmt.Sprintf("sudo ip link add %s type veth peer name %s", iface.Local.Name, iface.Peer.Name)
	_, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		return nil, err
	}
	if err := IfSetCommon(connectedClient, &iface.Local); err != nil {
		return nil, *err
	}
	if err := IfSetCommon(connectedClient, &iface.Peer); err != nil {
		return nil, *err
	}
	return iface, nil
}
