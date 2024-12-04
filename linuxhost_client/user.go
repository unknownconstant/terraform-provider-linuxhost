package linuxhost_client

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UserCommand struct {
	cmd      string
	complete bool
}

func NewUserCommand(createNew bool) UserCommand {
	initialCommand := ""
	if createNew {
		initialCommand = "sudo useradd"
	} else {
		initialCommand = "sudo usermod"
	}
	U := UserCommand{
		cmd: initialCommand,
	}
	return U
}
func (u *UserCommand) wrapArgument(CB func()) *UserCommand {
	if u.complete {
		fmt.Println("ERROR")
		return u
	}
	CB()
	return u
}
func (u *UserCommand) WithUid(v types.Number) *UserCommand {
	return u.wrapArgument(func() {
		f, _ := v.ValueBigFloat().Float64()
		u.cmd = u.cmd + fmt.Sprintf(" -u %d", int(f))
	})
}
func (u *UserCommand) WithGid(v types.Number) *UserCommand {
	return u.wrapArgument(func() {
		f, _ := v.ValueBigFloat().Float64()
		u.cmd = u.cmd + fmt.Sprintf(" -g %d", int(f))
	})
}
func (u *UserCommand) WithPrimaryGroup(PrimaryGroup types.String) *UserCommand {
	return u.wrapArgument(func() {
		u.cmd = u.cmd + fmt.Sprintf(" -g %s", PrimaryGroup.ValueString())
	})
}
func (u *UserCommand) WithHomeDirectory(HomeDirectory types.String) *UserCommand {
	return u.wrapArgument(func() {
		u.cmd = u.cmd + " -d '" + HomeDirectory.ValueString() + "' -m"
	})
}
func (u *UserCommand) WithShell(Shell types.String) *UserCommand {
	return u.wrapArgument(func() {
		u.cmd = u.cmd + " -s '" + Shell.ValueString() + "'"
	})
}
func (u *UserCommand) WithModifiedUsername(Username types.String) *UserCommand {
	return u.wrapArgument(func() {
		u.cmd = u.cmd + " -l " + Username.ValueString()
	})

}
func (u *UserCommand) WithCurrentUsername(Username string) {
	u.cmd = u.cmd + " " + Username
}

func buildUserCommand(user *models.UserModel, targetUser *string) UserCommand {
	userCommand := NewUserCommand(targetUser == nil)

	// cmd := "sudo useradd"

	if !user.UID.IsNull() && !user.UID.IsUnknown() {
		userCommand.WithUid(user.UID)
	}

	if !user.GID.IsNull() && !user.GID.IsUnknown() {
		userCommand.WithGid(user.GID)
	}

	if !user.PrimaryGroup.IsNull() && !user.PrimaryGroup.IsUnknown() {
		userCommand.WithPrimaryGroup(user.PrimaryGroup)
	}

	if !user.HomeDirectory.IsNull() && !user.HomeDirectory.IsUnknown() {
		userCommand.WithHomeDirectory(user.HomeDirectory)
	}
	if !user.Shell.IsNull() && !user.Shell.IsUnknown() {
		userCommand.WithShell(user.Shell)
	}

	if targetUser == nil {
		userCommand.WithCurrentUsername(user.Username.ValueString())
	} else {
		userCommand.WithModifiedUsername(user.Username).WithCurrentUsername(*targetUser)
	}

	return userCommand
}

func SetUser(connectedClient *SSHClientContext, user *models.UserModel, targetUser *string) error {
	fmt.Println("Creating user")
	fmt.Println(user)
	userCommand := buildUserCommand(user, targetUser)
	fmt.Println("cmd: " + userCommand.cmd)

	result, err := connectedClient.ExecuteCommand(userCommand.cmd)

	if err != nil {
		fmt.Println("Error setting user: "+err.Error(), "\nresult: ", result)
		return err
	}
	fmt.Println(result)
	return nil
}
func DeleteUser(ConnectedClient *SSHClientContext, user *models.UserModel) error {
	cmd := fmt.Sprintf("sudo userdel %s", user.Username.ValueString())

	_, err := ConnectedClient.ExecuteCommand(cmd)
	return err
}

func RefreshUsers(HostData *HostData) ([]models.UserModel, error) {
	groups, err := GetGroups(HostData)
	if err != nil {
		return nil, err
	}
	groupById := ListToMap(groups, func(a models.GroupModel) int { return TFNumberToInt(a.GID) })
	cmd := "cat /etc/passwd"
	result, err := CommandRunner(HostData, cmd)
	if err != nil {
		return nil, err
	}
	hostnameString, err := GetHostname(HostData)
	if err != nil {
		return nil, err
	}
	users := []models.UserModel{}
	parseUser := func(line string) {
		passwdRegex := regexp.MustCompile(`([^:]*):([^:]*):([^:]*):([^:]*):([^:]*):([^:]*):([^:]*)`)

		match := passwdRegex.FindStringSubmatch(line)
		if match == nil {
			return
		}

		GID := strToTFNumber(match[4])
		PrimaryGroup := groupById[TFNumberToInt(GID)]

		user := models.UserModel{
			Username:      types.StringValue(match[1]),
			UID:           strToTFNumber(match[3]),
			GID:           GID,
			PrimaryGroup:  PrimaryGroup.Name,
			Groups:        types.SetValueMust(types.StringType, []attr.Value{}),
			Hostname:      types.StringValue(*hostnameString),
			HomeDirectory: types.StringValue(match[6]),
			Shell:         types.StringValue(match[7]),
		}
		userGroups := UserGroups(groups, user)
		user.Groups = userGroups

		users = append(users, user)
	}
	LineMatcher(*result, parseUser)
	return users, nil
}
func strToTFNumber(v string) types.Number {
	num := new(big.Float)
	num.SetString(v)
	return types.NumberValue(num)
}
func TFNumberToInt(v types.Number) int {
	f, _ := v.ValueBigFloat().Float64()
	return int(f)
}
func GetUsers(HostData *HostData) ([]models.UserModel, error) {
	if HostData.Users == nil {
		return RefreshUsers(HostData)
	}
	return HostData.Users, nil
}

func GetHostname(HostData *HostData) (*string, error) {
	if HostData.Hostname == nil {
		cmd := "hostname"
		result, err := CommandRunner(HostData, cmd)
		if err != nil {
			return nil, err
		}
		r := strings.TrimSpace(*result)
		return &r, nil
	}
	return HostData.Hostname, nil
}

func CommandRunner(HostData *HostData, cmd string) (*string, error) {
	result, err := HostData.Client.ExecuteCommand(cmd)
	if err != nil {
		return nil, err
	}
	fmt.Println("got result")
	return &result, nil
}

func LineMatcher(stdout string, processLine func(trimmedLine string)) {
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		processLine(line)
	}
}
