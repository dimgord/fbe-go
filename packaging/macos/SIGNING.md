# macOS codesigning + notarization — one-time setup

This file documents the secrets the GitHub Actions release workflow needs in
order to ship signed + notarized `.dmg` artifacts. It's checked in so the
setup is reproducible if the Secrets get wiped or the repo is forked.

**What you get after setup:** `v1.0.0` (and any future tag) produces a DMG
that Gatekeeper clears without the `"fbe-go" cannot be opened because the
developer cannot be verified` dialog. Users double-click, it runs.

**What's required:** an Apple Developer Program membership ($99/yr), a
Developer ID Application certificate, and an App Store Connect API key.

Notarization uses an **App Store Connect API key** (not the legacy
`--apple-id` + app-specific password flow). This is Apple's recommended
path for CI: not tied to a specific Apple ID, scopable per key, revocable
without rotating your account password. The codesigning step still needs
the Developer ID Application certificate — that part hasn't moved.

## 1. Export the Developer ID Application cert (for codesigning)

Apple issues several distinct certificate types — **only
`Developer ID Application` works for notarized distribution outside the
Mac App Store**. The one you probably already have in Keychain (`Apple
Development`) is for local Xcode builds on your own devices and is
**not** accepted by notarytool. Check before you waste 15 minutes:

```bash
security find-identity -v -p codesigning \
  | grep -i "developer id application"
```

No output → you don't have one yet; create it first (§1a). Output
exists → skip to §1b and export.

### 1a. Create a new Developer ID Application cert

**Preferred path — Xcode:**

1. Xcode → Settings (`⌘,`) → **Accounts**.
2. Select your Apple ID in the left list → **Manage Certificates…**.
3. Bottom-left `+` → **Developer ID Application**.
4. Xcode generates a CSR locally, requests the cert, and installs it in
   your login keychain automatically.

(Requires **Admin** or **Account Holder** role on your developer team;
solo / personal accounts have this by default.)

**Fallback path — manual, via developer.apple.com:**

If Xcode refuses ("You already have a current cert" when the existing
one is a different type, or silent no-op):

1. Keychain Access → menu **Keychain Access** → **Certificate Assistant**
   → **Request a Certificate From a Certificate Authority…**
   - User Email Address: your Apple ID email
   - Common Name: your name
   - CA Email Address: leave blank
   - Request is: **Saved to disk** (mandatory — don't pick "Emailed")
   - Save the `.certSigningRequest` file somewhere
   - Keychain silently generates a paired private key in your login
     keychain; keep it there until the issued cert is installed.
2. <https://developer.apple.com/account/resources/certificates/list>
   → blue `+` button → under **Software** pick **Developer ID
   Application** → Continue.
3. CA selection prompt: pick **G2 Sub-CA (Xcode 11.4.1 or later)**.
4. Upload the `.certSigningRequest` → Continue → Download the issued
   `.cer` file.
5. Double-click the `.cer` to install into the login keychain. Since
   the matching private key is already there (step 1), Keychain Access
   automatically pairs them.

### 1b. Verify and export

In Keychain Access, **switch to the `My Certificates` tab** (top bar) —
not the `Certificates` tab, which lists *all* CA certs including
Apple's intermediate authorities (names like `Developer ID
Certification Authority` — these are Apple's internal signing CAs, not
your personal cert; clicking them doesn't get you anything).

Find the entry named literally `Developer ID Application: Your Name
(TEAMID)`. Click the disclosure triangle — a child row with the 🔑 icon
and name matching the cert must be visible. **No key visible → the
cert is orphaned; don't bother exporting.** Go back to §1a and re-do
the CSR path.

Right-click the cert (not the key inside it) → **Export** → `.p12`
format → `developer_id_application.p12`. You'll be prompted for an
**export password**; pick any strong string and record it — this is
the value of the `APPLE_DEVELOPER_ID_APPLICATION_P12_PASSWORD` secret
in step 4.

**Verify the .p12 is complete** (cert + private key):

```bash
openssl pkcs12 -info -in developer_id_application.p12 -nodes -legacy \
  | grep -E "friendlyName|subject="
```

Expected output:

```
friendlyName: Developer ID Application: Your Name (TEAMID)
subject=UID=TEAMID, CN=Developer ID Application: Your Name (TEAMID), ...
friendlyName: Mac Developer ID Application: Your Name
```

Three things to confirm:

- `friendlyName` / `CN` starts with **`Developer ID Application:`** —
  not `Apple Development:`, `iPhone Developer:`, or `Mac Installer:`.
  Those are different certs that look similar in the UI but don't work
  for distribution notarization.
- `UID` matches your Team ID (10 alphanumeric chars).
- A `Mac Developer ID Application: …` entry for the **private key** also
  appears. If this line is missing, the .p12 has no key in it —
  Keychain Access's Export skips the key when it can't find one for
  the cert. Re-check §1b's "child row" visibility and re-export.

## 2. Generate an App Store Connect API key (for notarization)

The API key is a `.p8` file + two IDs. It has nothing to do with
`account.apple.com` — it lives in App Store Connect.

1. Sign in at **<https://appstoreconnect.apple.com/access/integrations/api>**
   (Users and Access → Integrations → App Store Connect API). If the URL
   redirects to the dashboard, navigate: top nav **Users and Access** →
   **Integrations** tab → **App Store Connect API**.
2. Click **Generate API Key** (or the `+` button if keys already exist).
3. **Name:** `fbe-go-notarization`
4. **Access:** select **Developer** (minimum role for notarytool —
   principle of least privilege; don't give it Admin).
5. Click **Generate**. The `.p8` file downloads — **this is a one-shot
   download**, Apple won't let you retrieve it again.
6. Note three values:
   - **Issuer ID** — UUID at the top of the API keys page (same for all
     keys in your team), e.g. `69a6de70-03db-47e3-e053-5b8c7c11a4d1`.
   - **Key ID** — 10-character alphanumeric in the table next to the key
     name, e.g. `ABCD1E2345`.
   - **Key file** — downloaded as `AuthKey_ABCD1E2345.p8`.

## 3. Base64-encode both secrets

GitHub Secrets can only hold text, so the binary files have to be
base64-encoded:

```bash
base64 -i developer_id_application.p12 -o developer_id_application.p12.base64
base64 -i AuthKey_ABCD1E2345.p8        -o AuthKey.p8.base64
```

On macOS, `base64` is from coreutils and produces single-line output by
default — paste the entire file contents straight into the Secret value.

**⚠️ The base64 outputs are sensitive.** Despite looking like noise,
they decode back to your real private key material. Treat them with
password-level discipline: don't paste into chats, screenshots, pastebins,
ChatGPT, or public logs. If one leaks, revoke immediately (§6) — for the
.p8 that's a 30-second fix in App Store Connect; for the .p12 you'd need
to revoke the cert in developer.apple.com which breaks stapled already-
signed artifacts.

To move the blob into GitHub's Secret field without typos, pipe through
`pbcopy`:

```bash
cat developer_id_application.p12.base64 | pbcopy
# → switch to browser, paste into the Secret value field
cat AuthKey.p8.base64 | pbcopy
# → same
```

## 4. Add five repository secrets

Repository → Settings → Secrets and variables → **Actions** → green
**New repository secret** button on the right. Add these five, exactly
these names:

| Secret name                                       | Value                                                 |
|---------------------------------------------------|-------------------------------------------------------|
| `APPLE_API_KEY_ID`                                | 10-char Key ID from step 2 (e.g. `ABCD1E2345`)        |
| `APPLE_API_ISSUER_ID`                             | UUID Issuer ID from step 2                            |
| `APPLE_API_KEY_P8_BASE64`                         | Contents of `AuthKey.p8.base64`                       |
| `APPLE_DEVELOPER_ID_APPLICATION_P12_BASE64`       | Contents of `developer_id_application.p12.base64`     |
| `APPLE_DEVELOPER_ID_APPLICATION_P12_PASSWORD`     | Export password you chose in step 1                   |

### GitHub UI gotchas

The **New repository secret** form has two separate fields, stacked:

```
Name *
┌─────────────────────────────────────┐
│ APPLE_API_KEY_P8_BASE64             │   ← short identifier [A-Z0-9_]
└─────────────────────────────────────┘

Secret *
┌─────────────────────────────────────┐
│ LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0t... │   ← the actual value (any content)
│                                     │
└─────────────────────────────────────┘

            [ Add secret ]
```

- **Name** is strict: only `[a-zA-Z0-9_]`, must start with a letter or
  underscore. GitHub rejects with:
  > Secret names can only contain alphanumeric characters ([a-z],
  > [A-Z], [0-9]) or underscores (\_). Spaces are not allowed.
  - **No leading/trailing whitespace** — pasting a secret name with a
    stray space at the start trips this too. Copy-paste the names from
    the table above exactly; double-check before Save.
- **Secret** (value) field accepts anything — whitespace, newlines,
  giant base64 blobs. This is where the actual content goes.
- The cursor defaults to the **Name** field when the form opens. If you
  auto-paste immediately (⌘V after Add secret), the blob lands in the
  wrong field. Click into Secret first.

### Detection

The release workflow detects these via `APPLE_API_KEY_ID != ''`. If the
API Key ID secret is missing, all signing + notarization steps skip and
the workflow falls back to producing the unsigned DMG — so older tags can
be rebuilt without backporting the cert setup.

## 5. Verify on a throwaway tag

Before tagging `v1.0.0-rc1`, confirm the pipeline works end-to-end on a
disposable tag:

```bash
git tag v0.2.0-signtest
git push origin v0.2.0-signtest
```

Watch the Actions run → `macOS · universal .app + .dmg` job. The two
notarization steps (`Notarize .app` and `Codesign + notarize DMG`) each
take 1–10 minutes depending on Apple's queue. Check the build log for
`status: Accepted` from `notarytool submit`.

Download the DMG from the resulting Release. On a Mac (ideally a second
one that's never seen the dev build — or at minimum, remove quarantine
on the build machine so the simulation is honest):

```bash
# Strip any browser-added quarantine attr (simulates a fresh download):
xattr -dr com.apple.quarantine ~/Downloads/fbe-go-*.dmg || true

# Ask Gatekeeper what it thinks:
spctl --assess --type open --context context:primary-signature --verbose ~/Downloads/fbe-go-*.dmg
```

Output should include `source=Notarized Developer ID`. Double-click the
DMG; drag the app to Applications; first launch should skip the
"unidentified developer" dialog.

If Gatekeeper still complains:
- `source=Unnotarized Developer ID` → notarization step ran but
  `stapler staple` didn't. Check Actions logs for staple errors.
- `source=No usable signature` → the .p12 was exported without its
  private key. Redo step 1.
- `The executable does not have the hardened runtime enabled` → the
  `--options runtime` flag was stripped from the codesign call; verify
  `.github/workflows/release.yml` has it.

After a successful test, delete the throwaway tag + release:

```bash
git push origin :v0.2.0-signtest
gh release delete v0.2.0-signtest --yes
```

## 6. Rotating

- **API Key `.p8`** — revoke at App Store Connect → Users and Access →
  Integrations → App Store Connect API → your key → **Revoke**. Then
  generate a new one and rotate the three `APPLE_API_*` secrets. Do this
  annually, or immediately if a laptop with the `.p8` was stolen.
- **Developer ID cert** — expires 5 years from issue. Renew via Xcode at
  year 4 and re-upload the new base64. The CA chain is long-lived
  independently.
- **Issuer ID** — team-level, never changes unless you transfer team
  ownership.

## Troubleshooting

**`security: SecKeychainItemImport: One or more parameters passed to a
function were not valid`** — the p12 password is wrong, or the base64
decoded to garbage (retype the secret; no stray newlines).

**`notarytool submit` hangs for 30+ min** — Apple's queue, not us. The
`--timeout 30m` caps our wait; in practice it's under 5 min on weekdays.

**`notarytool: Invalid Credentials`** — either the `.p8` base64 got
mangled in transit (re-paste), or the Key ID / Issuer ID don't match the
`.p8`. Re-check step 2.

**`stapler staple` says "The staple and validate action failed"** — the
notary verdict was `Invalid`. Run `xcrun notarytool log <submission-id>
--key $API_KEY --key-id $APPLE_API_KEY_ID --issuer $APPLE_API_ISSUER_ID`
to see the specific entitlements / signature rejection.

**`codesign: no identity found`** — the p12 imported OK but the cert's
Common Name doesn't start with `Developer ID Application:`. If you
accidentally exported an "Apple Development" or "Mac App Distribution"
cert, re-export the Developer ID one specifically.

## Trust model

Uploading signing secrets to GitHub is a trust decision. What you're
trusting:

- **GitHub's backend** — secrets are encrypted at rest (LibSodium sealed
  box, per-repo key), served over TLS 1.3 to the browser, write-only in
  the UI (plaintext never re-displayed after Save), auto-masked in
  public Action logs, and not exposed to workflows triggered by forked
  PRs. GitHub's SOC 2 Type II audit covers the operational side. You
  are **not** getting a zero-knowledge guarantee — GitHub's systems
  decrypt secrets at workflow runtime.
- **Your own machine at upload time** — the secret exists in plaintext
  on your laptop during base64 encoding and pbcopy. Clean browser
  session (no rogue extensions with "read all site data") and a
  private network reduce the attack surface to "my own OS".

What you're risking if something leaks despite this:

- **API key (.p8)** compromise → attacker can submit notary jobs as
  your team. Mitigation: **revoke in 2 clicks** at App Store Connect
  (Users and Access → Integrations → your key → Revoke). Generate a
  new one, rotate the three `APPLE_API_*` secrets, move on.
- **Developer ID cert (.p12)** compromise → attacker can codesign
  binaries as you. Mitigation is harder: revoke the cert at
  developer.apple.com — but revocation also **invalidates every
  previously-stapled artifact** you've already shipped, because
  Gatekeeper re-checks the cert chain. For a beta with few users this
  is acceptable; for production you'd plan the response before it
  matters.

For a solo open-source beta project this trust model is industry-norm
(Electron, Tauri, Obsidian, every Rust/Go project that ships signed
macOS releases lives on the same model). If you grow and need stronger
guarantees, the standard upgrade paths are:

- **OIDC federation** — GitHub Actions swaps a short-lived token with
  AWS KMS / HashiCorp Vault at runtime; no long-lived secrets in GH.
- **Self-hosted runner** on your Mac mini — secrets live in local
  keychain, GitHub sees only build output.
- **Hardware tokens** (YubiKey for codesign, CloudHSM for notary) —
  enterprise-grade, significant setup cost.

None of these are warranted at Phase-5 stage; listing them here so the
upgrade path is clear when scale changes.

## Why not app-specific passwords?

The legacy `--apple-id / --team-id / --password` notarytool auth path
still works but is end-of-life trajectory. The API-key path is:

- Tied to a team + key, not an individual's Apple ID. No breakage if
  someone's personal account email changes.
- Revocable per-key from the App Store Connect dashboard, without
  disrupting other CI jobs or your ability to sign in normally.
- Scope-minimizable: the `Developer` role on the key lets it submit
  notary jobs but nothing else — it can't publish apps, access sales
  data, or change team settings.
- Recommended by Apple for CI in current docs (notarytool man page
  promotes the `--key` flow first).
