package windows

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/crypto/pbkdf2"
	syswin "golang.org/x/sys/windows"

	"github.com/sjzar/chatlog/internal/wechat/decrypt/common"
)

const (
	v4PageSize               = 4096
	v4ReserveSize            = 80
	v4HMACSize               = 64
	v4PassphraseSize         = 32
	v4PassphraseIterations   = 256000
	v4MACKeyIterations       = 2
	v4CaptureTimeout         = 120 * time.Second
	maxPassphraseBreakpoints = 24
	stackScanBytes           = 2048
)

const (
	debugExceptionEvent    = 1
	exceptionBreakpoint    = 0x80000003
	exceptionSingleStep    = 0x80000004
	dbgContinue            = 0x00010002
	dbgExceptionNotHandled = 0x80010001
	contextAMD64           = 0x00100000
	contextControl         = contextAMD64 | 0x00000001
	contextInteger         = contextAMD64 | 0x00000002
	contextFull            = contextControl | contextInteger
	eflagsTrapFlag         = 0x100
	imageScnMemExecute     = 0x20000000
	cipherAnchorString     = "com.Tencent.WCDB.Config.Cipher"
	minUserModePointer     = uintptr(0x10000)
)

var passphraseAnchorStrings = []string{
	cipherAnchorString,
	"cipher_default_kdf_iter",
	"cipher_default_settings",
	"cipher_plaintext_header_size",
	"cipher_hmac_algorithm",
	"PBKDF2",
	"SHA512",
}

var (
	errDebugWaitTimeout = errors.New("debug wait timeout")

	kernel32                      = syswin.NewLazySystemDLL("kernel32.dll")
	procDebugActiveProcess        = kernel32.NewProc("DebugActiveProcess")
	procDebugActiveProcessStop    = kernel32.NewProc("DebugActiveProcessStop")
	procDebugSetProcessKillOnExit = kernel32.NewProc("DebugSetProcessKillOnExit")
	procWaitForDebugEvent         = kernel32.NewProc("WaitForDebugEvent")
	procContinueDebugEvent        = kernel32.NewProc("ContinueDebugEvent")
	procGetThreadContext          = kernel32.NewProc("GetThreadContext")
	procSetThreadContext          = kernel32.NewProc("SetThreadContext")
	procFlushInstructionCache     = kernel32.NewProc("FlushInstructionCache")
)

type passphraseBreakpoint struct {
	addr    uintptr
	orig    byte
	enabled bool
}

type processModule struct {
	base uintptr
	path string
}

type debugEvent struct {
	code      uint32
	processID uint32
	threadID  uint32
	_         uint32
	u         [256]byte
}

type exceptionRecord struct {
	code        uint32
	flags       uint32
	record      uintptr
	address     uintptr
	paramCount  uint32
	_           uint32
	information [15]uintptr
}

type exceptionDebugInfo struct {
	record      exceptionRecord
	firstChance uint32
	_           uint32
}

type m128a struct {
	low  uint64
	high int64
}

type xmmSaveArea32 struct {
	controlWord   uint16
	statusWord    uint16
	tagWord       byte
	reserved1     byte
	errorOpcode   uint16
	errorOffset   uint32
	errorSelector uint16
	reserved2     uint16
	dataOffset    uint32
	dataSelector  uint16
	reserved3     uint16
	mxCsr         uint32
	mxCsrMask     uint32
	floatRegs     [8]m128a
	xmmRegs       [16]m128a
	reserved4     [96]byte
}

type amd64Context struct {
	p1Home uint64
	p2Home uint64
	p3Home uint64
	p4Home uint64
	p5Home uint64
	p6Home uint64

	contextFlags uint32
	mxCsr        uint32

	segCs  uint16
	segDs  uint16
	segEs  uint16
	segFs  uint16
	segGs  uint16
	segSs  uint16
	eFlags uint32

	dr0 uint64
	dr1 uint64
	dr2 uint64
	dr3 uint64
	dr6 uint64
	dr7 uint64

	rax uint64
	rcx uint64
	rdx uint64
	rbx uint64
	rsp uint64
	rbp uint64
	rsi uint64
	rdi uint64
	r8  uint64
	r9  uint64
	r10 uint64
	r11 uint64
	r12 uint64
	r13 uint64
	r14 uint64
	r15 uint64
	rip uint64

	fltSave              xmmSaveArea32
	vectorRegister       [26]m128a
	vectorControl        uint64
	debugControl         uint64
	lastBranchToRip      uint64
	lastBranchFromRip    uint64
	lastExceptionToRip   uint64
	lastExceptionFromRip uint64
}

func keyMapFromPairs(pairs []keySaltPair, dbSalts []dbSaltEntry) map[string]keyFileEntry {
	out := map[string]keyFileEntry{}
	for _, pair := range pairs {
		key, err := hex.DecodeString(strings.TrimSpace(pair.KeyHex))
		if err != nil || len(key) != common.KeySize {
			continue
		}
		for _, ds := range dbSalts {
			if pair.SaltHex != ds.SaltHex || !validateV4DBKey(key, ds.Page1) {
				continue
			}
			if _, exists := out[ds.DBRel]; !exists {
				out[ds.DBRel] = keyFileEntry{EncKey: strings.ToLower(pair.KeyHex)}
			}
		}
	}
	return out
}

func scanPassphraseKeysByPID(pid uint32, dbSalts []dbSaltEntry, status func(string)) (map[string]keyFileEntry, error) {
	passphrase, err := capturePassphraseByPID(pid, dbSalts, status)
	if err != nil {
		return nil, err
	}
	out := deriveValidatedKeysFromPassphrase(passphrase, dbSalts)
	if len(out) == 0 {
		return nil, fmt.Errorf("passphrase 已捕获，但未派生出可验证数据库密钥")
	}
	if status != nil {
		status(fmt.Sprintf("passphrase 派生完成：%d 个数据库密钥验证通过", len(out)))
	}
	return out, nil
}

func deriveValidatedKeysFromPassphrase(passphrase []byte, dbSalts []dbSaltEntry) map[string]keyFileEntry {
	out := map[string]keyFileEntry{}
	keyBySalt := map[string][]byte{}
	for _, ds := range dbSalts {
		encKey, ok := keyBySalt[ds.SaltHex]
		if !ok {
			encKey = pbkdf2.Key(passphrase, ds.Salt, v4PassphraseIterations, common.KeySize, sha512.New)
			keyBySalt[ds.SaltHex] = encKey
		}
		if validateV4DBKey(encKey, ds.Page1) {
			out[ds.DBRel] = keyFileEntry{EncKey: hex.EncodeToString(encKey)}
		}
	}
	return out
}

func validateV4DBKey(encKey []byte, page1 []byte) bool {
	if len(encKey) != common.KeySize || len(page1) < v4PageSize {
		return false
	}
	salt := page1[:common.SaltSize]
	macSalt := make([]byte, len(salt))
	for i, b := range salt {
		macSalt[i] = b ^ 0x3A
	}
	macKey := pbkdf2.Key(encKey, macSalt, v4MACKeyIterations, common.KeySize, sha512.New)
	mac := hmac.New(sha512.New, macKey)
	mac.Write(page1[common.SaltSize : v4PageSize-v4ReserveSize+common.IVSize])
	var pageNo [4]byte
	binary.LittleEndian.PutUint32(pageNo[:], 1)
	mac.Write(pageNo[:])
	return hmac.Equal(mac.Sum(nil), page1[v4PageSize-v4HMACSize:v4PageSize])
}

func capturePassphraseByPID(pid uint32, dbSalts []dbSaltEntry, status func(string)) ([]byte, error) {
	if runtime.GOARCH != "amd64" {
		return nil, fmt.Errorf("新版 passphrase 捕获当前仅支持 amd64")
	}
	probe := pickPassphraseProbeDB(dbSalts)
	if probe == nil {
		return nil, fmt.Errorf("没有可用于验证 passphrase 的数据库首页")
	}
	exeHint := processExePath(pid)
	targets, matchedModules, err := findLoadedCipherTargets(pid, exeHint, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("未在微信已加载模块中定位到 WCDB cipher 引用")
	}
	if status != nil {
		status(fmt.Sprintf("定位 WCDB cipher 引用 %d 处（%d 个模块），等待微信触发数据库初始化...", len(targets), matchedModules))
	}
	handle, err := syswin.OpenProcess(syswin.PROCESS_VM_READ|syswin.PROCESS_VM_WRITE|syswin.PROCESS_VM_OPERATION|syswin.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return nil, fmt.Errorf("open process failed: %w", err)
	}
	defer syswin.CloseHandle(handle)
	return capturePassphraseWithBreakpoints(pid, handle, targets, *probe, status)
}

func findLoadedCipherTargets(pid uint32, exeHint string, timeout time.Duration) ([]uintptr, int, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		modules, err := processModules(pid, exeHint)
		if err != nil {
			lastErr = fmt.Errorf("定位微信模块失败: %w", err)
		} else {
			targets := make([]uintptr, 0, maxPassphraseBreakpoints)
			matchedModules := 0
			for _, module := range modules {
				refs, err := findCipherAnchorXrefs(module.path)
				if err != nil || len(refs) == 0 {
					continue
				}
				matchedModules++
				for _, rva := range refs {
					targets = append(targets, module.base+uintptr(rva))
					if len(targets) >= maxPassphraseBreakpoints {
						break
					}
				}
				if len(targets) >= maxPassphraseBreakpoints {
					break
				}
			}
			if len(targets) > 0 || time.Now().After(deadline) {
				return targets, matchedModules, nil
			}
		}
		if time.Now().After(deadline) {
			if lastErr != nil {
				return nil, 0, lastErr
			}
			return nil, 0, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func pickPassphraseProbeDB(dbSalts []dbSaltEntry) *dbSaltEntry {
	preferred := []string{"session/session.db", "message/message_0.db", "message/message_1.db"}
	for _, want := range preferred {
		for i := range dbSalts {
			if normalizePath(dbSalts[i].DBRel) == want && len(dbSalts[i].Page1) >= v4PageSize {
				return &dbSalts[i]
			}
		}
	}
	for i := range dbSalts {
		if len(dbSalts[i].Page1) >= v4PageSize {
			return &dbSalts[i]
		}
	}
	return nil
}

func processExePath(pid uint32) string {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return ""
	}
	exe, err := p.Exe()
	if err != nil {
		return ""
	}
	return exe
}

func processModules(pid uint32, exeHint string) ([]processModule, error) {
	snapshot, err := syswin.CreateToolhelp32Snapshot(syswin.TH32CS_SNAPMODULE|syswin.TH32CS_SNAPMODULE32, pid)
	if err != nil {
		return nil, err
	}
	defer syswin.CloseHandle(snapshot)

	var entry syswin.ModuleEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := syswin.Module32First(snapshot, &entry); err != nil {
		return nil, err
	}

	out := make([]processModule, 0, 64)
	for {
		path := syswin.UTF16ToString(entry.ExePath[:])
		out = append(out, processModule{base: entry.ModBaseAddr, path: path})
		if err := syswin.Module32Next(snapshot, &entry); err != nil {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return modulePriority(out[i].path, exeHint) < modulePriority(out[j].path, exeHint)
	})
	if len(out) == 0 {
		return nil, fmt.Errorf("未找到已加载模块")
	}
	return out, nil
}

func modulePriority(path, exeHint string) int {
	base := strings.ToLower(filepath.Base(path))
	switch {
	case base == "weixin.dll":
		return 0
	case strings.Contains(base, "roam"):
		return 1
	case exeHint != "" && strings.EqualFold(filepath.Clean(path), filepath.Clean(exeHint)):
		return 2
	default:
		return 3
	}
}

func findCipherAnchorXrefs(exePath string) ([]uint32, error) {
	if exePath == "" {
		return nil, fmt.Errorf("微信主程序路径为空")
	}
	if _, err := os.Stat(exePath); err != nil {
		return nil, fmt.Errorf("微信主程序不可读: %w", err)
	}
	f, err := pe.Open(exePath)
	if err != nil {
		return nil, fmt.Errorf("解析微信 PE 失败: %w", err)
	}
	defer f.Close()

	anchors := map[uint32]struct{}{}
	for _, s := range f.Sections {
		data, err := s.Data()
		if err != nil {
			continue
		}
		for _, anchor := range passphraseAnchorStrings {
			needle := []byte(anchor)
			for start := 0; start < len(data); {
				idx := bytes.Index(data[start:], needle)
				if idx < 0 {
					break
				}
				anchors[s.VirtualAddress+uint32(start+idx)] = struct{}{}
				start += idx + 1
			}
		}
	}
	if len(anchors) == 0 {
		return nil, nil
	}

	seen := map[uint32]struct{}{}
	for _, s := range f.Sections {
		if s.Characteristics&imageScnMemExecute == 0 {
			continue
		}
		data, err := s.Data()
		if err != nil {
			continue
		}
		for i := 0; i+7 <= len(data); i++ {
			disp, ok := ripRelativeDisp(data[i:])
			if !ok {
				continue
			}
			instRVA := int64(s.VirtualAddress) + int64(i)
			target := instRVA + 7 + int64(disp)
			if target < 0 || target > int64(^uint32(0)) {
				continue
			}
			if _, ok := anchors[uint32(target)]; ok {
				seen[uint32(instRVA)] = struct{}{}
			}
		}
	}
	refs := make([]uint32, 0, len(seen))
	for rva := range seen {
		refs = append(refs, rva)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i] < refs[j] })
	return refs, nil
}

func ripRelativeDisp(code []byte) (int32, bool) {
	if len(code) < 7 {
		return 0, false
	}
	rex := code[0]
	if rex < 0x40 || rex > 0x4f {
		return 0, false
	}
	op := code[1]
	if op != 0x8d && op != 0x8b && op != 0x89 {
		return 0, false
	}
	if code[2]&0xc7 != 0x05 {
		return 0, false
	}
	return int32(binary.LittleEndian.Uint32(code[3:7])), true
}

func capturePassphraseWithBreakpoints(pid uint32, handle syswin.Handle, targets []uintptr, probe dbSaltEntry, status func(string)) ([]byte, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := debugActiveProcess(pid); err != nil {
		return nil, fmt.Errorf("附加调试器失败（需要管理员权限）: %w", err)
	}
	attached := true
	breakpoints := map[uintptr]*passphraseBreakpoint{}
	defer func() {
		restoreAllBreakpoints(handle, breakpoints)
		if attached {
			_ = debugActiveProcessStop(pid)
		}
	}()
	_ = debugSetProcessKillOnExit(false)

	for _, target := range targets {
		bp, err := setBreakpoint(handle, target)
		if err == nil {
			breakpoints[target] = bp
		}
	}
	if len(breakpoints) == 0 {
		return nil, fmt.Errorf("设置 passphrase 捕获断点失败")
	}
	if status != nil {
		status("passphrase 捕获已启动；如长时间无结果，请重新登录微信以触发数据库初始化")
	}

	deadline := time.Now().Add(v4CaptureTimeout)
	seenCandidates := map[string]struct{}{}
	stepping := map[uint32]*passphraseBreakpoint{}
	for time.Now().Before(deadline) {
		var ev debugEvent
		err := waitForDebugEvent(&ev, 500)
		if errors.Is(err, errDebugWaitTimeout) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("wait debug event failed: %w", err)
		}
		continueStatus := uint32(dbgContinue)
		if ev.code == debugExceptionEvent {
			passphrase, cont, err := handleDebugException(handle, &ev, breakpoints, stepping, probe, seenCandidates)
			continueStatus = cont
			if err != nil && status != nil {
				status(fmt.Sprintf("passphrase 断点处理跳过：%v", err))
			}
			if len(passphrase) == v4PassphraseSize {
				_ = continueDebugEvent(ev.processID, ev.threadID, continueStatus)
				restoreAllBreakpoints(handle, breakpoints)
				_ = debugActiveProcessStop(pid)
				attached = false
				if status != nil {
					status("passphrase 捕获成功，开始派生数据库密钥...")
				}
				return passphrase, nil
			}
		}
		if err := continueDebugEvent(ev.processID, ev.threadID, continueStatus); err != nil {
			return nil, fmt.Errorf("continue debug event failed code=%d pid=%d tid=%d status=0x%x: %w", ev.code, ev.processID, ev.threadID, continueStatus, err)
		}
	}
	return nil, fmt.Errorf("passphrase 捕获超时；请保持本程序等待时重新登录微信后重试")
}

func handleDebugException(handle syswin.Handle, ev *debugEvent, bps map[uintptr]*passphraseBreakpoint, stepping map[uint32]*passphraseBreakpoint, probe dbSaltEntry, seen map[string]struct{}) ([]byte, uint32, error) {
	info := (*exceptionDebugInfo)(unsafe.Pointer(&ev.u[0]))
	switch info.record.code {
	case exceptionBreakpoint:
		hit := info.record.address
		bp := bps[hit]
		if bp == nil {
			if ctx, err := debugThreadContext(ev.threadID); err == nil && ctx.rip > 0 {
				hit = uintptr(ctx.rip - 1)
				bp = bps[hit]
			}
		}
		if bp == nil {
			return nil, dbgContinue, nil
		}
		passphrase, singleStep, err := handlePassphraseBreakpoint(handle, ev.threadID, bp, probe, seen)
		if singleStep {
			stepping[ev.threadID] = bp
		}
		return passphrase, dbgContinue, err
	case exceptionSingleStep:
		if bp := stepping[ev.threadID]; bp != nil {
			delete(stepping, ev.threadID)
			_ = clearThreadTrapFlag(ev.threadID)
			return nil, dbgContinue, writeBreakpoint(handle, bp)
		}
		return nil, dbgExceptionNotHandled, nil
	default:
		return nil, dbgExceptionNotHandled, nil
	}
}

func debugThreadContext(threadID uint32) (*amd64Context, error) {
	thread, err := syswin.OpenThread(syswin.THREAD_GET_CONTEXT, false, threadID)
	if err != nil {
		return nil, err
	}
	defer syswin.CloseHandle(thread)
	ctx := &amd64Context{contextFlags: contextFull}
	if err := getThreadContext(thread, ctx); err != nil {
		return nil, err
	}
	return ctx, nil
}

func handlePassphraseBreakpoint(handle syswin.Handle, threadID uint32, bp *passphraseBreakpoint, probe dbSaltEntry, seen map[string]struct{}) ([]byte, bool, error) {
	thread, err := syswin.OpenThread(syswin.THREAD_GET_CONTEXT|syswin.THREAD_SET_CONTEXT|syswin.THREAD_SUSPEND_RESUME, false, threadID)
	if err != nil {
		return nil, false, err
	}
	defer syswin.CloseHandle(thread)

	ctx := amd64Context{contextFlags: contextFull}
	if err := getThreadContext(thread, &ctx); err != nil {
		return nil, false, err
	}
	passphrase := findPassphraseInContext(handle, &ctx, probe, seen)
	if err := restoreBreakpoint(handle, bp); err != nil {
		return passphrase, false, err
	}
	ctx.rip = uint64(bp.addr)
	if len(passphrase) == 0 {
		ctx.eFlags |= eflagsTrapFlag
	} else {
		ctx.eFlags &^= eflagsTrapFlag
	}
	if err := setThreadContext(thread, &ctx); err != nil {
		return passphrase, false, err
	}
	return passphrase, len(passphrase) == 0, nil
}

func clearThreadTrapFlag(threadID uint32) error {
	thread, err := syswin.OpenThread(syswin.THREAD_GET_CONTEXT|syswin.THREAD_SET_CONTEXT|syswin.THREAD_SUSPEND_RESUME, false, threadID)
	if err != nil {
		return err
	}
	defer syswin.CloseHandle(thread)
	ctx := amd64Context{contextFlags: contextFull}
	if err := getThreadContext(thread, &ctx); err != nil {
		return err
	}
	ctx.eFlags &^= eflagsTrapFlag
	return setThreadContext(thread, &ctx)
}

func findPassphraseInContext(handle syswin.Handle, ctx *amd64Context, probe dbSaltEntry, seen map[string]struct{}) []byte {
	regs := []uint64{
		ctx.rax, ctx.rbx, ctx.rcx, ctx.rdx, ctx.rsi, ctx.rdi,
		ctx.r8, ctx.r9, ctx.r10, ctx.r11, ctx.r12, ctx.r13, ctx.r14, ctx.r15,
		ctx.rbp, ctx.rsp,
	}
	for _, reg := range regs {
		if passphrase := tryPassphraseAddress(handle, uintptr(reg), probe, seen); len(passphrase) > 0 {
			return passphrase
		}
		for off := int64(-64); off <= 128; off += 8 {
			addr, ok := addOffset(uintptr(reg), off)
			if !ok {
				continue
			}
			if passphrase := tryPassphraseAddress(handle, addr, probe, seen); len(passphrase) > 0 {
				return passphrase
			}
			ptr, ok := readUintptr(handle, addr)
			if ok {
				if passphrase := tryPassphraseAddress(handle, ptr, probe, seen); len(passphrase) > 0 {
					return passphrase
				}
			}
		}
	}
	return findPassphraseOnStack(handle, uintptr(ctx.rsp), probe, seen)
}

func findPassphraseOnStack(handle syswin.Handle, rsp uintptr, probe dbSaltEntry, seen map[string]struct{}) []byte {
	if rsp < minUserModePointer {
		return nil
	}
	buf := make([]byte, stackScanBytes)
	var read uintptr
	if err := syswin.ReadProcessMemory(handle, rsp, &buf[0], uintptr(len(buf)), &read); err != nil || read < 8 {
		return nil
	}
	buf = buf[:read]
	for i := 0; i+v4PassphraseSize <= len(buf); i += 8 {
		if passphrase := tryPassphraseBytes(buf[i:i+v4PassphraseSize], probe, seen); len(passphrase) > 0 {
			return passphrase
		}
	}
	for i := 0; i+8 <= len(buf); i += 8 {
		ptr := uintptr(binary.LittleEndian.Uint64(buf[i : i+8]))
		if passphrase := tryPassphraseAddress(handle, ptr, probe, seen); len(passphrase) > 0 {
			return passphrase
		}
	}
	return nil
}

func tryPassphraseAddress(handle syswin.Handle, addr uintptr, probe dbSaltEntry, seen map[string]struct{}) []byte {
	if addr < minUserModePointer {
		return nil
	}
	buf := make([]byte, v4PassphraseSize)
	var read uintptr
	if err := syswin.ReadProcessMemory(handle, addr, &buf[0], uintptr(len(buf)), &read); err != nil || read != uintptr(len(buf)) {
		return nil
	}
	return tryPassphraseBytes(buf, probe, seen)
}

func tryPassphraseBytes(raw []byte, probe dbSaltEntry, seen map[string]struct{}) []byte {
	if len(raw) != v4PassphraseSize || bytes.Count(raw, []byte{0}) == v4PassphraseSize {
		return nil
	}
	id := string(raw)
	if _, ok := seen[id]; ok {
		return nil
	}
	seen[id] = struct{}{}
	encKey := pbkdf2.Key(raw, probe.Salt, v4PassphraseIterations, common.KeySize, sha512.New)
	if !validateV4DBKey(encKey, probe.Page1) {
		return nil
	}
	passphrase := make([]byte, len(raw))
	copy(passphrase, raw)
	return passphrase
}

func readUintptr(handle syswin.Handle, addr uintptr) (uintptr, bool) {
	if addr < minUserModePointer {
		return 0, false
	}
	var buf [8]byte
	var read uintptr
	if err := syswin.ReadProcessMemory(handle, addr, &buf[0], uintptr(len(buf)), &read); err != nil || read != uintptr(len(buf)) {
		return 0, false
	}
	return uintptr(binary.LittleEndian.Uint64(buf[:])), true
}

func addOffset(base uintptr, off int64) (uintptr, bool) {
	if off < 0 {
		delta := uintptr(-off)
		if base < delta {
			return 0, false
		}
		return base - delta, true
	}
	return base + uintptr(off), true
}

func setBreakpoint(handle syswin.Handle, addr uintptr) (*passphraseBreakpoint, error) {
	bp := &passphraseBreakpoint{addr: addr}
	var read uintptr
	if err := syswin.ReadProcessMemory(handle, addr, &bp.orig, 1, &read); err != nil || read != 1 {
		return nil, fmt.Errorf("read breakpoint byte failed: %w", err)
	}
	if err := writeBreakpoint(handle, bp); err != nil {
		return nil, err
	}
	return bp, nil
}

func writeBreakpoint(handle syswin.Handle, bp *passphraseBreakpoint) error {
	if err := writeProcessByte(handle, bp.addr, 0xcc); err != nil {
		return err
	}
	bp.enabled = true
	return nil
}

func restoreBreakpoint(handle syswin.Handle, bp *passphraseBreakpoint) error {
	if bp == nil || !bp.enabled {
		return nil
	}
	if err := writeProcessByte(handle, bp.addr, bp.orig); err != nil {
		return err
	}
	bp.enabled = false
	return nil
}

func restoreAllBreakpoints(handle syswin.Handle, bps map[uintptr]*passphraseBreakpoint) {
	for _, bp := range bps {
		_ = restoreBreakpoint(handle, bp)
	}
}

func writeProcessByte(handle syswin.Handle, addr uintptr, b byte) error {
	var oldProtect uint32
	if err := syswin.VirtualProtectEx(handle, addr, 1, syswin.PAGE_EXECUTE_READWRITE, &oldProtect); err != nil {
		return err
	}
	var written uintptr
	err := syswin.WriteProcessMemory(handle, addr, &b, 1, &written)
	var ignored uint32
	_ = syswin.VirtualProtectEx(handle, addr, 1, oldProtect, &ignored)
	_ = flushInstructionCache(handle, addr, 1)
	if err != nil {
		return err
	}
	if written != 1 {
		return fmt.Errorf("short write at 0x%x", addr)
	}
	return nil
}

func debugActiveProcess(pid uint32) error {
	r1, _, e1 := procDebugActiveProcess.Call(uintptr(pid))
	if r1 == 0 {
		return syscallError(e1)
	}
	return nil
}

func debugActiveProcessStop(pid uint32) error {
	r1, _, e1 := procDebugActiveProcessStop.Call(uintptr(pid))
	if r1 == 0 {
		return syscallError(e1)
	}
	return nil
}

func debugSetProcessKillOnExit(kill bool) error {
	v := uintptr(0)
	if kill {
		v = 1
	}
	r1, _, e1 := procDebugSetProcessKillOnExit.Call(v)
	if r1 == 0 {
		return syscallError(e1)
	}
	return nil
}

func waitForDebugEvent(ev *debugEvent, timeoutMS uint32) error {
	r1, _, e1 := procWaitForDebugEvent.Call(uintptr(unsafe.Pointer(ev)), uintptr(timeoutMS))
	if r1 == 0 {
		err := syscallError(e1)
		if errors.Is(err, syswin.ERROR_SEM_TIMEOUT) || errors.Is(err, syswin.WAIT_TIMEOUT) {
			return errDebugWaitTimeout
		}
		return err
	}
	return nil
}

func continueDebugEvent(pid, tid, status uint32) error {
	r1, _, e1 := procContinueDebugEvent.Call(uintptr(pid), uintptr(tid), uintptr(status))
	if r1 == 0 {
		return syscallError(e1)
	}
	return nil
}

func getThreadContext(thread syswin.Handle, ctx *amd64Context) error {
	r1, _, e1 := procGetThreadContext.Call(uintptr(thread), uintptr(unsafe.Pointer(ctx)))
	if r1 == 0 {
		return syscallError(e1)
	}
	return nil
}

func setThreadContext(thread syswin.Handle, ctx *amd64Context) error {
	r1, _, e1 := procSetThreadContext.Call(uintptr(thread), uintptr(unsafe.Pointer(ctx)))
	if r1 == 0 {
		return syscallError(e1)
	}
	return nil
}

func flushInstructionCache(handle syswin.Handle, addr uintptr, size uintptr) error {
	r1, _, e1 := procFlushInstructionCache.Call(uintptr(handle), addr, size)
	if r1 == 0 {
		return syscallError(e1)
	}
	return nil
}

func syscallError(err error) error {
	if err != nil && err != syscall.Errno(0) {
		return err
	}
	return syscall.EINVAL
}
