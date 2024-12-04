package linuxhost_client

import (
	"fmt"
	"regexp"
	"strings"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type groupCommand struct {
	cmd      string
	complete bool
}

func newGroupCommand(createNew bool) groupCommand {
	initialCommand := ""
	if createNew {
		initialCommand = "sudo groupadd"
	} else {
		initialCommand = "sudo groupmod"
	}
	G := groupCommand{
		cmd:      initialCommand,
		complete: false,
	}
	return G
}
func (g *groupCommand) wrapArgument(cb func()) *groupCommand {
	if g.complete {
		fmt.Println("error, command already complete")
		return g
	}
	cb()
	return g
}
func (g *groupCommand) withGid(v types.Number) *groupCommand {
	return g.wrapArgument(func() {
		f, _ := v.ValueBigFloat().Float64()
		g.cmd = g.cmd + fmt.Sprintf(" -g %d", int(f))
	})
}
func (g *groupCommand) withModifiedName(name types.String) *groupCommand {
	return g.wrapArgument(func() {
		g.cmd = g.cmd + " -n " + name.ValueString()
	})
}
func (g *groupCommand) withCurrentName(name string) {
	g.cmd = g.cmd + " " + name
	g.complete = true
}
func buildGroupCommand(group *models.GroupModel, targetGroup *string) groupCommand {
	g := newGroupCommand(targetGroup == nil)
	if !group.GID.IsNull() && !group.GID.IsUnknown() {
		g.withGid(group.GID)
	}
	if targetGroup == nil {
		g.withCurrentName(group.Name.ValueString())
	} else {
		g.withModifiedName(group.Name).withCurrentName(*targetGroup)
	}
	return g
}

func SetGroup(clientContext *SSHClientContext, group *models.GroupModel, targetGroup *string) error {
	fmt.Println("creating group")
	fmt.Println(group)

	g := buildGroupCommand(group, targetGroup)
	fmt.Println("New group cmd: ", g.cmd)
	r := NewSSHCommandContext(clientContext).Exec(g.cmd)
	if r.Error != nil {
		fmt.Println("Error creating group", r.Error.Error(), r.Output)
		return r.Error
	}
	fmt.Println("Created group", r.Output)
	return nil
}

func DeleteGroup(clientContext *SSHClientContext, group *models.GroupModel) error {
	cmd := fmt.Sprintf("sudo groupdel %s", group.Name.ValueString())
	result := NewSSHCommandContext(clientContext).Exec(cmd)
	fmt.Println("Groupdel result was: ", result)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Function to check if a string is in a SetValue
func stringInSet(set types.Set, value types.String) bool {
	_value := value.ValueString()
	for _, elem := range set.Elements() {
		if strVal, ok := elem.(types.String); ok {
			if strVal.ValueString() == _value {
				return true
			}
		}
	}
	return false
}

func UserGroups(groups []models.GroupModel, user models.UserModel) types.Set {
	groupNames := []attr.Value{user.PrimaryGroup}
	for _, group := range groups {
		if !stringInSet(group.Members, user.Username) {
			continue
		}
		if user.PrimaryGroup.Equal(group.Name) {
			continue
		}
		groupNames = append(groupNames, types.StringValue(group.Name.ValueString()))
	}

	return types.SetValueMust(types.StringType, groupNames)
}

func RefreshGroups(HostData *HostData) ([]models.GroupModel, error) {
	cmd := "cat /etc/group"
	result, err := CommandRunner(HostData, cmd)
	if err != nil {
		return nil, err
	}
	groups := []models.GroupModel{}
	parseGroup := func(line string) {
		groupRegex := regexp.MustCompile(`^([^:]*):([^:]*):([^:]*):(.*)$`)

		match := groupRegex.FindStringSubmatch(line)
		if match == nil {
			return
		}
		membersSlice := strings.Split(match[4], ",")
		membersValueSlice := []attr.Value{}
		for _, member := range membersSlice {
			if member == "" {
				continue
			}
			membersValueSlice = append(membersValueSlice, types.StringValue(member))
		}
		group := models.GroupModel{
			GID:     strToTFNumber(match[3]),
			Name:    types.StringValue(match[1]),
			Members: types.SetValueMust(types.StringType, membersValueSlice),
		}
		groups = append(groups, group)
	}
	LineMatcher(*result, parseGroup)
	return groups, nil
}
func GetGroups(HostData *HostData) ([]models.GroupModel, error) {
	if HostData.Groups == nil {
		return RefreshGroups(HostData)
	}
	return HostData.Groups, nil
}
