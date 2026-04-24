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

Most developer accounts already have one. If not:

1. Xcode → Settings → Accounts → your Apple ID → Manage Certificates → `+`
   → **Developer ID Application**. (You need at least **Admin** or
   **Account Holder** role; personal / solo accounts have it by default.)
2. Keychain Access → **login** keychain → Category: **My Certificates** →
   find `Developer ID Application: Your Name (TEAMID)` → right-click →
   **Export** → save as `developer_id_application.p12`. You'll be prompted
   for an **export password** — pick any strong string; you'll paste it
   into Secrets later as `APPLE_DEVELOPER_ID_APPLICATION_P12_PASSWORD`.

**Verify the .p12 is complete** (cert + private key):

```bash
openssl pkcs12 -info -in developer_id_application.p12 -nodes -legacy \
  | grep -E "friendlyName|subject="
```

You should see at least one **private key** entry and the matching
certificate subject. If there's no private key, re-export and make sure
you select *both* the certificate and its attached key in Keychain Access
before File → Export.

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

## 4. Add five repository secrets

Repository → Settings → Secrets and variables → **Actions** → New
repository secret. Add these five, exactly these names:

| Secret name                                       | Value                                                 |
|---------------------------------------------------|-------------------------------------------------------|
| `APPLE_API_KEY_ID`                                | 10-char Key ID from step 2 (e.g. `ABCD1E2345`)        |
| `APPLE_API_ISSUER_ID`                             | UUID Issuer ID from step 2                            |
| `APPLE_API_KEY_P8_BASE64`                         | Contents of `AuthKey.p8.base64`                       |
| `APPLE_DEVELOPER_ID_APPLICATION_P12_BASE64`       | Contents of `developer_id_application.p12.base64`     |
| `APPLE_DEVELOPER_ID_APPLICATION_P12_PASSWORD`     | Export password you chose in step 1                   |

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
