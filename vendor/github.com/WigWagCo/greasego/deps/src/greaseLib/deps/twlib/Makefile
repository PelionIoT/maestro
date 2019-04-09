
LDFLAGS ?= -lpthread -pthread
TWLIBFLAG ?= -L. -lTW
TWLIBNAME ?= libTW
TWSOLIBNAME ?= $(TWLIBNAME).so
TWSONAME ?= $(TWSOLIBNAME).1.0.1
TWSOVERSION ?= $(TWSOLIBNAME).1
TWSTATICNAME ?= $(TWLIBNAME).a
# TWSOVERSION is the compiler version...
# see http://rute.2038bug.com/node26.html.gz

CXX ?= g++ -g -O0 -fPIC
CC ?= gcc -g -O0 -fPIC
AR ?= ar

ARCH ?=x86
#ARCH=armel
SYSCALLS= syscalls-$(ARCH).c

ALLOBJS= $($<:%.cpp=%.o)

DEBUG_OPTIONS=-rdynamic -D_TW_TASK_DEBUG_THREADS_ 
#-D_TW_BUFBLK_DEBUG_STACK_
CFLAGS= $(DEBUG_OPTIONS) $(GLIBCFLAG) -D_TW_DEBUG -I./include  -D__DEBUG   -fPIC -std=c++11

CROSS_PREREQ_LIBS=freescale.out/expanded-prereqs/lib
CROSS_PREREQ_HEADERS=freescale.out/expanded-prereqs/include

DEBUG_CFLAGS= -g

ROOT_DIR=.
OUTPUT_DIR=.


EXTRA_TARGET=

ifdef FREESCALE
	EXTRA_TARGET+=freescale_dir
	TARGET_ARCH=-march=armv5te 
#"-march=armv7-a" # must be at least armv5te
#export TARGET_TUNE="-mtune=cortex-a8 -mfpu=neon -mfloat-abi=softfp -mthumb-interwork -mno-thumb" # optional
	TARGET_TUNE=-mtune=arm926ej-s -mfloat-abi=soft
	TOOL_PREFIX=arm-fsl-linux-gnueabi
	CSTOOLS_LIB=/opt/ltib/rootfs/lib
	CROSS_CC_BASE=/opt/freescale/usr/local/gcc-4.4.4-glibc-2.11.1-multilib-1.0/arm-fsl-linux-gnueabi
	CROSS_INCLUDE=$(CROSS_CC_BASE)/arm-fsl-linux-gnueabi/multi-libs/usr/include/
	#/opt/freescale/usr/local/gcc-4.4.4-glibc-2.11.1-multilib-1.0/arm-fsl-linux-gnueabi
	CROSS_CC=$(CROSS_CC_BASE)/bin/$(TOOL_PREFIX)-gcc
	CROSS_CXX=$(CROSS_CC_BASE)/bin/$(TOOL_PREFIX)-g++
	CROSS_AR=$(CROSS_CC_BASE)/bin/$(TOOL_PREFIX)-ar
#	@echo Using GCC toolchain for Freescale i.MX28: $(CROSS_CC)
	CC= $(CROSS_CC) -g -O0 -fPIC -D__ZDB_ARM__ $(TARGET_ARCH) $(TARGET_TUNE) -I$(CROSS_INCLUDE) -I$(CROSS_PREREQ_HEADERS)
	CXX= $(CROSS_CXX) -g -O0 -fPIC -D__ZDB_ARM__ $(TARGET_ARCH) $(TARGET_TUNE) -I$(CROSS_INCLUDE) -I$(CROSS_PREREQ_HEADERS)
	AR=$(CROSS_AR)
	LD_FLAGS+= -L$(CROSS_CC_BASE)/multi-libs/armv5te/usr/lib -L$(CROSS_PREREQ_LIBS)  -Wl,-rpath-link,$(CSTOOLS_LIB) -Wl,-O1 -Wl,--hash-style=gnu 
	CFLAGS+= -Lfreescale.out/expanded-prereqs/lib
	OUTPUT_DIR=freescale.out
else
	CFLAGS+= -Lexpanded-prereqs/lib -Iexpanded-prereqs/include  -fPIC $(DEBUG_CFLAGS)
endif

GLIBCFLAG=-D_USING_GLIBC_
LD_TEST_FLAGS= -lgtest

HRDS= include/TW/tw_bufblk.h  include/TW/tw_globals.h  include/TW/tw_object.h include/TW/tw_stack.h\
include/TW/tw_dlist.h   include/TW/tw_llist.h    include/TW/tw_socktask.h    include/TW/tw_syscalls.h\
include/TW/tw_macros.h include/TW/tw_globals.h include/TW/tw_alloc.h \
include/TW/tw_stringmap.h

#include/TW/tw_sparsehash.h include/TW/tw_densehash.h\

SRCS_CPP= tw_object.cpp tw_globals.cpp tw_socktask.cpp tw_globals.cpp tw_log.cpp tw_alloc.cpp tw_utils.cpp tw_task.cpp tw_stringmap.cpp

SRCS_C= $(SYSCALLS)
OBJS= $(SRCS_CPP:%.cpp=$(OUTPUT_DIR)/%.o) $(SRCS_C:%.c=$(OUTPUT_DIR)/%.o)
OBJS_NAMES= $(SRCS_CPP:%.cpp=$%.o) $(SRCS_C:%.c=%.o)
TPLS= include/TW/tw_fifo.h include/TW/tw_task.h include/TW/tw_alloc.h include/TW/tw_bufblk.h 

#include/TW/tw_sparsehash.h include/TW/tw_densehash.h

##tw_sparsehash.h

## The -fPIC option tells gcc to create position 
## independant code which is necessary for shared libraries. Note also, 
## that the object file created for the static library will be 
## overwritten. That's not bad, however, because we have a static 
## library that already contains the needed object file.

$(OUTPUT_DIR)/%.o: %.cpp
	$(CXX) $(CFLAGS) -c $< -o $@

$(OUTPUT_DIR)/%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

ztests: testzstrings zeventtest testzmonitornstore testzsparsehash testprotoparser testzdbstore

ZSTRING_SRC = zstring.cpp dsrnstring.cpp
ZSTRING_HDR = $(ZSTRING_SRC:%.cpp=%.h) protocol.h
ZSTRING_OBJ = $(ZSTRING_SRC:%.cpp=%.o)  $(SYSCALLS:%.c=%.o)
testzstring: testzstrings testzstrings2 testzstringmem $(ZSTRING_OBJ) $(ZSTRING_HDR)


tw_lib: $(OBJS) $(HDRS) $(TPLS) $(EXTRA_TARGET)
	$(CXX) $(CFLAGS) -I. $(LDFLAGS) -shared -Wl,-soname,$(TWSOVERSION) -o $(OUTPUT_DIR)/$(TWSONAME) $(OBJS) $(TPLS)
	ln -sf $(TWSONAME) $(OUTPUT_DIR)/$(TWSOVERSION) && \
		ln -sf $(TWSONAME) $(OUTPUT_DIR)/$(TWSOLIBNAME)
	$(AR) rcs $(OUTPUT_DIR)/$(TWSTATICNAME) $(OBJS)  # build static library


test_fifo: tests/test_fifo.cpp $(TPLS)
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. -o $@ tests/$@.cpp $(TPLS) 

test_fifo_task: tw_lib tests/test_fifo_task.cpp $(TPLS)
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -I. -o $@ tests/$@.cpp $(TPLS) 

test_fifo_bufs: tw_lib tests/test_fifo_bufs.cpp $(TPLS) tw_log.o 
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS)  -I. -o $@ tests/$@.cpp tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_tw_sema_basetask: tw_lib tests/test_tw_sema_basetask.cpp $(TPLS) tw_log.o 
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS)  -I. -o $@ tests/$@.cpp tw_log.o syscalls-$(ARCH).o $(TPLS) 

freescale_dir:
	-mkdir -p freescale.out
 
.PHONY: freescale_dir

testtwcontainers: tw_lib tests/testtwcontainers.cpp $(TPLS) tw_log.o 
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS)  -I. -o $@ tests/$@.cpp tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_tw_sema: tw_lib tests/test_tw_sema.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS)  -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_twcircular: tw_lib tests/test_twcircular.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -g -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o 

test_twcircular_unblockall: tw_lib tests/test_twcircular_unblockall.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -g -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_twcircular_array: tw_lib tests/test_twcircular_array.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -g -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_twcircular_array_transfer: tw_lib tests/test_twcircular_array_transfer.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -g -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_twcircular_mv: tw_lib tests/test_twcircular_mv.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -g -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_twcircular_mv_noblock: tw_lib tests/test_twcircular_mv_noblock.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -g -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_twcircular_mv_timeout: tw_lib tests/test_twcircular_mv_timeout.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -g -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_twcircular_slow_consumer: tw_lib tests/test_twcircular_slow_consumer.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS) -g -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

test_tw_bndsafefifo: tw_lib tests/test_tw_bndsafefifo.cpp $(TPLS) tw_log.o tw_utils.o
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LDFLAGS)  -I. -o $@ tests/$@.cpp tw_utils.o tw_log.o syscalls-$(ARCH).o $(TPLS) 

regr_tw_bufblk: tw_lib tests/regr_tw_bufblk.cpp $(TPLS) tw_log.o tests/testutils.cpp
	$(CXX) $(CFLAGS) -c -I. tests/testutils.cpp
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LD_TEST_FLAGS) $(LDFLAGS)  -I. -o $@ tests/$@.cpp tw_log.o testutils.o syscalls-$(ARCH).o $(TPLS) 

test_list: tw_lib tests/test_list.cpp $(TPLS) tw_log.o tests/testutils.cpp
	$(CXX) $(CFLAGS) -c -I. tests/testutils.cpp
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LD_TEST_FLAGS) $(LDFLAGS)  -I. -o $@ tests/$@.cpp tw_log.o testutils.o syscalls-$(ARCH).o $(TPLS) 

test_twarray: tw_lib tests/test_twarray.cpp $(TPLS) tw_log.o tests/testutils.cpp
	$(CXX) $(CFLAGS) -c -I. tests/testutils.cpp
	$(CXX) $(CFLAGS) $(TWLIBFLAG) $(LD_TEST_FLAGS) $(LDFLAGS)  -I. -o $@ tests/$@.cpp tw_log.o testutils.o syscalls-$(ARCH).o $(TPLS) 

test_log: tests/test_log.cpp tw_log.o syscalls-$(ARCH).o include/TW/tw_log.h
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. -o $@ tests/test_log.cpp tw_log.o syscalls-$(ARCH).o 

test_alloc: tests/test_alloc.cpp tw_log.o syscalls-$(ARCH).o $(HDRS) tw_alloc.o
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. -o $@ tests/test_alloc.cpp tw_log.o syscalls-$(ARCH).o tw_alloc.o 

test_sparsehash: tests/test_sparsehash.cpp tw_log.o tw_alloc.o syscalls-$(ARCH).o $(HDRS)
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. -o $@ tests/test_sparsehash.cpp tw_log.o syscalls-$(ARCH).o tw_alloc.o  

test_densehash: tests/test_densehash.cpp tw_log.o tw_alloc.o syscalls-$(ARCH).o $(HDRS)
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. -o $@ tests/test_densehash.cpp tw_log.o syscalls-$(ARCH).o tw_alloc.o  

test_autopointer: tests/autopointertest.cpp tw_log.o syscalls-$(ARCH).o include/TW/tw_log.h
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. -o $@ tests/autopointertest.cpp tw_log.o syscalls-$(ARCH).o 

test_simple_khash: tests/simple_khashtest.c 
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. -o $@ tests/simple_khashtest.c tw_log.o syscalls-$(ARCH).o

test_khash: tests/test_khashtest.cpp include/TW/tw_khash.h include/TW/khash.h
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. $(TWLIBFLAG) -o $@ tests/test_khashtest.cpp tw_log.o syscalls-$(ARCH).o

test_rbtree: tw_lib tests/test_rbtree.cpp include/TW/tw_rbtree.h include/TW/provos_rb_tree.h include/TW/tw_khash.h include/TW/khash.h
	$(CXX) $(CFLAGS) $(LDFLAGS) -I. $(TWLIBFLAG) -o $@ tests/test_rbtree.cpp tw_log.o syscalls-$(ARCH).o

test_hashes: tw_lib tests/test_hashes.cpp include/TW/tw_khash.h include/TW/khash.h
	$(CXX) $(CFLAGS) $(LDFLAGS) $(LD_TEST_FLAGS) -I. $(TWLIBFLAG) -o $@ tests/test_hashes.cpp tw_log.o syscalls-$(ARCH).o

install: tw_lib $(EXTRA_TARGET)
	./install-sh $(TWSOVERSION) $(INSTALLPREFIX)
	ln -sf $(INSTALLPREFIX)/lib/$(TWSONAME) $(INSTALLPREFIX)/lib/$(TWSOVERSION) && \
	ln -sf $(INSTALLPREFIX)/lib/$(TWSONAME) $(INSTALLPREFIX)/lib/$(TWSOLIBNAME)





clean: 
	-rm -rf $(OUTPUT_DIR)/*.o $(OUTPUT_DIR)/*.obj $(OUTPUT_DIR)/*.rpo $(OUTPUT_DIR)/*.idb $(OUTPUT_DIR)/*.lib $(OUTPUT_DIR)/*.exe $(OUTPUT_DIR)/*~ $(OUTPUT_DIR)/core
	-rm -rf Debug
	-rm -f $(TWSOLIBNAME) $(TWSONAME) $(TWSOVERSION)
# DO NOT DELETE

