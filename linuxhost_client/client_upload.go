package linuxhost_client

import (
	"fmt"
	"os"
)

type FileTransferParams struct {
	SourcePath      string
	DestinationPath string
	Permissions     *int
	Uid             *int
	Gid             *int
}

func SetTextFile(c *SSHCommandContext, params *FileTransferParams) *SSHCommandContext {
	/// Open the local file
	localFile, err := os.ReadFile(params.SourcePath)
	if err != nil {
		c.Error = fmt.Errorf("failed to open local file: %v", err)
		return c
	}
	fileContent := string(localFile)
	r := c.
		Exec(fmt.Sprintf("sudo touch %s", params.DestinationPath)).
		Then(func(ctx SSHCommandContext) SSHCommandContext {
			if params.Uid == nil && params.Gid == nil {
				return SSHCommandContext{}
			}
			User := ""
			if params.Uid != nil {
				User = fmt.Sprintf("%d", *params.Uid)
			}
			Group := ""
			if params.Gid != nil {
				Group = fmt.Sprintf(":%d", *params.Gid)
			}
			cmd := fmt.Sprintf("sudo chown %s%s %s", User, Group, params.DestinationPath)
			return ctx.Exec(cmd)
		}).
		Exec(fmt.Sprintf("sudo chmod %o %s", *params.Permissions, params.DestinationPath)).
		Exec(fmt.Sprintf("cat << EOF | sudo tee %s\n%s\nEOF", params.DestinationPath, fileContent))
	return &r
}
