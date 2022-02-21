package config

type Data struct {
	AnalyticsEnabled bool `json:"analyticsEnabled"`
}

func (c *Data) EnableAnalytics() {
	c.AnalyticsEnabled = true

}

func (c *Data) DisableAnalytics() {
	c.AnalyticsEnabled = false
}
