package linuxhost_client

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

// SSHClientConfiguration wraps the SSH connection logic.
type SSHClientConfiguration struct {
	Host   string
	Port   int64
	Config *ssh.ClientConfig
}
type SSHClientContext struct {
	Configuration *SSHClientConfiguration
	Client        *ssh.Client
}

func (c SSHClientConfiguration) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// NewSSHClient creates a new SSHClient with the given parameters.
func NewSSHClient(host string, port int64, username, password, privateKey string) (*SSHClientContext, error) {
	sshConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Replace this for production use
	}

	// Add password authentication if provided
	if password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(password))
	}

	// Add private key authentication if provided
	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	Configruation := &SSHClientConfiguration{
		Host:   host,
		Port:   port,
		Config: sshConfig,
	}
	address := Configruation.Address()
	client, err := ssh.Dial("tcp", address, Configruation.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH host %s: %w", address, err)
	}
	ctx := &SSHClientContext{
		Configuration: Configruation,
		Client:        client,
	}
	return ctx, nil
}

// ExecuteCommand runs a command on the remote host and returns the output.
func (c *SSHClientContext) ExecuteCommand(cmd string) (string, error) {
	session, err := c.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	var result string
	if output == nil {
		result = ""
	} else {
		result = string(output)
	}

	if err != nil {
		fmt.Println("Got output from command:\n", result)
		return result, fmt.Errorf("CMD FAILED: \"%w\"\ncmd: \"%v\"\nSTDERR:\n%v", err, cmd, result)
	}
	return result, nil
}

// ValuesToStrings converts a slice of attr.Value to a slice of strings.
func ValuesToStrings(values []attr.Value) ([]string, error) {
	result := make([]string, len(values))
	for i, v := range values {
		stringValue, ok := v.(types.String)
		if !ok {
			return nil, fmt.Errorf("value at index %d is not a string: %T", i, v)
		}
		result[i] = stringValue.ValueString()
	}
	return result, nil
}
