# Security

## Report a vulnerability privately

Use [GitHub's private vulnerability report](https://github.com/smirnoffff/contgen/security/advisories/new) for authentication, payment, authorization, or credential exposure issues. Include a minimal reproduction and affected commit; redact credentials and personal data. Do not publish working secrets in issues or pull requests.

This project is maintained by an individual and has no guaranteed response time. The current `main` branch is the version maintained here.

## Keep local configuration private

- Keep bot tokens, `SESSION_SECRET`, database credentials, private datasets, and runtime files outside Git. `.env.example` contains placeholders only.
- Generate a unique session secret and database password for each deployment. Keep `DEV_LOGIN=false` when exposing the app through HTTPS.
- Do not upload `.env`, database dumps, cookies, local logs, or account sessions as CI artifacts.
- If a real credential is published, revoke or rotate it first. Deleting a file or making a new commit does not invalidate the old credential.

GitHub secret scanning and push protection are enabled for the public repository. CI also runs Gitleaks with redacted output. Automated scans reduce accidental exposure but do not prove that every possible secret has been detected.
