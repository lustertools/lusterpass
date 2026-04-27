# Bitwarden Setup Guide for lusterpass

Follow these steps in order. Takes about 10-15 minutes.

## Step 1: Create a Bitwarden Account

1. Go to https://vault.bitwarden.com/#/register
2. Register with email + master password
3. Verify your email

## Step 2: Create an Organization

You need an organization to use Secrets Manager (even for solo use).

1. Log in to https://vault.bitwarden.com
2. Click **New organisation** (top right or sidebar)
3. Name it anything (e.g., `lusterpass`)
4. Select the **Free** plan
5. Click **Submit**

## Step 3: Enable Secrets Manager

1. Go to your organization's **Admin Console** (click the org name in sidebar)
2. Navigate to **Billing → Subscription**
3. Check **Subscribe to Secrets Manager**
4. Click **Submit**

## Step 4: Find Your Organization ID

You'll need this for lusterpass commands.

**Method A: From the browser URL (easiest)**
1. Open Secrets Manager in the web vault (Step 5)
2. Look at the browser address bar — the URL contains the org ID:
   ```
   https://vault.bitwarden.com/#/sm/ORG_ID_HERE/secrets
   ```
3. The UUID in the URL path is your organization ID

**Method B: From the `bws` CLI**

IMPORTANT: `bws` (Secrets Manager CLI) and `bw` (Password Manager CLI) are **different tools**.
- `bws` → `brew install bws` — this is what lusterpass needs
- `bw` → `brew install bitwarden-cli` — this is for your personal password vault, NOT used here

1. Install `bws`:
   ```bash
   curl -sSfL https://bws.bitwarden.com/install | sh
   ```
   This installs to `/usr/local/bin/bws`. (Note: `bws` is NOT in Homebrew.)
2. Run:
   ```bash
   export BWS_ACCESS_TOKEN="your-token"
   bws project list
   ```
3. The JSON response includes `organizationId` for each project

Note: The Admin Console Settings page shows an "Account Fingerprint Phrase" — this is NOT the org ID. The org ID is a UUID like `4016326f-98b6-42ff-b9fc-ac63014988f5`.

## Step 5: Open Secrets Manager

1. In the top-left of the web vault, click the **product switcher** (grid icon)
2. Select **Secrets Manager**
3. You should now see the Secrets Manager dashboard

## Step 6: Create 3 Projects

**Option A: Automated via lusterpass (recommended)**

If you've already completed Steps 7-9 (machine account + access token + `lusterpass login`), you can create the projects automatically:

```bash
# Switch to the account you want to set up
❯ lusterpass account use my2ndaccount
Active account: my2ndaccount

# Create the default projects
❯ lusterpass account setup
  + credentials (created)
  + certificates (created)
  + testing (created)

Created 3 project(s).
```

If any projects already exist, they'll be skipped:

```bash
❯ lusterpass account setup
  ✓ credentials (already exists)
  ✓ certificates (already exists)
  ✓ testing (already exists)

All projects already exist.
```

**Option B: Manual via Bitwarden web UI**

Create each one via **New → Project**:

| Project Name   | Purpose                              |
|---------------|--------------------------------------|
| `credentials` | Passwords, API keys, tokens          |
| `certificates`| SSH keys, TLS certs, long-form secrets|
| `testing`     | Sandbox for lusterpass e2e tests        |

## Step 7: Create a Machine Account

1. Click **New → Machine account**
2. Name it `lusterpass-local`
3. Click **Save**
4. Open the `lusterpass-local` machine account
5. Go to the **Projects** tab
6. Click **Add project** — add all 3 projects (`credentials`, `certificates`, `testing`)
7. Set permission to **Can read, write** for each

## Step 8: Generate an Access Token

1. Still in the `lusterpass-local` machine account
2. Go to the **Access tokens** tab
3. Click **Create access token**
4. Name: `local-dev`
5. Expiration: choose based on preference (or "Never" for dev)
6. Click **Create access token**
7. **COPY THE TOKEN IMMEDIATELY** — it cannot be retrieved later
8. Save it securely (you'll paste it into `lusterpass login` later)

The token looks like: `0.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.yyyyyyyy...`

## Step 9: Verify via CLI (Optional)

If you have `bws` installed, you can verify:

```bash
# IMPORTANT: Use single quotes — double quotes break tokens containing $ or ! characters
export BWS_ACCESS_TOKEN='your-token-here'

# bws v2.0.0 does NOT accept --organization-id (it infers from the access token)
bws project list
```

If `bws project list` returns `[]`, your machine account doesn't have access to any projects yet.
Go to Secrets Manager → Machine accounts → your account → Projects tab → add your projects.

You should see your 3 projects listed after granting access.

❯ bws project list
```json
[
  {
    "id": "af4e7796-88a1-4e68-8a62-b40a00d5ba23",
    "organizationId": "a1e4a796-78c7-41c7-8d7c-b40a00cf6392",
    "name": "credentials",
    "creationDate": "2026-03-11T12:58:09.374378300Z",
    "revisionDate": "2026-03-11T12:58:09.374378300Z"
  },
  {
    "id": "d94ebc24-40f4-4c7f-bcf7-b40a00d5d70f",
    "organizationId": "a1e4a796-78c7-41c7-8d7c-b40a00cf6392",
    "name": "certificates",
    "creationDate": "2026-03-11T12:58:34.052801500Z",
    "revisionDate": "2026-03-11T12:58:34.052801600Z"
  },
  {
    "id": "13638e39-0a48-48c6-85d9-b40a00d5e750",
    "organizationId": "a1e4a796-78c7-41c7-8d7c-b40a00cf6392",
    "name": "testing",
    "creationDate": "2026-03-11T12:58:47.926530700Z",
    "revisionDate": "2026-03-11T12:58:47.926530800Z"
  }
]
```

---

## What to Provide for lusterpass

After completing the steps above, you'll have:

| Item | Where it's used | Example |
|------|----------------|---------|
| **Organization ID** | `--org` flag on pull/enrol/list/test commands | `4016326f-98b6-...` |
| **Access Token** | `lusterpass login` (stored encrypted locally) | `0.xxxxxxxx-...` |

That's all lusterpass needs. Your master password is never used by lusterpass.

---

## Troubleshooting

**"Secrets Manager not available"**
- Make sure you created an Organization (Step 2). Personal vaults don't support Secrets Manager.

**"Access denied" when listing secrets**
- Check that the machine account has access to the correct projects (Step 7).

**"Token expired"**
- Generate a new access token (Step 8) and run `lusterpass login` again.

**Can't find Organization ID**
- Admin Console → Settings → Organization info. Or use `bws` CLI: the org ID appears in project/secret responses.
