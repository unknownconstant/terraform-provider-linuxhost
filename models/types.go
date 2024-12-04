package linuxhost_models

type IPWithSubnet struct {
	IP     string
	Subnet string
}

func (ip *IPWithSubnet) String() string {
	return ip.IP + "/" + ip.Subnet
}
func IPList(IPs []IPWithSubnet) []string {
	values := []string{}
	for _, i := range IPs {
		values = append(values, i.String())
	}
	return values
}
