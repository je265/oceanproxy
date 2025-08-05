package domain

import "fmt"

// Region represents a geographical/logical region configuration
type Region struct {
	Name            string   `yaml:"name" json:"name"`
	Subdomain       string   `yaml:"subdomain" json:"subdomain"`
	DomainSuffix    string   `yaml:"domain_suffix" json:"domain_suffix"`
	OutboundPort    int      `yaml:"outbound_port" json:"outbound_port"`
	Description     string   `yaml:"description" json:"description"`
	PlanTypes       []string `yaml:"plan_types" json:"plan_types"`
	NginxConfigFile string   `yaml:"nginx_config_file" json:"nginx_config_file"`
}

// GetFullDomain returns the complete domain for this region
func (r *Region) GetFullDomain() string {
	return fmt.Sprintf("%s.%s", r.Subdomain, r.DomainSuffix)
}

// GetProxyEndpoint returns the customer-facing proxy endpoint
func (r *Region) GetProxyEndpoint(username, password string) string {
	return fmt.Sprintf("http://%s:%s@%s:%d", username, password, r.GetFullDomain(), r.OutboundPort)
}

// PlanTypeConfig represents configuration for a specific plan type
type PlanTypeConfig struct {
	Name              string    `yaml:"name" json:"name"`
	Provider          string    `yaml:"provider" json:"provider"`
	Region            string    `yaml:"region" json:"region"`
	PlanType          string    `yaml:"plan_type" json:"plan_type"`
	UpstreamPort      int       `yaml:"upstream_port" json:"upstream_port"`
	UpstreamHost      string    `yaml:"upstream_host" json:"upstream_host"`
	LocalPortRange    PortRange `yaml:"local_port_range" json:"local_port_range"`
	OutboundPort      int       `yaml:"outbound_port" json:"outbound_port"`
	NginxUpstreamName string    `yaml:"nginx_upstream_name" json:"nginx_upstream_name"`
}

// PortRange defines a range of ports
type PortRange struct {
	Start int `yaml:"start" json:"start"`
	End   int `yaml:"end" json:"end"`
}

// Contains checks if a port is within this range
func (pr *PortRange) Contains(port int) bool {
	return port >= pr.Start && port <= pr.End
}

// Size returns the number of ports in this range
func (pr *PortRange) Size() int {
	return pr.End - pr.Start + 1
}

// GetPlanTypeKey generates a unique key for plan type identification
func (ptc *PlanTypeConfig) GetPlanTypeKey() string {
	return fmt.Sprintf("%s_%s_%s", ptc.Provider, ptc.Region, ptc.PlanType)
}

// GetUpstreamEndpoint returns the upstream provider endpoint
func (ptc *PlanTypeConfig) GetUpstreamEndpoint() string {
	return fmt.Sprintf("%s:%d", ptc.UpstreamHost, ptc.UpstreamPort)
}
