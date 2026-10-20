# Pushwoosh SDK Integrator

SDK Integrator wires the Pushwoosh SDK into your project with one command. Copy it from the Pushwoosh Control Panel:

```sh
curl -fsSL https://github.com/Pushwoosh/SDK-Integrator/releases/latest/download/install.sh | sh -s -- <install-code>
```

It detects your stack, shows the full diff, and changes nothing until you confirm. Then it writes the init code with your application code and device API token, adds the service worker, and installs the package with your own package manager.

| Stack | What it does |
|---|---|
| Next.js (app router) | `public/pushwoosh-service-worker.js`, a `PushwooshInit` client component mounted in the root layout |
| Nuxt | `public/pushwoosh-service-worker.js`, `plugins/pushwoosh.client.ts` |
| Vite (React, Vue, …) | `public/pushwoosh-service-worker.js`, init in `src/main.*` |
| Static HTML | `pushwoosh-service-worker.js` and the CDN snippet in `index.html` |

Mobile SDKs (iOS, Android, React Native, Flutter, Unity, …) are coming. Until then, and for any stack above that is not detected, use your AI agent.

## With an AI agent

```sh
curl -fsSL https://github.com/Pushwoosh/SDK-Integrator/releases/latest/download/install.sh | sh -s -- <install-code> claude
curl -fsSL https://github.com/Pushwoosh/SDK-Integrator/releases/latest/download/install.sh | sh -s -- <install-code> cursor
```

- **Claude Code**: adds this repository as a plugin marketplace and installs the `pushwoosh` plugin (the `/pushwoosh:setup` skill and the Pushwoosh MCP server). Then run `/pushwoosh:setup <install-code>`.
- **Cursor**: adds the Pushwoosh MCP server to `.cursor/mcp.json` and a `pushwoosh` rule to `.cursor/rules/`.

## What it will not do

- Write anything before you confirm, or when the project has uncommitted changes (unless `--force`).
- Send your source code anywhere. The only network calls are the download from this repository's releases and one request that redeems the install code.
- Replace keys of an existing Pushwoosh setup, or initialize Pushwoosh twice.

## Verify what you run

`install.sh` downloads the binary from the same GitHub release and checks it against that release's `checksums.txt`. Every release is built by [GitHub Actions](.github/workflows/release.yml) from its tag, with a build provenance attestation:

```sh
gh attestation verify pushwoosh-integrator_darwin_arm64 --repo Pushwoosh/SDK-Integrator
```

To pin a version, use `releases/download/vX.Y.Z/install.sh` instead of `releases/latest/download/install.sh`.

## Flags

```
pushwoosh-integrator [--dry-run] [--yes] [--force] [--dir DIR] [--api URL] <install-code> [claude|cursor]
pushwoosh-integrator redeem <install-code>   # print application_code and api_token as JSON
pushwoosh-integrator verify [--timeout 15m] <install-code>   # wait for a test device, then send it a test push
```

`verify` works with the same code for an hour after the install. It opens the application's test-device window; open the app and send it to the background and back 6 times within 30 seconds (the knock gesture of the iOS and Android SDKs), and a test push goes to that device. The Web SDK has no knock yet.

The install code works once and expires after 30 minutes. `--dry-run` and a declined confirmation leave it unused.
