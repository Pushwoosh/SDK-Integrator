---
name: setup
description: Integrate the Pushwoosh SDK into the current project with an install code from the Pushwoosh Control Panel. Use when the user runs /pushwoosh:setup <code> or asks to add Pushwoosh push notifications to this project.
---

# Pushwoosh setup

The install code (`pwi_…`) comes from the Control Panel. It works once and expires; if it is rejected, ask the user for a fresh one, or create one with the `create_sdk_install_code` tool of the `pushwoosh` MCP server.

Run every step from the project root. `INSTALL` below is:

```
curl -fsSL https://github.com/Pushwoosh/SDK-Integrator/releases/latest/download/install.sh | sh -s --
```

1. Show the plan without spending the code: `INSTALL --dry-run <code>`. Show the user the diff it prints.
2. When the user agrees, apply it: `INSTALL --yes <code>`. It writes the service worker, adds the init code, and installs the package with the project's own package manager.
3. If step 1 reports an unsupported stack, integrate by hand:
   - `INSTALL redeem <code>` prints `{"application_code", "api_token"}`; the code is spent.
   - Follow the official guide for the stack at https://docs.pushwoosh.com and pass **both** `applicationCode` and `apiToken` to the SDK init. Without the device API token, applications of accounts registered after 21 April 2025 reject device registration.
   - For the Web SDK, `pushwoosh-service-worker.js` must be served from the site root with `importScripts('https://cdn.pushwoosh.com/webpush/v3/pushwoosh-service-worker.js' + self.location.search);`.
4. Finish by telling the user to open the app or site and allow notifications; the device then appears in the Control Panel.

Do not change files unrelated to the integration, and do not regenerate or replace keys of an existing Pushwoosh setup.
