package provider

import "terraform-provider-linuxhost/linuxhost_client"

type LinuxhostCommonResource struct {
	hostData *linuxhost_client.HostData
}
type IsLinuxhostIFResource interface {
	GetHostData() *linuxhost_client.HostData
}
