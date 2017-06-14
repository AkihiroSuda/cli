// demonstration of using REST API client with cli config (push/pull with credential)

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/registry"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	dockerregistry "github.com/docker/docker/registry"
)

func pullImage(s string) error {
	distributionRef, err := reference.ParseNormalizedNamed(s)
	if err != nil {
		return err
	}
	repoInfo, err := dockerregistry.ParseRepositoryInfo(distributionRef)
	if err != nil {
		return err
	}
	c, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	cfg := config.LoadDefaultConfigFile(os.Stderr)
	authConfig, warns, err := registry.ResolveAuthConfig(context.TODO(), c, cfg, repoInfo.Index)
	if err != nil {
		return err
	}
	for _, w := range warns {
		logrus.Warn(w)
	}
	encodedAuth, err := registry.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}
	responseBody, err := c.ImagePull(context.TODO(), reference.FamiliarString(distributionRef), types.ImagePullOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}
	defer responseBody.Close()
	return jsonmessage.DisplayJSONMessagesStream(responseBody, os.Stdout, os.Stdout.Fd(), true, nil)
}

func main() {
	if len(os.Args) != 3 {
		logrus.Fatalf("Usage: %s pull IMAGE", os.Args[0])
	}
	op, img := os.Args[1], os.Args[2]
	var err error
	switch op {
	case "pull":
		err = pullImage(img)
	default:
		err = fmt.Errorf("unknown op: %s", op)

	}
	if err != nil {
		logrus.Fatal(err)
	}
}
