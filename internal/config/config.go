package config

type Config struct {
	GitHub GitHubConfig `yaml:"github"`
	AWS    AWSConfig    `yaml:"aws"`
	Repo   RepoConfig   `yaml:"repo"`
}

type GitHubConfig struct {
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	RepoName string `yaml:"repo_name"`
	Branch   string `yaml:"branch"`
}

type AWSConfig struct {
	Region               string `yaml:"region"`
	AccountID            string `yaml:"account_id"`
	StateBucket          string `yaml:"state_bucket"`
	PipelineRoleARN      string `yaml:"pipeline_role_arn"`
	FailOnSecurityIssues bool   `yaml:"fail_on_security_issues"`
	HasExistingBackend   bool   `yaml:"has_existing_backend"`
}

type RepoConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Private     bool   `yaml:"private"`
}

func NewConfig() *Config {
	return &Config{
		GitHub: GitHubConfig{},
		AWS: AWSConfig{
			Region: "us-east-1",
		},
		Repo: RepoConfig{
			Private: true,
		},
	}
}