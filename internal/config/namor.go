package config

type NamorConfig struct {
	Port     int                      `yaml:"port"`
	Host     string                   `yaml:"host"`
	Timeout  int                      `yaml:"timeout"`
	Webhook  WebhookConfig            `yaml:"webhook"`
	Services map[string]ServiceConfig `yaml:"services"`
	Runtime  string                   `yaml:"runtime"`
}

type WebhookConfig struct {
	Secret string
	Url    string
	//Payload string
}

type ServiceConfig struct {
	RequiresAuth bool     `yaml:"requires_auth"`
	Registry     string   `yaml:"registry"`
	Image        string   `yaml:"image"`
	User         string   `yaml:"user"`
	Tag          string   `yaml:"tag"`
	Ports        []string `yaml:"ports"`
	Environment  []string `yaml:"environment"`
	Volumes      []string `yaml:"volumes"`
	Command      string   `yaml:"command"`
	Args         []string `yaml:"args"`
	// Add more fields as necessary
}
