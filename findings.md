# Findings: WeChat 4.1 Windows Passphrase Extraction

## Known Local Facts

- Target account: `wxid_8akg7yot0sxo22_0b24`.
- Target data dir: `D:\xwechat_files\wxid_8akg7yot0sxo22_0b24`.
- WeChat version observed: `4.1.10.53`.
- Project-local Go toolchain exists at `.cache/toolchains/go1.24.0.windows-amd64/go/bin/go.exe`.
- Project-local CGO compiler that worked: `.cache/toolchains/w64devkit-1.23.0/w64devkit/bin/gcc.exe`.

## Prior Verification

- HTTP `/health` worked after CGO build.
- `/api/v1/sessions` failed with `file is not database` when invalid generated keys present.
- Bad generated `all_keys.json`, wcdb cache, config keys cleared before task.

## Technical Facts

- Existing Windows v4 scan searches memory ASCII `x'<64 hex key><32 hex salt>'`.
- On machine, admin memory read worked found DB salts in memory.
- No legacy `x'key+salt'` candidates found.
- strict first-page validation rejected naive nearby 32-byte candidates.
- Upstream `wcdb-key-tool` validates SQLCipher4 enc keys by HMAC-SHA512 over page 1: MAC salt is DB salt XOR `0x3A`, MAC key is PBKDF2-SHA512(enc_key, mac_salt, 2, 32B).
- Upstream passphrase mode derives each DB enc_key as PBKDF2-SHA512(passphrase, db_salt, 256000, 32B).
- Windows `x/sys` in the local module cache exposes process memory read/write but not DebugActiveProcess/GetThreadContext wrappers, so debugger fallback needs `kernel32.dll` dynamic calls.

## Open Questions

- Whether the Windows cipher-name breakpoint exposes passphrase candidates in registers/stack for WeChat 4.1.10.53 without restarting WeChat.
