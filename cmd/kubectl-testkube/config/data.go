package config

type Data struct {
	AnalyticsEnabled bool   `json:"analyticsEnabled,omitempty"`
	Namespace        string `json:"namespace,omitempty"`
}

func (c *Data) EnableAnalytics() {
	c.AnalyticsEnabled = true

}

func (c *Data) DisableAnalytics() {
	c.AnalyticsEnabled = false
}

func (c *Data) SetNamespace(ns string) {
	c.Namespace = ns
}
