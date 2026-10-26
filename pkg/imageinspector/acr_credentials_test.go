package imageinspector

import "testing"

type recordingACRHelper struct {
	server string
}

func (h *recordingACRHelper) Get(server string) (string, string, error) {
	h.server = server
	return "user", "token", nil
}

// Azure credentials must never reach a host that only embeds an Azure registry suffix.
func TestACRCredentialsHelper(t *testing.T) {
	for _, tc := range []struct {
		server string
		allow  bool
	}{
		{"registry.azurecr.io", true},
		{"registry.azurecr.cn", true},
		{"registry.azurecr.de", true},
		{"registry.azurecr.us", true},
		{"registry.azurecr.io:443", true},
		{"mcr.microsoft.com", true},
		{"mcr.microsoft.com:443", true},
		{"evil.azurecr.io.attacker.com", false},
		{"evil.azurecr.cn.attacker.com", false},
		{"evil.azurecr.de.attacker.com", false},
		{"evil.azurecr.us.attacker.com", false},
		{"mcr.microsoft.com.attacker.com", false},
		{"registry.azurecr.io@attacker.com", false},
		{"attacker.com@registry.azurecr.io", false},
		{"attacker.com/registry.azurecr.io", false},
		{"registry.azurecr.io/path", false},
		{"registry.azurecr.io?redirect=attacker.com", false},
		{"registry.azurecr.io?", false},
		{"registry.azurecr.io#attacker.com", false},
		{"registry.azurecr.io:invalid", false},
		{"registry.azurecr.io%2eattacker.com", false},
		{"registry.azurecr.io\\attacker.com", false},
		{"https://registry.azurecr.io", false},
		{"docker.io", false},
		{"", false},
	} {
		t.Run(tc.server, func(t *testing.T) {
			delegate := &recordingACRHelper{}
			helper := acrCredentialsHelper{delegate}
			user, token, err := helper.Get(tc.server)
			if tc.allow {
				if err != nil || delegate.server != tc.server || user != "user" || token != "token" {
					t.Fatalf("got (%q, %q, %v), delegated to %q; want credentials for %q", user, token, err, delegate.server, tc.server)
				}
			} else if err == nil || delegate.server != "" || user != "" || token != "" {
				t.Fatalf("got (%q, %q, %v), delegated to %q; want rejection without delegation", user, token, err, delegate.server)
			}
		})
	}
}
