# Task Plan: WeChat 4.1 Windows Passphrase Extraction

## Goal

Support extracting valid WeChat 4.1.10.53 Windows database keys locally, then verify chatlog can read sessions without exposing secrets.

## Current Phase

Phase 3

## Phases

### Phase 1: Context and Upstream Research

- [x] Confirm current failure facts project boundaries
- [x] Inspect current Windows v4 extraction/decryption code
- [x] Inspect upstream passphrase approach
- **Status:** complete

### Phase 2: Minimal Implementation

- [x] Add Windows v4 fallback after legacy key/salt scan fails
- [x] Derive per-DB keys from passphrase and DB salt
- [x] Write `all_keys.json` only after strict DB HMAC validation
- [x] Make Windows v4 decryptor return SQLCipher4 HMAC/PBKDF2 parameters for WAL handling
- **Status:** complete

### Phase 3: Local Verification

- [x] Run local syntax/type checks for changed packages
- [ ] Run authorized local key extraction without printing secrets
- [ ] Verify `/health` and `/api/v1/sessions` status/count only
- **Status:** in_progress

### Phase 4: Cleanup Report

- [ ] Remove stale bad cache if needed
- [x] Re-run codegraph index because files were created/deleted
- [ ] Report exact result remaining risk
- **Status:** in_progress

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Keep scope to Windows v4 extraction/decryption path | HTTP/TUI already work once keys valid. |
| Do not print data_key/passphrase/all_keys contents | Secrets local only not needed in chat. |
| No Git actions | Root project rules forbid Git without explicit authorization. |
| Use HMAC validation before writing keys | Header-only validation can accept false positives on Windows v4. |
| Use breakpoint capture instead of broad PBKDF2 memory scan | Full memory candidate PBKDF2 is too slow and already produced bad heuristics. |

## Errors Encountered

| Error | Attempt | Resolution |
|-------|---------|------------|
| CGO disabled caused sqlite3 stub | Prior attempt | Fixed with project-local w64devkit CGO build. |
| Legacy `x'key+salt'` scan found no candidate | Prior attempt | Implemented newer passphrase fallback. |
| Naive nearby-key heuristic produced invalid keys | Prior attempt | Removed bad keys/cache; require strict HMAC validation. |
| Dynamic DLL `Proc.Call` returned `error` not `syscall.Errno` | First local package check | Changed wrapper to accept `error`. |
| WCDB cipher xrefs not found in main EXE | First elevated extraction attempt | Expanded lookup to loaded modules such as `Weixin.dll`. |
| UAC did not complete | Second elevated extraction attempt | Waiting for user approval or visible-window retry. |
| Passphrase capture returned `The handle is invalid` | First loaded-module extraction run | Added debug-event context to the returned error. |
| Debugger wait returned `handle invalid` | Capture after module-scan fix | Locked debugger API calls to one OS thread. |
| Breakpoint did not trigger | Stable capture on already-running WeChat | Added watch-mode helper to attach to a newly started main process. |
| Watch mode did not see new PID | Current run | Need manual full WeChat exit/reopen or explicit permission to terminate Weixin. |
| Passphrase-first capture still missed init | Restart capture | Added module-load wait and skipped legacy scan delay. |
| WCDB anchor did not fire | Restart capture | Broadened fallback breakpoints to SQLCipher/KDF/HMAC anchors. |
| Broadened 21 breakpoints still timed out | Latest run | Need user UI action during capture or lower-level hook. |
