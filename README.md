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

1. **GitHub OAuth App**
   - Create OAuth app at: https://github.com/settings/applications/new
   - Set Authorization callback URL: `http://localhost:8080/callback`
   - Note Client ID and Client Secret for `.env` file

2. **AWS OIDC Setup**

   **Step 1: Create OIDC Identity Provider**
   ```bash
   # AWS Console: IAM → Identity providers → Add provider
   # OR AWS CLI:
   aws iam create-open-id-connect-provider \
     --url https://token.actions.githubusercontent.com \
     --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1 \
     --client-id-list sts.amazonaws.com
   ```

   **Step 2: Create IAM Role with Web Identity**
   ```bash
   # AWS Console: IAM → Roles → Create role → Web identity
   # Identity provider: token.actions.githubusercontent.com
   # Audience: sts.amazonaws.com
   ```

   **Step 3: Configure Trust Policy**
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Principal": {
           "Federated": "arn:aws:iam::YOUR-ACCOUNT-ID:oidc-provider/token.actions.githubusercontent.com"
         },
         "Action": "sts:AssumeRoleWithWebIdentity",
         "Condition": {
           "StringEquals": {
             "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
           },
           "StringLike": {
             "token.actions.githubusercontent.com:sub": "repo:YOUR-ORG/YOUR-REPO:*"
           }
         }
       }
     ]
   }
   ```

   **Step 4: Attach Permissions Policy**
   - Attach policies for Terraform operations (EC2, S3, etc.)
   - Ensure S3 access for Terraform state bucket

### Installation

```bash
# Clone and build
git clone <repository>
cd indlovu-pipeline
go mod tidy
go build -o indlovu-pipeline ./cmd

# Setup environment variables
cp .env.example .env
# Edit .env with your GitHub OAuth credentials

# Run the application
./indlovu-pipeline
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
4. **Auto Deploy** → Apply runs only on PR merge (not direct push)

### Branch Behavior
- **PRs** → Plan only (shows proposed changes)
- **PR merge to main/master** → Plan + Apply to prod environment
- **PR merge to dev/qa** → Plan + Apply to respective environments
- **Direct push to branches** → Plan only (no apply)
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