# Progress Log

## Session: 2026-06-28

### Phase 1: Context and Upstream Research

- **Status:** complete
- Loaded root and agent rules.
- Loaded planning files workflow.
- Started focused plan for WeChat 4.1 Windows passphrase extraction.
- Confirmed project has no separate Go tech rule; project rules cover this task.
- Confirmed CodeGraph index is initialized and inspected Windows v4 extraction/decryption symbols.
- Confirmed old Windows v4 validation is not strict enough for candidate discovery because first-page decrypt writes the SQLite header before checking decrypted body.
- Confirmed passphrase derivation parameters from upstream: PBKDF2-SHA512(passphrase, db_salt, 256000, 32B) and SQLCipher HMAC validation with PBKDF2-SHA512(enc_key, salt^0x3A, 2, 32B).

### Phase 2: Minimal Implementation

- **Status:** complete
- Added Windows v4 passphrase fallback after old `key+salt` memory scan.
- Added strict SQLCipher4 HMAC validation before accepting old scan keys or passphrase-derived keys.
- Added Windows v4 decryptor HMAC/PBKDF2 support for validation and WAL frame handling.
- Kept passphrase/data keys out of status text and output.

### Phase 3: Local Verification

- **Status:** in_progress
- `go test .\internal\wechat\key\windows .\internal\wechat\decrypt\windows` passed.
- `go test .\internal\chatlog\wechat` passed.
- Real extraction not run in this turn because it requires UAC/admin attach and likely user re-login during the capture window.
- First admin verification reached the passphrase fallback but failed to locate WCDB xrefs in the main EXE; fixed lookup to scan loaded modules.
- Rebuilt the temporary verification helper and restarted UAC verification, but Windows UAC is still waiting for user approval; no key extraction process has started yet.
- Added debug-event context to the Windows passphrase capture error path after the first module-scan run returned `The handle is invalid`.
- Rebuilt the helper and started another elevated verification; Windows UAC is again waiting for approval.
- Added `runtime.LockOSThread()` around Windows debugger attach/wait/continue after `wait debug event failed: handle invalid`, which stopped the immediate invalid-handle failure.
- A stable capture attempt timed out because the WCDB initialization breakpoint did not fire in the already-running WeChat process.
- Replaced the temporary verification helper with a watch mode that ignores existing Weixin main PIDs and waits for a newly started main process before attaching.
- Watch mode ran with UAC approval but timed out waiting for a new Weixin main process; current main process remained PID 45568.
- Added passphrase-first key initialization path so the watch helper skips legacy memory scan and attaches before the new process finishes DB setup.
- Added a short wait for Weixin DLL modules to load before locating WCDB xrefs.
- Broadened passphrase breakpoints from the single WCDB cipher anchor to SQLCipher/KDF/HMAC anchor references; the latest run set 21 breakpoints.
- Latest authorized restart/capture run still timed out; no breakpoint fired during the 120s capture window.

### Phase 4: Cleanup Report

- **Status:** in_progress
- Ran `codegraph index` after creating/deleting files.

## Test Results

| Check | Command | Result |
|-------|---------|--------|
| Previous CGO issue | `/api/v1/sessions` after CGO build | sqlite3 stub error removed, but invalid key caused bad DB. |
| Changed Windows key/decrypt packages | `go test .\internal\wechat\key\windows .\internal\wechat\decrypt\windows` | Passed. |
| Direct WAL caller package | `go test .\internal\chatlog\wechat` | Passed. |
| First real extraction attempt | elevated `.cache\keysetup\keysetup-cgo.exe` | Reached passphrase fallback; failed because WCDB anchor was in DLL modules, not main EXE. |
| Second real extraction attempt | elevated `.cache\keysetup\keysetup-cgo.exe` | Waiting on Windows UAC approval; status file not written yet. |
| Debug-context rebuild | `go test .\internal\wechat\key\windows .\internal\wechat\decrypt\windows .\internal\chatlog\wechat` and helper build | Passed. |
| Third real extraction attempt | elevated `.cache\keysetup\keysetup-cgo.exe` | Waiting on Windows UAC approval; status file not written yet. |
| Debugger thread affinity fix | `go test .\internal\wechat\key\windows .\internal\wechat\decrypt\windows .\internal\chatlog\wechat` and helper build | Passed. |
| Stable capture attempt | elevated `.cache\keysetup\keysetup-cgo.exe` | Timed out waiting for WCDB initialization in already-running WeChat. |
| Watch-mode helper attempt | elevated `.cache\keysetup\keysetup-cgo.exe` | Timed out waiting for a new Weixin main process. |
| Passphrase-first rebuild | `go test .\internal\wechat\key\windows .\internal\wechat\decrypt\windows .\internal\chatlog\wechat` and helper build | Passed. |
| Module-wait restart capture | elevated `.cache\keysetup\keysetup-cgo.exe` | Found new PID but timed out with 1 WCDB breakpoint. |
| Broadened SQLCipher breakpoint capture | elevated `.cache\keysetup\keysetup-cgo.exe` | Found new PID, set 21 breakpoints, still timed out. |

## Error Log

| Timestamp | Error | Resolution |
|-----------|-------|------------|
| 2026-06-28 | Legacy key/salt scan incompatible with WeChat 4.1.10.53 | Research and implement passphrase fallback. |
| 2026-06-28 | Main EXE did not contain WCDB cipher xrefs | Expanded passphrase breakpoint lookup to loaded modules. |
| 2026-06-28 | UAC prompt still waiting | User needs to approve or confirm no visible popup. |
| 2026-06-28 | Passphrase capture returned `The handle is invalid` | Added debug-event context to the returned error for the next run. |
| 2026-06-28 | Windows debugger wait returned invalid handle | Locked debugger operations to one OS thread. |
| 2026-06-28 | Capture timed out on current WeChat process | Need attach before DB initialization by restarting WeChat. |
| 2026-06-28 | Watch mode did not see a new main Weixin PID | WeChat was not fully exited; manual full exit or explicit permission to terminate is required. |
| 2026-06-28 | New process was detected before WCDB module/xrefs loaded | Added module/xref wait before attaching breakpoints. |
| 2026-06-28 | WCDB anchor path did not execute during startup | Broadened breakpoints to SQLCipher/KDF/HMAC anchors. |
| 2026-06-28 | Broadened breakpoints still did not fire | Need manual UI action that opens encrypted DB during capture or a lower-level hook. |
