# GitHub OAuth Setup Instructions

## Recommended: Create GitHub OAuth App

**Use GitHub OAuth Apps (not GitHub Apps)** for the Indlovu Pipeline Generator.

### Why OAuth Apps?
- ✅ Simpler setup
- ✅ Direct repository access
- ✅ No installation required
- ✅ Token starts with `gho_`

### Setup Steps

1. **Go to OAuth Apps:**
   - Visit: https://github.com/settings/applications/new

2. **Configure OAuth App:**
   - **Application name:** `Indlovu Pipeline Generator`
   - **Homepage URL:** `http://localhost:8080`
   - **Authorization callback URL:** `http://localhost:8080/callback`

3. **Register Application:**
   - Click "Register application"
   - Copy the **Client ID**
   - Generate and copy **Client Secret**

### Scopes Requested
The app will request these scopes:
- `repo` - Full repository access
- `workflow` - GitHub Actions workflow access
- `admin:repo_hook` - Repository webhook management
- `read:org` - Organization membership

## Alternative: Use Personal Access Token

If you prefer not to create an OAuth App:

1. When asked "Use GitHub OAuth?", choose **No**
2. Go to: https://github.com/settings/tokens
3. Click "Generate new token (classic)"
4. Select scopes: `repo`, `workflow`, `admin:repo_hook`
5. Copy the token (starts with `ghp_`)
6. Enter the token in the app

## ❌ Don't Use GitHub Apps

Avoid creating GitHub Apps for this tool because:
- Complex installation process
- Requires repository permission setup
- Token starts with `ghu_` (limited access)
- Needs installation-specific API calls