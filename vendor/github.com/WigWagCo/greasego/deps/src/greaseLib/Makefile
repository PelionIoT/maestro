
#LDFLAGS ?= -lpthread -pthread 
# TWSOVERSION is the compiler version...
# see http://rute.2038bug.com/node26.html.gz

CXX ?= g++ -g -O0 -fPIC -std=c++11
CC ?= gcc -g -O0 -fPIC
AR ?= ar

LIBUVDIR= libuv-v1.10.1

CXXFLAGS= -fPIC -std=c++11

ARCH ?=x86 
#ARCH=armel	
SYSCALLS= syscalls-$(ARCH).c

ALLOBJS= $($<:%.cpp=%.o)

DEBUG_OPTIONS=-rdynamic -D_TW_TASK_DEBUG_THREADS_ -DLOGGER_HEAVY_DEBUG  -D__DEBUG -D_TW_DEBUG 
#-D_TW_BUFBLK_DEBUG_STACK_
CFLAGS= $(GLIBCFLAG) -I./include   -fPIC -I./deps/$(LIBUVDIR)/include -I./deps/build/include  -L./deps/build/lib -DGREASE_LIB  -Wno-error=format-security

DEBUG_CFLAGS= -g -DERRCMN_DEBUG_BUILD $(DEBUG_OPTIONS)

ROOT_DIR=.
OUTPUT_DIR=.



EXTRA_TARGET=

CFLAGS+= -fPIC

GLIBCFLAG=-D_USING_GLIBC_
LD_TEST_FLAGS= -lgtest

## concerning the -whole-archive flags: http://stackoverflow.com/questions/14889941/link-a-static-library-to-a-shared-one-during-build
## originally we used that when creating the node module version of greaseLogger - but apparently needed for tcmalloc here also
# -luv -lTW
LDFLAGS += -L./deps/build/lib  -ldl -lm -ltcmalloc_minimal -lstacktrace -lstdc++ -lpthread -pthread
# -Wl,-whole-archive deps/build/lib/libtcmalloc_minimal.a -Wl,-no-whole-archive 
STATIC_LIB_FLAGS= -ldl -lTW -lm -Wl,-whole-archive deps/build/lib/libtcmalloc_minimal.a -Wl,-no-whole-archive

HRDS= include/TW/tw_bufblk.h  include/TW/tw_globals.h  include/TW/tw_object.h include/TW/tw_stack.h\
include/TW/tw_dlist.h   include/TW/tw_llist.h    include/TW/tw_socktask.h    include/TW/tw_syscalls.h\
include/TW/tw_macros.h include/TW/tw_globals.h include/TW/tw_alloc.h include/TW/tw_sparsehash.h include/TW/tw_densehash.h\
include/TW/tw_stringmap.h

SRCS_CPP= error-common.cc logger.cc standalone_test_logsink.cc grease_lib.cc

SRCS_C= grease_client.c local_strdup.c
OBJS= $(SRCS_CPP:%.cc=$(OUTPUT_DIR)/%.o) $(SRCS_C:%.c=$(OUTPUT_DIR)/%.o)
OBJS_NAMES= $(SRCS_CPP:%.cc=$%.o) $(SRCS_C:%.c=%.o)

##tw_sparsehash.h

## The -fPIC option tells gcc to create position 
## independant code which is necessary for shared libraries. Note also, 
## that the object file created for the static library will be 
## overwritten. That's not bad, however, because we have a static 
## library that already contains the needed object file.


ifdef DEBUG
CFLAGS += $(DEBUG_CFLAGS)
endif

$(OUTPUT_DIR)/%.o: %.cc
	$(CXX) $(CXXFLAGS) $(CFLAGS) -c $< -o $@

$(OUTPUT_DIR)/%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

.PHONY: getversion

getversion:
	./update_version.sh

# -Wl,-z,defs 
libgrease.so.1: CFLAGS += -shared -Wl,-soname,libgrease.so.1  -Wl,--no-as-needed -ldl
libgrease.so.1: getversion $(OBJS)
	$(CXX) $(CFLAGS)  -o $@ $(OBJS) deps/build/lib/libuv.a deps/build/lib/libTW.a $(LDFLAGS) 

# -g -O0
# -DERRCMN_DEBUG_BUILD  -Wl,--whole-archive -ldl -Wl,--no-whole-archive -Wl,--no-undefined -Wl,-Bstatic -ldl -Wl,-Bdynamic -Wl,--whole-archive -Wl,--no-whole-archive
#  -Wl,-z,defs (to test for missing defs)
libgrease.so.1-debug: CFLAGS += -DERRCMN_DEBUG_BUILD -g -O0 -shared -Wl,-soname,libgrease.so.1 -Wl,--no-as-needed -ldl
libgrease.so.1-debug: getversion $(OBJS)
	$(CXX) $(CFLAGS) -o $@ $(OBJS) deps/build/lib/libuv.a deps/build/lib/libTW.a $(LDFLAGS)  


libgrease.a: CFLAGS += -static -DGREASE_LIB
libgrease.a: getversion $(OBJS)
	$(AR) rcs $@ $(OBJS)

libgrease.a-server-debug: CFLAGS += -DERRCMN_DEBUG_BUILD -g -O0 -DGREASE_IS_LOCAL -static -DGREASE_LIB 
libgrease.a-server-debug: getversion $(OBJS) deps/build/lib/libuv.a deps/build/lib/libTW.a
	$(AR) rcs libgrease.a $(OBJS)

libgrease.a-server: CFLAGS += -DGREASE_IS_LOCAL -static -DGREASE_LIB 
libgrease.a-server: getversion $(OBJS)
	$(AR) rcs libgrease.a $(OBJS) 


standalone_test_logsink: CFLAGS+= -DGREASE_LIB -I./deps/$(LIBUVDIR)/include -I./deps/twlib/include
standalone_test_logsink: getversion standalone_test_logsink.o libgrease.a
	$(CXX) $(CXXFLAGS) $(CFLAGS) libgrease.a deps/build/lib/libuv.a deps/build/lib/libTW.a $(LDFLAGS) -lpthread -pthread  -o $@ 

grease_echo: CFLAGS+= -DGREASE_LIB -I./deps/$(LIBUVDIR)/include -I./deps/twlib/include
grease_echo: $(OUTPUT_DIR)/grease_client.o $(OUTPUT_DIR)/grease_echo.o
	$(CXX) $(CXXFLAGS) $(CFLAGS) $^ -ldl -o $@ 


install: tw_lib $(EXTRA_TARGET)
	./install-sh $(TWSOVERSION) $(INSTALLPREFIX)
	ln -sf $(INSTALLPREFIX)/lib/$(TWSONAME) $(INSTALLPREFIX)/lib/$(TWSOVERSION) && \
	ln -sf $(INSTALLPREFIX)/lib/$(TWSONAME) $(INSTALLPREFIX)/lib/$(TWSOLIBNAME)

all: grease_echo libgrease.a-server libgrease.so.1


clean: 
	-rm -rf $(OUTPUT_DIR)/*.o $(OUTPUT_DIR)/*.obj $(OUTPUT_DIR)/*.rpo $(OUTPUT_DIR)/*.idb $(OUTPUT_DIR)/*.lib $(OUTPUT_DIR)/*.exe $(OUTPUT_DIR)/*.a $(OUTPUT_DIR)/*~ $(OUTPUT_DIR)/core
	-rm -rf Debug
	-rm -f $(TWSOLIBNAME) $(TWSONAME) $(TWSOVERSION)
# DO NOT DELETE
