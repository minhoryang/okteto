//go:build integration
// +build integration

// Copyright 2023 The Okteto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package up

import (
	"context"
	"github.com/okteto/okteto/integration"
	"github.com/okteto/okteto/integration/commands"
	"github.com/okteto/okteto/pkg/constants"
	"github.com/okteto/okteto/pkg/k8s/kubeconfig"
	"github.com/okteto/okteto/pkg/okteto"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

const (
	hybridManifest = `
deploy:
  compose: docker-compose.yml
dev:
  svc:
    context: svc
    namespace: user
    mode: hybrid
    command: ./checker.sh
    reverse:
    - 8080:8080
    args:
    - value-of-arg1
    - $VALUE_OF_ARG2
    envFiles:
    - .env
    environment:
    - TEST_ENV_VAR1=test-value1
    metadata:
      annotations:
        e2e/test-1: annotation-1
      labels:
        custom.label/e2e: "true"
`
	hybridCompose = `services:
 svc:
  build: .
  environment:
  - ENV_IN_POD=value_from_pod`

	svcDockerfile = `FROM busybox
ENV ENV_IN_IMAGE value_from_image`
	envFile      = `VALUE_OF_ARG2=value-of-arg2`
	localProcess = `
#!/bin/bash

if [ "$#" -ne 2 ]; then
  echo "Total arguments count should be 2, got $# instead"
  exit 1
fi

if [ "$1" != "value-of-arg1" ]; then
  echo "First argument should be 'value-of-arg1', got '$1' instead"
  exit 1
fi

if [ "$2" != "value-of-arg2" ]; then
  echo "Second argument should be 'value-of-arg2', got '$2' instead"
  exit 1
fi

if [ "$TEST_ENV_VAR1" != "test-value1" ]; then
  echo "TEST_ENV_VAR1 should be 'test-value1', got '$TEST_ENV_VAR1' instead"
  exit 1
fi

for x in ENV_IN_POD,value_from_pod ENV_IN_IMAGE,value_from_image ; do
  IFS=, read name value <<< "$x"
  if [ "${!name}" != "$value" ]; then
    echo "env '$name' not found. Expected value '$value'"
    exit 1
  fi
done

echo "!Successful envs check!"
exit 0`
)

// TestUpUsingHybridMode test hybrid mode checking:
// - envs from config map, image and dev container are available
// - we cannot run a micro application locally and check that it is exposed
// using the reverse in the remote because we do not know the dependencies
// locally. The testing of the reverse is addressed in other tests
func TestUpUsingHybridMode(t *testing.T) {
	t.Parallel()
	// Prepare environment
	dir := t.TempDir()
	oktetoPath, err := integration.GetOktetoPath()
	require.NoError(t, err)

	testNamespace := integration.GetTestNamespace("TestHybridMode", user)
	namespaceOpts := &commands.NamespaceOptions{
		Namespace:  testNamespace,
		OktetoHome: dir,
		Token:      token,
	}
	require.NoError(t, commands.RunOktetoCreateNamespace(oktetoPath, namespaceOpts))
	defer commands.RunOktetoDeleteNamespace(oktetoPath, namespaceOpts)
	require.NoError(t, commands.RunOktetoKubeconfig(oktetoPath, dir))
	c, _, err := okteto.NewK8sClientProvider().Provide(kubeconfig.Get([]string{filepath.Join(dir, ".kube", "config")}))
	require.NoError(t, err)

	require.NoError(t, writeFile(filepath.Join(dir, "okteto.yml"), hybridManifest))
	require.NoError(t, writeFile(filepath.Join(dir, "docker-compose.yml"), hybridCompose))
	require.NoError(t, writeFile(filepath.Join(dir, ".stignore"), stignoreContent))
	require.NoError(t, writeFile(filepath.Join(dir, "Dockerfile"), svcDockerfile))
	require.NoError(t, writeFile(filepath.Join(dir, "checker.sh"), localProcess))
	require.NoError(t, writeFile(filepath.Join(dir, ".env"), envFile))
	err = os.Chmod(filepath.Join(dir, "checker.sh"), 0755)
	if err != nil {
		require.NoError(t, err)
	}

	up1Options := &commands.UpOptions{
		Name:       "svc",
		Namespace:  testNamespace,
		Workdir:    dir,
		OktetoHome: dir,
		Token:      token,
		Service:    "svc",
		Deploy:     true,
	}
	output, err := commands.RunOktetoUpAndWaitWithOutput(oktetoPath, up1Options)
	require.NoError(t, err)

	// Test warnings for unsupported fields
	require.Contains(t, output.String(), "In hybrid mode, the field(s) 'context, namespace' specified in your manifest are ignored")

	// Get deployment and check for annotations and labels
	deploy, err := integration.GetDeployment(context.Background(), testNamespace, "svc", c)
	require.NoError(t, err)
	require.Equal(t, constants.OktetoHybridModeFieldValue, deploy.Annotations[constants.OktetoDevModeAnnotation])

	// TODO: understand why this Label is not getting set
	//require.Equal(t, "true", deploy.Annotations["custom.label/e2e"])

	// Test okteto down command
	down1Opts := &commands.DownOptions{
		Namespace: testNamespace,
		Workdir:   dir,
		Service:   "svc",
		Token:     token,
	}
	require.NoError(t, commands.RunOktetoDown(oktetoPath, down1Opts))
}
