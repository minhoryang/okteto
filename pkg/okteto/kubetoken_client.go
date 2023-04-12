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

package okteto

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"

	oktetoErrors "github.com/okteto/okteto/pkg/errors"
)

const kubetokenPath = "auth/kubetoken"

type storeRegister struct {
	ContextName string                        `json:"context"`
	Namespace   string                        `json:"namespace"`
	Token       authenticationv1.TokenRequest `json:"token"`
}

type FileByteStore struct {
	FileName string
}

func (s *FileByteStore) Get() ([]byte, error) {
	if _, err := os.Stat(s.FileName); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("error checking if file exists: %w", err)
		}

		if err := os.WriteFile(s.FileName, []byte("[]"), 0600); err != nil {
			return nil, fmt.Errorf("error creating file: %w", err)
		}
	}

	contents, err := os.ReadFile(s.FileName)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return contents, nil
}

func (s *FileByteStore) Set(value []byte) error {
	return os.WriteFile(s.FileName, value, 0600)
}

type stringStore interface {
	Get() ([]byte, error)
	Set([]byte) error
}

type KubeTokenCache struct {
	StringStore stringStore
}

func (c *KubeTokenCache) read() ([]storeRegister, error) {
	contents, err := c.StringStore.Get()
	if err != nil {
		return nil, err
	}

	var store []storeRegister

	if err := json.Unmarshal(contents, &store); err != nil {
		return nil, fmt.Errorf("error decoding") // TODO: we should probably delete the file contents
	}

	return store, nil
}

func (c *KubeTokenCache) Get(contextName, namespace string) (string, error) {
	store, err := c.read()
	if err != nil {
		return "", err
	}

	for _, register := range store {
		if register.ContextName == contextName && register.Namespace == namespace {
			now := time.Now() // TODO: inject this
			if register.Token.Status.ExpirationTimestamp.Time.After(now) {
				tokenString, _ := json.MarshalIndent(register.Token, "", "\t")

				return string(tokenString), nil
			} else {
				// TODO: we could invalidate this cache here
				return "", nil
			}
		}
	}

	return "", nil
}

func (c *KubeTokenCache) setWithErr(contextName, namespace string, token authenticationv1.TokenRequest) error {
	store, err := c.read()
	if err != nil {
		return err
	}

	existed := false
	for i, r := range store {
		if r.ContextName == contextName && r.Namespace == namespace {
			store[i].Token = token
			existed = true
		}
	}
	if !existed {
		store = append(store, storeRegister{
			ContextName: contextName,
			Namespace:   namespace,
			Token:       token,
		})
	}

	newStore, err := json.MarshalIndent(store, "", "\t")
	if err != nil {
		return err
	}

	return c.StringStore.Set(newStore)
}

func (c *KubeTokenCache) Set(contextName, namespace string, token authenticationv1.TokenRequest) {
	if err := c.setWithErr(contextName, namespace, token); err != nil {
		// TODO: log this
	}
}

type cacheSetter interface {
	Set(contextName, namespace string, token authenticationv1.TokenRequest)
}

type KubeTokenClient struct {
	httpClient  *http.Client
	url         string
	contextName string
	namespace   string
	cache       cacheSetter
}

func NewKubeTokenClient(contextName, token, namespace string, cache cacheSetter) (*KubeTokenClient, error) {
	if contextName == "" {
		return nil, oktetoErrors.ErrCtxNotSet
	}

	httpClient, url, err := newOktetoHttpClient(contextName, token, fmt.Sprintf("%s/%s", kubetokenPath, namespace))
	if err != nil {
		return nil, err
	}

	return &KubeTokenClient{
		httpClient:  httpClient,
		url:         url,
		contextName: contextName,
		namespace:   namespace,
		cache:       cache,
	}, nil
}

func (c *KubeTokenClient) GetKubeToken() (string, error) {
	resp, err := c.httpClient.Get(c.url)
	if err != nil {
		return "", fmt.Errorf("failed GET request: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf(oktetoErrors.ErrNotLogged, c.contextName)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET request returned status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read kubetoken response: %w", err)
	}

	token := authenticationv1.TokenRequest{}

	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("failed to unmarshal kubetoken response: %w", err) // TODO check this error
	}

	c.cache.Set(c.contextName, c.namespace, token)

	return string(body), nil
}
