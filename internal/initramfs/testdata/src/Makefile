# SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
#
# SPDX-License-Identifier: MIT

.SECONDEXPANSION:

CC ?= clang
STRIP ?= strip

BIN_DIR ?= ../bin
LIB_DIR ?= ../lib

CFLAGS = -Wall -Werror -fPIC -L$(LIB_DIR) $(addprefix -l,$(LIBS))
# no stdlib needed as we have our won _start function in start.S
CFLAGS += -nostdlib
# reduce size
CFLAGS += -Wl,-z,max-page-size=0x1000,-z,norelro

STRIPFLAGS ?= -s

# Set target specific libs as prerequisites using SECONDEXPANSION after LIBS is
# set.
LIBS_PREREQ := $$(patsubst %,$$(LIB_DIR)/lib%.so,$$(LIBS))

export LD_LIBRARY_PATH = $(LIB_DIR)

.PHONY: test
test: $(BIN_DIR)/main
	# main is supposed to return 0111.
	$(BIN_DIR)/main; [ $$? -eq 73 ]

# Compile custom _start function in start.S so we don't have any external
# library dependencies and can use -nostdlib flag.
$(BIN_DIR)/main: LIBS = func2 func3
$(BIN_DIR)/main: main.c start.S $(LIBS_PREREQ)
	mkdir -p $(@D)
	$(CC) $(CFLAGS) -o $@ $< start.S
	$(STRIP) $(STRIPFLAGS) $@

$(LIB_DIR)/libfunc3.so: LIBS = func1
$(LIB_DIR)/libfunc3.so: $(LIBS_PREREQ)
$(LIB_DIR)/lib%.so: LIBS =
$(LIB_DIR)/lib%.so: %.o
	mkdir -p $(@D)
	$(CC) -shared $(CFLAGS) -o $@ $<
	$(STRIP) $(STRIPFLAGS) $@

.PHONY: clean
clean:
	rm -fvr $(BIN_DIR) $(LIB_DIR)
