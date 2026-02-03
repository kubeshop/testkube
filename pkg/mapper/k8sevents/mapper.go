package k8sevents

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// TestkubeEventPrefix is prefix for testkube event
const TestkubeEventPrefix = "testkube-event-"

// MapAPIToCRD maps OpenAPI Event spec To CRD Event
func MapAPIToCRD(event testkube.Event, namespace string, eventTime time.Time) corev1.Event {
	var action, reason, message string
	var labels map[string]string

	objectReference := corev1.ObjectReference{
		Kind:      "testkube",
		Name:      "testkube",
		Namespace: namespace,
	}

	if event.TestWorkflowExecution != nil {
		message = fmt.Sprintf("executionId=%s", event.TestWorkflowExecution.Id)
		objectReference.APIVersion = testworkflowsv1.Group + "/" + testworkflowsv1.Version
		objectReference.Kind = testworkflowsv1.Resource
		if event.TestWorkflowExecution.Workflow != nil {
			labels = event.TestWorkflowExecution.Workflow.Labels
			objectReference.Name = event.TestWorkflowExecution.Workflow.Name
		}
	}

	if event.Type_ != nil {
		reason = string(*event.Type_)
		switch *event.Type_ {
		case *testkube.EventStartTestWorkflow:
			action = "started"
		case *testkube.EventEndTestWorkflowSuccess:
			action = "succeed"
		case *testkube.EventEndTestWorkflowFailed:
			action = "failed"
		case *testkube.EventEndTestWorkflowAborted:
			action = "aborted"
		case *testkube.EventQueueTestWorkflow:
			action = "queued"
		case *testkube.EventCreated:
			action = "created"
		case *testkube.EventUpdated:
			action = "updated"
		case *testkube.EventDeleted:
			action = "deleted"
		}
	}

	return corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", TestkubeEventPrefix, event.Id),
			Namespace: namespace,
			Labels:    sanitizeLabels(labels),
		},
		InvolvedObject:      objectReference,
		Action:              action,
		Reason:              reason,
		Message:             message,
		EventTime:           metav1.NewMicroTime(eventTime),
		FirstTimestamp:      metav1.NewTime(eventTime),
		LastTimestamp:       metav1.NewTime(eventTime),
		Type:                "Normal",
		ReportingController: "testkkube.io/services",
		ReportingInstance:   "testkkube.io/services/testkube-api-server",
	}
}

// sanitizeLabels sanitizes label keys and values to conform to Kubernetes RFC 1123 naming rules.
// Invalid characters are replaced with hyphens, and labels that cannot be sanitized are dropped.
func sanitizeLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}

	sanitized := make(map[string]string)
	for key, value := range labels {
		// Sanitize the label key
		sanitizedKey := sanitizeLabelKey(key)
		if sanitizedKey == "" {
			// Drop the label if the key cannot be sanitized
			continue
		}

		// Sanitize the label value
		sanitizedValue := sanitizeLabelValue(value)
		if sanitizedValue == "" {
			// Drop the label if the value cannot be sanitized
			continue
		}

		sanitized[sanitizedKey] = sanitizedValue
	}

	return sanitized
}

// sanitizeLabelKey sanitizes a label key to conform to Kubernetes qualified name rules.
// The key must be a valid DNS subdomain prefix (optional) and name, separated by a slash.
func sanitizeLabelKey(key string) string {
	if key == "" {
		return ""
	}

	// Split into prefix and name if there's a slash
	var prefix, name string
	if idx := strings.Index(key, "/"); idx >= 0 {
		prefix = key[:idx]
		name = key[idx+1:]
	} else {
		name = key
	}

	// Sanitize prefix if it exists (must be a DNS subdomain)
	if prefix != "" {
		prefix = sanitizeDNSSubdomain(prefix)
		if prefix == "" {
			// If prefix cannot be sanitized, drop it and use just the name part
			prefix = ""
		}
	}

	// Sanitize name (must be a DNS label)
	name = sanitizeDNSLabel(name)
	if name == "" {
		return ""
	}

	// Reconstruct the key
	if prefix != "" {
		key = prefix + "/" + name
	} else {
		key = name
	}

	// Final validation
	errs := validation.IsQualifiedName(key)
	if len(errs) > 0 {
		return ""
	}

	return key
}

// sanitizeLabelValue sanitizes a label value to conform to Kubernetes label value rules.
func sanitizeLabelValue(value string) string {
	if value == "" {
		return value
	}

	// Replace invalid characters with hyphens
	sanitized := strings.Map(func(r rune) rune {
		if isAllowedLabelRune(r) {
			return r
		}
		return '-'
	}, value)

	// Truncate if too long
	if len(sanitized) > validation.LabelValueMaxLength {
		sanitized = sanitized[:validation.LabelValueMaxLength]
	}

	// Trim non-alphanumeric characters from start and end
	sanitized = strings.TrimLeftFunc(sanitized, func(r rune) bool { return !isAlphaNumeric(r) })
	sanitized = strings.TrimRightFunc(sanitized, func(r rune) bool { return !isAlphaNumeric(r) })

	// Final validation
	errs := validation.IsValidLabelValue(sanitized)
	if len(errs) > 0 {
		return ""
	}

	return sanitized
}

// sanitizeDNSSubdomain sanitizes a string to be a valid DNS subdomain.
func sanitizeDNSSubdomain(s string) string {
	if s == "" {
		return ""
	}

	// Replace invalid characters with hyphens
	sanitized := strings.Map(func(r rune) rune {
		if isAlphaNumeric(r) || r == '-' || r == '.' {
			return r
		}
		return '-'
	}, s)

	// Truncate if too long
	if len(sanitized) > validation.DNS1123SubdomainMaxLength {
		sanitized = sanitized[:validation.DNS1123SubdomainMaxLength]
	}

	// Trim non-alphanumeric characters from start and end
	sanitized = strings.TrimLeftFunc(sanitized, func(r rune) bool { return !isAlphaNumeric(r) })
	sanitized = strings.TrimRightFunc(sanitized, func(r rune) bool { return !isAlphaNumeric(r) })

	// Final validation
	errs := validation.IsDNS1123Subdomain(sanitized)
	if len(errs) > 0 {
		return ""
	}

	return sanitized
}

// sanitizeDNSLabel sanitizes a string to be a valid DNS label.
func sanitizeDNSLabel(s string) string {
	if s == "" {
		return ""
	}

	// Replace invalid characters with hyphens
	sanitized := strings.Map(func(r rune) rune {
		if isAlphaNumeric(r) || r == '-' {
			return r
		}
		return '-'
	}, s)

	// Truncate if too long
	if len(sanitized) > validation.DNS1123LabelMaxLength {
		sanitized = sanitized[:validation.DNS1123LabelMaxLength]
	}

	// Trim non-alphanumeric characters from start and end
	sanitized = strings.TrimLeftFunc(sanitized, func(r rune) bool { return !isAlphaNumeric(r) })
	sanitized = strings.TrimRightFunc(sanitized, func(r rune) bool { return !isAlphaNumeric(r) })

	// Final validation
	errs := validation.IsDNS1123Label(sanitized)
	if len(errs) > 0 {
		return ""
	}

	return sanitized
}

// isAllowedLabelRune returns true if the rune is allowed in a label value.
func isAllowedLabelRune(r rune) bool {
	return isAlphaNumeric(r) || r == '-' || r == '_' || r == '.'
}

// isAlphaNumeric returns true if the rune is alphanumeric.
func isAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
