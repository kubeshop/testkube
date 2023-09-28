package slaves

import (
	"strings"

	"golang.org/x/exp/slices"
)

type SlaveMeta map[string]string

func (m *SlaveMeta) Names() []string {
	var names []string
	for k := range *m {
		names = append(names, k)
	}
	return names
}

func (m *SlaveMeta) IPs() []string {
	var ips []string
	for _, v := range *m {
		ips = append(ips, v)
	}
	return ips
}

func (m *SlaveMeta) ToIPString() string {
	ips := m.IPs()
	slices.Sort(ips)
	return strings.Join(m.IPs(), ",")
}
