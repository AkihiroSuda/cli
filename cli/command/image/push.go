package image

import (
	"fmt"
	"golang.org/x/net/context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/registry"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/jsonmessage"
	dockerregistry "github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

// NewPushCommand creates a new `docker push` command
func NewPushCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [OPTIONS] NAME[:TAG]",
		Short: "Push an image or a repository to a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(dockerCli, args[0])
		},
	}

	flags := cmd.Flags()

	command.AddTrustSigningFlags(flags)

	return cmd
}

func runPush(dockerCli command.Cli, remote string) error {
	ref, err := reference.ParseNormalizedNamed(remote)
	if err != nil {
		return err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := dockerregistry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve the Auth config relevant for this server
	authConfig, warns, err := registry.ResolveAuthConfig(ctx, dockerCli.Client(), dockerCli.ConfigFile(), repoInfo.Index)
	for _, w := range warns {
		fmt.Fprintf(dockerCli.Err(), "Warning: %v\n", w)
	}
	if err != nil {
		return err
	}
	requestPrivilege := command.RegistryAuthenticationPrivilegedFunc(dockerCli, repoInfo.Index, "push")

	if command.IsTrusted() {
		return trustedPush(ctx, dockerCli, repoInfo, ref, authConfig, requestPrivilege)
	}

	responseBody, err := imagePushPrivileged(ctx, dockerCli, authConfig, ref, requestPrivilege)
	if err != nil {
		return err
	}

	defer responseBody.Close()
	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), nil)
}
