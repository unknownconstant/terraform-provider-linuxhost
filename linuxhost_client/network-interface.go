package linuxhost_client

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NetworkInterface struct {
	Id              string
	Type            string
	ParentInterface string
	VLAN_id         int
}

type IfCommon struct {
	Name         string
	Mac          string
	State        string
	BridgeMember *IfBridgeMember

	IPv4 []models.IPWithSubnet
	IPv6 []models.IPWithSubnet
}
type IfBridgeMember struct {
	Name string
}
type IsIf interface {
	GetCommon() *IfCommon
}

func CreateInterface(connectedClient *SSHClientContext, adapter *NetworkInterface) (*NetworkInterface, error) {
	fmt.Println("adding interface")
	if connectedClient == nil {
		fmt.Println("....")
		return nil, errors.New("DON'T HAVE A CONNECTOR")
	}
	cmd := ""
	if adapter.Type == "vlan" {
		// 'sudo ip link add link enxb827eb20b2ba name eth0.108 type vlan id 108'
		cmd = fmt.Sprintf("sudo ip link add link %s name %s type vlan id %s", adapter.ParentInterface, adapter.Id, strconv.Itoa(adapter.VLAN_id))
	} else if adapter.Type == "dummy" {
		cmd = fmt.Sprintf("sudo ip link add %s type dummy", adapter.Id)
	} else {

		fmt.Println("Error!!! unknown adapter type!! \"" + adapter.Type + "\"")
		return nil, errors.New("Unknown interface type" + adapter.Type)
	}
	fmt.Println("command is " + cmd + ";")
	result, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		fmt.Println("Error!!!" + err.Error())
		return nil, err
	}
	fmt.Println(result)

	iface := &NetworkInterface{
		Id: result,
	}
	return iface, nil
}

func InterfaceUpDown(cli *SSHClientContext, Id string, UpDown string) error {
	cmd := fmt.Sprintf("sudo ip link set dev %s %s; sleep 1", Id, UpDown)
	fmt.Println("DO CMD: " + cmd)
	result, err := cli.ExecuteCommand(cmd)

	if err != nil {
		fmt.Println("Error!!!" + err.Error())
		return err
	}
	fmt.Println(result)
	return nil
}

func DeleteInterface(connectedClient *SSHClientContext, Id string) (bool, error) {
	fmt.Println("deleting interface")
	result, err := connectedClient.ExecuteCommand(fmt.Sprintf("sleep 1; sudo ip link del %s", Id))

	if err != nil {
		fmt.Println("Error!!!" + err.Error())
		fmt.Println(result)
		return false, err
	}
	return true, nil
}

func AssignIP(connectedClient *SSHClientContext, adapterName string, ip string) (*models.NetowrkInterfaceIPAssignmentModel, error) {
	fmt.Println("Assigning IP")
	result, err := connectedClient.ExecuteCommand(fmt.Sprintf("sudo ip addr add %s dev %s", ip, adapterName))
	if err != nil {
		fmt.Println("Error!!" + err.Error())
		return nil, err
	}
	fmt.Println(result)
	assignment := &models.NetowrkInterfaceIPAssignmentModel{
		InterfaceName: types.StringValue(adapterName),
		IPv4:          types.StringValue(ip),
	}
	return assignment, nil
}
func DeleteIP(connectedClient *SSHClientContext, adapterName string, ip string) error {
	cmd := fmt.Sprintf("sudo ip addr del %s dev %s", ip, adapterName)
	fmt.Println("Deleting IP: " + cmd)
	result, err := connectedClient.ExecuteCommand(cmd)
	if err != nil {
		fmt.Println("Error!!" + err.Error())
		fmt.Println(result)
		return err
	}
	return nil
}

type AdapterInfo struct {
	Name             string
	MAC              string
	Up               bool
	IPv4             []models.IPWithSubnet
	IPv6             []models.IPWithSubnet
	Type             string
	Vlan             *int
	Vni              *int64
	Port             *int32
	ParentInterface  *string
	DHCP             *string
	BridgeInfo       *BridgeInfo
	DesignatedBridge *string
}

type AdapterInfoSlice []*AdapterInfo

func (l *AdapterInfoSlice) GetBridgeId(bridgeId string) *AdapterInfo {
	for _, a := range *l {
		if a.BridgeInfo == nil {
			continue
		}
		if a.BridgeInfo.BridgeId != bridgeId {
			continue
		}
		return a
	}
	return nil
}

func (l *AdapterInfoSlice) GetByName(name string) *AdapterInfo {
	for _, a := range *l {
		if a.Name == name {
			return a
		}
	}
	return nil
}

func (l *AdapterInfoSlice) Clear() {
	*l = nil
}

type BridgeInfo struct {
	BridgeId      string
	VlanFiltering bool
}

func AdapterInfoListToMap(items []*AdapterInfo) map[string]*AdapterInfo {
	result := make(map[string]*AdapterInfo, len(items)) // Preallocate map size for efficiency
	for _, item := range items {
		result[item.Name] = item
	}
	return result
}
func ListToMap[T any, K comparable](items []T, keyExtractor func(T) K) map[K]T {
	result := make(map[K]T, len(items)) // Preallocate map size for efficiency
	for _, item := range items {
		result[keyExtractor(item)] = item
	}
	return result
}

func AdapterInfoSliceToPointers(items []AdapterInfo) []*AdapterInfo {
	result := make([]*AdapterInfo, len(items)) // Preallocate map size for efficiency
	for i := range items {
		result[i] = &items[i]
	}
	return result
}

type HostData struct {
	Client     *SSHClientContext
	Interfaces AdapterInfoSlice
	Users      []models.UserModel
	Groups     []models.GroupModel
	Hostname   *string
}

func RefreshAdapters(hostData *HostData) (AdapterInfoSlice, error) {
	stmt := "ip -d a"
	fmt.Println(stmt)
	result, err := hostData.Client.ExecuteCommand(stmt)
	if err != nil {
		return nil, err
	}
	fmt.Println("SSH RESULT", result)
	adapterInfo := ParseAdapters(result)

	// fmt.Println("Refreshing DHCP")
	// fmt.Println(adapterInfo)
	// AI := AdapterInfoSliceToPointers(adapterInfo)
	err = RefreshDhcp(hostData, adapterInfo)
	// fmt.Println("Finished refreshing DHCP")
	// for i, _ := range adapterInfo {
	// fmt.Println("---Done")
	// fmt.Printf("Adapter: %p", &adapterInfo[i])
	// fmt.Printf("Adapter: %p", AI[i])
	// }
	// fmt.Println(adapterInfo)

	hostData.Interfaces = adapterInfo
	if err != nil {
		return nil, err
	}
	i := hostData.Interfaces
	return AdapterInfoSlice(i), nil
}
func ReadAdapters(hostData *HostData) (AdapterInfoSlice, error) {
	if hostData.Interfaces == nil {
		return RefreshAdapters(hostData)
	}
	return hostData.Interfaces, nil
}

// parseAdapters extracts adapter information from the `ip a` output
func ParseAdapters(ipOutput string) AdapterInfoSlice {
	var adapters AdapterInfoSlice

	// Regular expressions to match interface lines and IP addresses with subnets
	adapterRegex := regexp.MustCompile(`^\d+: ([a-zA-Z0-9\._-]+)[a-zA-Z0-9@\.-]*:`)
	ipv4Regex := regexp.MustCompile(`inet (\d+\.\d+\.\d+\.\d+)/(\d+)`)
	ipv6Regex := regexp.MustCompile(`inet6 ([a-fA-F0-9:]+)/(\d+)`)
	macRegex := regexp.MustCompile(`(?:ether|loopback)\s*(([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2}))`)
	upRegex := regexp.MustCompile(`state (UP|DOWN|UNKNOWN)`)
	noArpRegex := regexp.MustCompile(`<.*(NOARP).*>`)
	vlanRegex := regexp.MustCompile(`vlan protocol 802\.1Q id (\d+)`)
	vxlanRegex := regexp.MustCompile(`vxlan id (\d+).*dstport (\d+)`)
	parentInterfaceRegex := regexp.MustCompile(`@([^:]+):`)

	bridgeInfoRegex := regexp.MustCompile(`bridge.*vlan_filtering ([01]).*bridge_id ([^\s]+)`)
	bridgeMemberRegex := regexp.MustCompile(`bridge_slave.*designated_bridge ([^\s]+)`)

	// Split output into lines
	lines := strings.Split(ipOutput, "\n")

	var currentAdapter *AdapterInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Match adapter lines
		if match := adapterRegex.FindStringSubmatch(line); match != nil {
			if currentAdapter != nil {
				// Save the previous adapter
				adapters = append(adapters, currentAdapter)
			}

			// Start a new adapter
			name := match[1]
			fmt.Println("Interface name is " + name + ";")
			// if idx := strings.Index(name, "@"); idx != -1 {
			// 	name = name[:idx] // Remove any "@..." suffix
			// }
			currentAdapter = &AdapterInfo{
				Name: name,
				Up:   false,
				Type: "unknown",
			}

			// Match Up Down
			if match := upRegex.FindStringSubmatch(line); match != nil {
				currentAdapter.Up = match[1] != "DOWN"
			}

			// Match interface type dummy
			if match := noArpRegex.FindStringSubmatch(line); match != nil {
				currentAdapter.Type = "dummy"
			} else {
			}

			// Match base interface
			if match := parentInterfaceRegex.FindStringSubmatch(line); match != nil {
				fmt.Println("parentInterface is: \"" + match[1] + "\"")
				if currentAdapter.ParentInterface == nil {
					currentAdapter.ParentInterface = new(string)
				}
				*currentAdapter.ParentInterface = match[1]
			}

			continue
		}
		if currentAdapter == nil {
			continue
		}

		// Match IPv4 addresses
		if match := ipv4Regex.FindStringSubmatch(line); match != nil {
			currentAdapter.IPv4 = append(currentAdapter.IPv4, models.IPWithSubnet{
				IP:     match[1],
				Subnet: match[2],
			})
		}

		// Match IPv6 addresses
		if match := ipv6Regex.FindStringSubmatch(line); match != nil {
			currentAdapter.IPv6 = append(currentAdapter.IPv6, models.IPWithSubnet{
				IP:     match[1],
				Subnet: match[2],
			})
		}

		// Match MAC
		if match := macRegex.FindStringSubmatch(line); match != nil {
			currentAdapter.MAC = match[1]
			fmt.Println("MAC: " + match[1])
		}

		if match := bridgeMemberRegex.FindStringSubmatch(line); match != nil {
			currentAdapter.DesignatedBridge = &match[1]
		}

		// Interface specific

		// Match VLAN
		if match := vlanRegex.FindStringSubmatch(line); match != nil {
			res, err := strconv.Atoi(match[1])
			if err != nil {
				fmt.Println("Error converting str to int")
			}
			currentAdapter.Vlan = &res
			currentAdapter.Type = "vlan"
			fmt.Println("vlan: " + match[1])
		}

		// Match vxlan
		if match := vxlanRegex.FindStringSubmatch(line); match != nil {
			vni, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				fmt.Println("Error converting vni str to int")
			}
			currentAdapter.Vni = &vni
			port, err := strconv.ParseInt(match[2], 10, 64)
			if err != nil {
				fmt.Println("Error converting port str to int")
			}
			p := int32(port)
			currentAdapter.Port = &p
			currentAdapter.Type = "vxlan"
			fmt.Println("vxlan: " + match[1])
		}

		if match := bridgeInfoRegex.FindStringSubmatch(line); match != nil {
			currentAdapter.Type = "bridge"
			currentAdapter.BridgeInfo = &BridgeInfo{
				BridgeId:      match[2],
				VlanFiltering: match[1] == "1",
			}
			// currentAdapter.BridgeId = match[2]
			// fmt.Println("bridgeId: " + match[2])
		}
	}

	// Add the last adapter if any
	if currentAdapter != nil {
		adapters = append(adapters, currentAdapter)
	}

	return adapters
}
