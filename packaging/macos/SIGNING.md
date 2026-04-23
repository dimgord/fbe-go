# macOS codesigning + notarization — one-time setup

This file documents the secrets the GitHub Actions release workflow needs in
order to ship signed + notarized `.dmg` artifacts. It's checked in so the
setup is reproducible if the Secrets get wiped or the repo is forked.

**What you get after setup:** `v1.0.0` (and any future tag) produces a DMG
that Gatekeeper clears without the `"fbe-go" cannot be opened because the
developer cannot be verified` dialog. Users double-click, it runs.

**What's required:** an Apple Developer Program membership ($99/yr), a
Developer ID Application certificate, and an Apple ID.

## 1. Find your Team ID

Log in at <https://developer.apple.com/account> → Membership. The 10-
character Team ID is shown there (`ABCD1E2345`-style). Save it for Step 5.

## 2. Generate / export the Developer ID Application cert

Most developer accounts already have one from a previous CLI project. If
not:

1. Xcode → Settings → Accounts → your Apple ID → Manage Certificates → `+`
   → **Developer ID Application**. (You need at least one **Account Holder**
   / **Admin** role in the team to create this cert; personal accounts
   have it.)
2. Keychain Access → **login** keychain → Category: **My Certificates** →
   find `Developer ID Application: Your Name (TEAMID)` → right-click →
   **Export** → save as `developer_id_application.p12`. You'll be prompted
   for an **export password** — pick any strong string; you'll paste it
   into Secrets as `APPLE_DEVELOPER_ID_APPLICATION_P12_PASSWORD`.

**Verify the .p12 is complete** (cert + private key):

```bash
openssl pkcs12 -info -in developer_id_application.p12 -nodes -legacy \
  | grep -E "friendlyName|subject="
```

You should see at least one **private key** entry and the matching
certificate subject. If there's no private key, re-export and make sure
you select *both* the certificate and its attached key in Keychain Access
before File → Export.

## 3. Generate an app-specific password for notarization

App-specific passwords are one-off tokens tied to your Apple ID — safer
than pasting your main password into CI.

1. Log in at <https://appleid.apple.com/account/manage>.
2. Sign-In and Security → **App-Specific Passwords** → `+` → label it
   `fbe-go-notarization`.
3. Copy the 19-character `xxxx-xxxx-xxxx-xxxx` string. This is
   `APPLE_APP_SPECIFIC_PASSWORD` in Secrets.

## 4. Base64-encode the .p12

GitHub Secrets can only hold text, so the binary .p12 has to be base64:

```bash
base64 -i developer_id_application.p12 -o developer_id_application.p12.base64
```

The output is the value for `APPLE_DEVELOPER_ID_APPLICATION_P12_BASE64`.

## 5. Add five repository secrets

Repository → Settings → Secrets and variables → **Actions** → New
repository secret. Add these five, exactly these names:

| Secret name                                       | Value                                                  |
|---------------------------------------------------|--------------------------------------------------------|
| `APPLE_TEAM_ID`                                   | 10-character team ID from step 1                       |
| `APPLE_ID`                                        | Your Apple ID email                                    |
| `APPLE_APP_SPECIFIC_PASSWORD`                     | App-specific password from step 3                      |
| `APPLE_DEVELOPER_ID_APPLICATION_P12_BASE64`       | Contents of the `.p12.base64` file from step 4         |
| `APPLE_DEVELOPER_ID_APPLICATION_P12_PASSWORD`     | The export password you chose in step 2                |

The release workflow detects these by checking `APPLE_TEAM_ID != ''`. If
any secret is missing, the signing + notarization steps are skipped and
the workflow falls back to producing the unsigned DMG — so older tags can
be rebuilt without backporting the cert setup.

## 6. Test on a throwaway tag

Before tagging `v1.0.0-rc1`, verify the pipeline works end-to-end on a
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
one that's never seen the dev build):

```bash
# Remove any quarantine attributes the browser added (simulates a fresh
# download of a real release):
xattr -dr com.apple.quarantine ~/Downloads/fbe-go-*.dmg || true

# Ask Gatekeeper what it thinks:
spctl --assess --type open --context context:primary-signature --verbose ~/Downloads/fbe-go-*.dmg
```

The output should include `source=Notarized Developer ID`. Double-click
the DMG; drag the app to Applications; first launch should skip the
"unidentified developer" dialog.

If Gatekeeper still complains:
- `Source=Unnotarized Developer ID` → notarization step didn't run or
  didn't staple. Check Actions logs for `stapler staple` errors.
- `Source=No usable signature` → the .p12 was exported without its
  private key. Redo step 2.

After a successful test, delete the throwaway tag + release:

```bash
git push origin :v0.2.0-signtest
gh release delete v0.2.0-signtest --yes
```

## 7. Rotating

- **App-specific password** expires only when you revoke it at
  appleid.apple.com. Rotate yearly for hygiene.
- **Developer ID cert** expires 5 years from issue. Renew via Xcode at
  year 4 and re-upload the new base64.
- **Team ID** never changes.

## Troubleshooting

**`security: SecKeychainItemImport: One or more parameters passed to a
function were not valid`** — the p12 password is wrong, or the base64
decoded to garbage (retype the secret; no stray newlines).

**`notarytool submit` hangs for 30+ min** — Apple's queue, not us. The
`--timeout 30m` caps our wait; in practice it's under 5 min on weekdays.

**`stapler staple` says "The staple and validate action failed"** — the
notary verdict was `Invalid`. Run `xcrun notarytool log <submission-id>
…` with the creds above to see the specific entitlements / signature
rejection.

**`codesign: no identity found`** — the p12 imported OK but the cert's
Common Name doesn't start with `Developer ID Application:`. If you
accidentally exported an "Apple Development" or "Mac App Distribution"
cert, re-export the Developer ID one specifically.
