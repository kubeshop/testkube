package controlplaneclient

import (
	"google.golang.org/grpc/metadata"
)

const (
	RuntimeNamespaceMetadataName = "r-namespace"
	AgentIdMetadataName          = "agent-id"
	AgentSecretKeyMetadataName   = "api-key"
	OrganizationIdMetadataName   = "organization-id"
	EnvironmentIdMetadataName    = "environment-id"
	ExecutionIdMetadataName      = "execution-id"
)

type MD metadata.MD

func (c *client) metadata() MD {
	return MD{}.
		SetRuntimeNamespace(c.opts.Runtime.Namespace).
		SetAgentID(c.proContext.Agent.ID).
		SetSecretKey(c.proContext.APIKey).
		SetExecutionID(c.opts.ExecutionID).
		SetOrganizationID(c.proContext.OrgID).
		SetEnvironmentID(c.proContext.EnvID)
}

func (m MD) SetRuntimeNamespace(namespace string) MD {
	if m == nil {
		m = make(MD)
	}
	if namespace == "" {
		delete(m, RuntimeNamespaceMetadataName)
	} else {
		m[RuntimeNamespaceMetadataName] = []string{namespace}
	}
	return m
}

func (m MD) SetAgentID(agentID string) MD {
	if m == nil {
		m = make(MD)
	}
	if agentID == "" {
		delete(m, AgentIdMetadataName)
	} else {
		m[AgentIdMetadataName] = []string{agentID}
	}
	return m
}

func (m MD) SetSecretKey(secretKey string) MD {
	if m == nil {
		m = make(MD)
	}
	if secretKey == "" {
		delete(m, AgentSecretKeyMetadataName)
	} else {
		m[AgentSecretKeyMetadataName] = []string{secretKey}
	}
	return m
}

func (m MD) SetOrganizationID(organizationID string) MD {
	if m == nil {
		m = make(MD)
	}
	if organizationID == "" {
		delete(m, OrganizationIdMetadataName)
	} else {
		m[OrganizationIdMetadataName] = []string{organizationID}
	}
	return m
}

func (m MD) SetEnvironmentID(environmentID string) MD {
	if m == nil {
		m = make(MD)
	}
	if environmentID == "" {
		delete(m, EnvironmentIdMetadataName)
	} else {
		m[EnvironmentIdMetadataName] = []string{environmentID}
	}
	return m
}

func (m MD) SetExecutionID(executionID string) MD {
	if m == nil {
		m = make(MD)
	}
	if executionID == "" {
		delete(m, ExecutionIdMetadataName)
	} else {
		m[ExecutionIdMetadataName] = []string{executionID}
	}
	return m
}

func (m MD) GRPC() metadata.MD {
	return metadata.MD(m)
}
