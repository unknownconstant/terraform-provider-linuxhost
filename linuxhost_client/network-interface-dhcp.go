package linuxhost_client

import (
	"fmt"
	"regexp"
	"strings"
)

func SetDhcp(connectedClient *SSHClientContext, adapterName string, dhcpMode string, enabled bool) error {
	// fmt.Println("Enabling DHCP")
	cmd := fmt.Sprintf("sudo dhclient -4 -v -i -pf /run/dhclient.%s.pid -lf /var/lib/dhcp/dhclient.%s.leases -I -df /var/lib/dhcp/dhclient6.%s.leases %%s %s", adapterName, adapterName, adapterName, adapterName)
	// fmt.Println("Base cmd: ", cmd)
	if enabled {
		cmd = fmt.Sprintf(cmd+" &", "")
	} else {
		cmd = fmt.Sprintf(cmd, " -r")
	}
	// fmt.Println("Cmd: ", cmd)
	result, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		fmt.Println("Error!!" + err.Error())
		return err
	}
	fmt.Println(result)
	return nil
}

func RefreshDhcp(hostData *HostData, adapters []*AdapterInfo) error {
	cmd := "sudo ps -ef | grep dhclient"
	fmt.Println("cmd: ", cmd)
	result, err := hostData.Client.ExecuteCommand(cmd)
	if err != nil {
		return err
	}
	fmt.Println("SSH Result (refresh DHCP): ", result)
	adaptersSlice := AdapterInfoListToMap(adapters)
	ParseDhclient(result, &adaptersSlice)
	return nil
}

func ParseDhclient(stdout string, adapters *map[string]*AdapterInfo) {
	dhclientRegex := regexp.MustCompile(`dhclient.* (.*)$`)

	// Split output into lines
	lines := strings.Split(stdout, "\n")

	// _adapters := *adapters

	for _, line := range lines {
		line = strings.TrimSpace(line)
		fmt.Println("Checking line", line)
		// Match adapter lines
		if match := dhclientRegex.FindStringSubmatch(line); match != nil {
			fmt.Println("Checking for adapter", match[1])
			adapterMap := *adapters
			adapter, found := adapterMap[match[1]]
			// adapter2, found2 := adapterMap[match[1]]
			// _ = found2
			fmt.Println("found?", found)
			if !found {
				continue
			}
			// fmt.Println("CCCC", adapter, adapter2)
			fmt.Println(adapter)
			// fmt.Println(adapter2)
			fmt.Printf("Pointer address 1: %p\n", adapter)
			// fmt.Printf("Pointer address 2: %p\n", adapter2)
			fmt.Println("Found: ", adapter)
			S := "dhclient"

			fmt.Println("Set dhclient!!!")
			adapter.DHCP = &S
			// (_adapters)[match[1]] = adapter
		}
	}
	println("Pointers!", adapters)
}
