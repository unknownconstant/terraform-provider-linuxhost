package linuxhost_client

type SSHCommandContext struct {
	client *SSHClientContext
	Output string
	Error  error
}

func NewSSHCommandContext(client *SSHClientContext) SSHCommandContext {
	return SSHCommandContext{client: client}
}

func (c SSHCommandContext) Exec(cmd string) SSHCommandContext {
	output, err := c.client.ExecuteCommand(cmd)
	return SSHCommandContext{
		client: c.client,
		Output: output,
		Error:  err,
	}
}

func (c SSHCommandContext) Then(fn func(SSHCommandContext) SSHCommandContext) SSHCommandContext {
	if c.Error != nil {
		return c
	}
	return fn(c)
}
