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

package v2

import (
	oktetoLog "github.com/okteto/okteto/pkg/log"
	"github.com/okteto/okteto/pkg/okteto"
	"github.com/spf13/afero"
)

type configRepositoryInterface interface {
	IsCleanContext(string) (bool, error)
}

type configRegistryInterface interface {
	HasGlobalPushAccess() (bool, error)
}

type oktetoBuilderConfig struct {
	hasGlobalAccess bool
	repository      configRepositoryInterface
	fs              afero.Fs
	isOkteto        bool
}

func getConfig(registry configRegistryInterface, gitRepo configRepositoryInterface) oktetoBuilderConfig {
	hasAccess, err := registry.HasGlobalPushAccess()
	if err != nil {
		oktetoLog.Infof("error trying to access globalPushAccess: %w", err)
	}

	return oktetoBuilderConfig{
		repository:      gitRepo,
		hasGlobalAccess: hasAccess,
		fs:              afero.NewOsFs(),
		isOkteto:        okteto.Context().IsOkteto,
	}
}

// IsOkteto checks if the context is an okteto managed context
func (oc oktetoBuilderConfig) IsOkteto() bool {
	return oc.isOkteto
}

// HasGlobalAccess checks if the user has access to global registry
func (oc oktetoBuilderConfig) HasGlobalAccess() bool {
	return oc.hasGlobalAccess
}
