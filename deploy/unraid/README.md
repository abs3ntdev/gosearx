# Deploying gosearx on Unraid

gosearx runs as a single Docker container. The image is published to GHCR at
`ghcr.io/abs3ntdev/gosearx:latest` (multi-arch: amd64 + arm64). Valkey/Redis is
**optional** and runs as a separate container.

## Option A — Add Container (manual, quickest)

Unraid → **Docker** tab → **Add Container**, then set:

| Field | Value |
|---|---|
| **Repository** | `ghcr.io/abs3ntdev/gosearx:latest` |
| **Network Type** | `Bridge` |
| **Port** | Container `8080` → Host e.g. `8080` (TCP) |
| **WebUI** | `http://[IP]:[PORT:8080]/` |

Add variables/paths as needed (all optional) with **Add another Path, Port,
Variable**:

| Type | Name | Container value | Notes |
|---|---|---|---|
| Variable | `VALKEY_URL` | `valkey://valkey:6379/1` | Point at your Valkey container; blank = in-memory |
| Variable | `GITHUB_TOKEN` | *(your token)* | 5000 req/hr + code search; mask it |
| Variable | `BRAVE_API_KEY` | *(your key)* | For the braveapi engine |
| Path (ro) | settings | `/app/settings.yml` ← `/mnt/user/appdata/gosearx/settings.yml` | Optional custom config |
| Path (ro) | plugins | `/app/custom-plugins` ← `/mnt/user/appdata/gosearx/plugins` | Optional custom plugins |

Apply, then open `http://YOUR-UNRAID-IP:8080`.

## Option B — Template XML

`deploy/unraid/gosearx.xml` is a Community-Apps-style template. To use it:

1. Copy `gosearx.xml` to `/boot/config/plugins/dockerMan/templates-user/` on
   your Unraid box (e.g. via the terminal or the `flash` share).
2. Docker → **Add Container** → select **gosearx** from the *Template*
   dropdown. The fields above are pre-filled.

## Valkey (optional companion container)

You said you already run Valkey externally — just set `VALKEY_URL` to reach it.
If you don't have one yet, add a second container:

| Field | Value |
|---|---|
| Repository | `valkey/valkey:8-alpine` |
| Network | same as gosearx (`Bridge`, or a custom Docker network) |
| Name | `valkey` |
| Post Arguments | `valkey-server --save 60 1 --appendonly no` |

Then set gosearx's `VALKEY_URL=valkey://valkey:6379/1`.

> Networking note: on the default `bridge` network, containers reach each other
> by **host IP + published port**, not by name. To use `valkey://valkey:6379/1`
> (name-based), put both containers on the **same custom Docker network**
> (Unraid → Docker → *Add network*), or use
> `VALKEY_URL=redis://YOUR-UNRAID-IP:6379/1` with Valkey's port published.

## Notes & gotchas

- **Port:** the container always listens on `8080` internally (the image's
  `CMD` passes `-addr :8080`). Map it to whatever host port you like.
- **Permissions:** the container runs as uid `65532`. Mount config/plugins
  **read-only (`ro`)**; world-readable files (Unraid appdata defaults) work
  fine. No write access is needed.
- **Custom settings.yml:** start from the repo's `settings.yml`, drop it at
  `/mnt/user/appdata/gosearx/settings.yml`, and map it `ro`. Secrets stay in
  env vars (`${GITHUB_TOKEN}` etc. expand at load) — don't paste them into the
  file.
- **Custom plugins:** drop `.lua`/`.mjs`/`.sh`/`.py` files in
  `/mnt/user/appdata/gosearx/plugins`; they **add to** the built-ins. Exec
  plugins work (the image ships `bash`/`python3`/`curl`).
- **AI synthesis:** to use a local Ollama, run Ollama on Unraid and set
  `ai.enabled: true` + `ai.base_url` in a mounted settings.yml pointing at it
  (e.g. `http://YOUR-UNRAID-IP:11434`).
- **GHCR auth:** if the package is private, log Unraid's Docker into GHCR first
  (Docker → *Add Container* → registry login, or `docker login ghcr.io`). Making
  the package public avoids this.
- **Updates:** Unraid's Docker tab shows an update when a new `:latest` is
  pushed; click **update**. CI republishes on every push to `main`.
