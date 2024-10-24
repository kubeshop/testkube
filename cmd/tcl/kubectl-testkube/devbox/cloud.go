// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devbox

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	"github.com/kubeshop/testkube/pkg/cloud/client"
)

type cloudObj struct {
	cfg       config.CloudContext
	envClient *client.EnvironmentsClient
	list      []client.Environment
}

func NewCloud(cfg config.CloudContext) (*cloudObj, error) {
	if cfg.ApiKey == "" || cfg.OrganizationId == "" || cfg.OrganizationName == "" {
		return nil, errors.New("login to the organization first")
	}
	if strings.HasPrefix(cfg.AgentUri, "https://") {
		cfg.AgentUri = strings.TrimPrefix(cfg.AgentUri, "https://")
		if !regexp.MustCompile(`:\d+$`).MatchString(cfg.AgentUri) {
			cfg.AgentUri += ":443"
		}
	} else if strings.HasPrefix(cfg.AgentUri, "http://") {
		cfg.AgentUri = strings.TrimPrefix(cfg.AgentUri, "http://")
		if !regexp.MustCompile(`:\d+$`).MatchString(cfg.AgentUri) {
			cfg.AgentUri += ":80"
		}
	}
	// TODO: FIX THAT
	if strings.HasPrefix(cfg.AgentUri, "api.") {
		cfg.AgentUri = "agent." + strings.TrimPrefix(cfg.AgentUri, "api.")
	}
	envClient := client.NewEnvironmentsClient(cfg.ApiUri, cfg.ApiKey, cfg.OrganizationId)
	obj := &cloudObj{
		cfg:       cfg,
		envClient: envClient,
	}

	err := obj.UpdateList()
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *cloudObj) List() []client.Environment {
	return c.list
}

func (c *cloudObj) ListObsolete() []client.Environment {
	obsolete := make([]client.Environment, 0)
	for _, env := range c.list {
		if !env.Connected {
			obsolete = append(obsolete, env)
		}
	}
	return obsolete
}

func (c *cloudObj) UpdateList() error {
	list, err := c.envClient.List()
	if err != nil {
		return err
	}
	result := make([]client.Environment, 0)
	for i := range list {
		if strings.HasPrefix(list[i].Name, "devbox-") {
			result = append(result, list[i])
		}
	}
	c.list = result
	return nil
}

func (c *cloudObj) AgentURI() string {
	return c.cfg.AgentUri
}

func (c *cloudObj) AgentInsecure() bool {
	return strings.HasPrefix(c.cfg.ApiUri, "http://")
}

func (c *cloudObj) ApiURI() string {
	return c.cfg.ApiUri
}

func (c *cloudObj) ApiKey() string {
	return c.cfg.ApiKey
}

func (c *cloudObj) ApiInsecure() bool {
	return strings.HasPrefix(c.cfg.ApiUri, "http://")
}

func (c *cloudObj) DashboardUrl(id, path string) string {
	return strings.TrimSuffix(fmt.Sprintf("%s/organization/%s/environment/%s/", c.cfg.UiUri, c.cfg.OrganizationId, id)+strings.TrimPrefix(path, "/"), "/")
}

func (c *cloudObj) CreateEnvironment(name string) (*client.Environment, error) {
	env, err := c.envClient.Create(client.Environment{
		Name:           name,
		Owner:          c.cfg.OrganizationId,
		OrganizationId: c.cfg.OrganizationId,
	})
	if err != nil {
		return nil, err
	}
	c.list = append(c.list, env)
	return &env, nil
}

func (c *cloudObj) DeleteEnvironment(id string) error {
	return c.envClient.Delete(id)
}

func (c *cloudObj) Debug() {
	PrintHeader("Control Plane")
	PrintItem("Organization", c.cfg.OrganizationName, c.cfg.OrganizationId)
	PrintItem("API URL", c.cfg.ApiUri, "")
	PrintItem("UI URL", c.cfg.UiUri, "")
	PrintItem("Agent Server", c.cfg.AgentUri, "")
}
