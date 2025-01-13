package controlplaneclient

import (
	"google.golang.org/grpc/metadata"
)

type MD metadata.MD

func (c *client) metadata() MD {
	return MD{}.
		SetAgentID(c.agentID).
		SetAgentToken(c.agentToken).
		SetExecutionID(c.opts.ExecutionID).
		SetOrganizationID(c.proContext.OrgID).
		SetEnvironmentID(c.proContext.EnvID)
}

func (m MD) SetAgentID(agentID string) MD {
	if m == nil {
		m = make(MD)
	}
	if agentID == "" {
		delete(m, "agent-id")
	} else {
		m["agent-id"] = []string{agentID}
	}
	return m
}

func (m MD) SetAgentToken(agentToken string) MD {
	if m == nil {
		m = make(MD)
	}
	if agentToken == "" {
		delete(m, "api-key")
	} else {
		m["api-key"] = []string{agentToken}
	}
	return m
}

func (m MD) SetOrganizationID(organizationID string) MD {
	if m == nil {
		m = make(MD)
	}
	if organizationID == "" {
		delete(m, "organization-id")
	} else {
		m["organization-id"] = []string{organizationID}
	}
	return m
}

func (m MD) SetEnvironmentID(environmentID string) MD {
	if m == nil {
		m = make(MD)
	}
	if environmentID == "" {
		delete(m, "environment-id")
	} else {
		m["environment-id"] = []string{environmentID}
	}
	return m
}

func (m MD) SetExecutionID(executionID string) MD {
	if m == nil {
		m = make(MD)
	}
	if executionID == "" {
		delete(m, "execution-id")
	} else {
		m["execution-id"] = []string{executionID}
	}
	return m
}

func (m MD) GRPC() metadata.MD {
	return metadata.MD(m)
}
