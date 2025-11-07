# 🐘 Indlovu Pipeline Generator

**Ubuntu-powered CI/CD Pipeline Setup Tool**

Indlovu (Elephant in Zulu) is a Go TUI application that helps you create GitHub Actions CI/CD pipelines with AWS OIDC authentication. Built with Ubuntu philosophy - "I am because we are."

## ✨ Features

- 🎨 **Interactive TUI** - Beautiful terminal interface using Bubble Tea
- 🔐 **OIDC Authentication** - Secure keyless AWS authentication
- 📁 **GitHub Integration** - Automatic repository and workflow creation
- 🔧 **Multi-Environment** - Support for dev/qa/prod environments
- 🛡️ **Security Scanning** - Built-in Checkov, TFLint, and TFSec
- 🌍 **Ubuntu Spirit** - Embracing African tech excellence

## 🚀 Quick Start

### Prerequisites

1. **GitHub Personal Access Token**
   - Go to: https://github.com/settings/tokens
   - Create token with `repo` and `workflow` permissions

2. **AWS OIDC Setup**
   - Create OIDC Identity Provider in AWS IAM
   - Create IAM roles for each environment
   - Configure trust policies

### Installation

```bash
# Clone and build
git clone <repository>
cd indlovu-pipeline
go mod tidy
go build -o indlovu-pipeline ./cmd

# Run the application
./indlovu
```

### Usage

1. **GitHub Configuration**
   - Enter your GitHub token
   - Specify username and repository name

2. **AWS Configuration**
   - Provide AWS Account ID and region
   - Enter S3 bucket for Terraform state
   - Configure IAM role ARNs

3. **Repository Settings**
   - Set repository description
   - Choose public/private visibility

4. **Review & Deploy**
   - Confirm configuration
   - Let Indlovu create everything!

## 🏗️ What It Creates

- ✅ GitHub repository
- ✅ Complete CI/CD workflow file with PR-based approval
- ✅ GitHub secrets configuration
- ✅ OIDC trust policy documentation
- ✅ Multi-environment support
- ✅ Security scanning integration
- ✅ Automated plan on PRs, apply on merge

## 🔧 Configuration

The tool collects:

### GitHub Settings
- Personal access token
- Username and repository name
- Repository visibility

### AWS Settings
- Account ID and region
- Terraform state S3 bucket
- IAM role ARNs for each environment

### Pipeline Features
- **PR-based approval workflow** - Plan runs on PRs, apply after merge
- **Branch-based deployments** - Automatic environment detection
- **Security scanning** - Checkov, TFLint, TFSec integration
- **OIDC authentication** - Keyless AWS access
- **Multi-environment support** - dev/qa/prod environments
- **Feature branch protection** - No apply on feature/ branches

## 🛡️ Security & Workflow

### Security Features
- Uses GitHub OIDC for keyless authentication
- No long-term AWS credentials stored
- Branch-specific IAM role restrictions
- Encrypted GitHub secrets
- Security scanning in pipeline

### Approval Workflow
1. **Create PR** → Terraform plan runs automatically
2. **Review PR** → Code and infrastructure changes reviewed together
3. **Approve & Merge** → Single approval gate for both code and infra
4. **Auto Deploy** → Apply runs immediately after merge

### Branch Behavior
- **PRs** → Plan only (shows proposed changes)
- **main/master** → Plan + Apply to prod environment
- **dev/qa** → Plan + Apply to respective environments
- **feature/** → Plan only (no apply)

## 🌍 Ubuntu Philosophy

Built with Ubuntu spirit - "I am because we are." This tool empowers African developers and the global community to build secure, scalable infrastructure with modern DevOps practices.

## 📚 Documentation

For detailed setup instructions, see the generated README in your new repository.

## 🤝 Contributing

Contributions welcome! Please read our contributing guidelines and code of conduct.

## 📄 License

MIT License - see LICENSE file for details.

---

**Sawubona!** 🐘 Happy building with Indlovu Pipeline!