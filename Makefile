.PHONY: build run clean deps test

# Build the application
build:
	go build -o bin/elephant-tf-ci ./cmd

# Run the application
run: build
	./bin/elephant-tf-ci

# Install for system-wide use
install: build
	sudo cp bin/elephant-tf-ci /usr/local/bin/

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/elephant-tf-ci-linux-amd64 ./cmd
	GOOS=darwin GOARCH=amd64 go build -o bin/elephant-tf-ci-darwin-amd64 ./cmd
	GOOS=windows GOARCH=amd64 go build -o bin/elephant-tf-ci-windows-amd64.exe ./cmd

# Create GitHub release
# Create GitHub release
release: build-all
	gh release create v1.0.0 \
		bin/elephant-tf-ci-linux-amd64 \
		bin/elephant-tf-ci-darwin-amd64 \
		bin/elephant-tf-ci-windows-amd64.exe \
		--title "🐘 Elephant TF CI v1.0.0" \
		--notes "## 🐘 Elephant TF CI - Terraform Pipeline Generator\n\n**Ubuntu-powered CI/CD Pipeline Setup Tool**\n\n### ✨ Features\n- 🎨 Interactive TUI with Bubble Tea\n- 🔐 OIDC keyless AWS authentication\n- 📁 Automatic GitHub repository creation\n- 🔧 Multi-environment support (dev/qa/prod)\n- 🛡️ Built-in security scanning (Checkov, TFLint, TFSec)\n- 🚀 PR-based approval workflow\n- 🌍 Ubuntu philosophy - \"I am because we are\"\n\n### 📦 Installation\n\n**One-liner install:**\n\`\`\`bash\ncurl -sSL https://raw.githubusercontent.com/yourusername/elephant-tf-ci/main/install.sh | bash\n\`\`\`\n\n**Manual download:**\n- **Linux:** \`elephant-tf-ci-linux-amd64\`\n- **macOS:** \`elephant-tf-ci-darwin-amd64\`\n- **Windows:** \`elephant-tf-ci-windows-amd64.exe\`\n\n### 🚀 Usage\n\`\`\`bash\n./elephant-tf-ci\n\`\`\`\n\n**Sawubona!** 🐘 Happy building with Elephant TF CI!"

