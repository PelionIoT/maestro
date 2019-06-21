/*
 * GreaseLogger.h
 *
 *  Created on: Aug 27, 2014
 *      Author: ed
 * (c) 2016, WigWag Inc
 */
/*
    MIT License

    Copyright (c) 2019, Arm Limited and affiliates.

    SPDX-License-Identifier: MIT
    
    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.
*/

#ifndef GreaseLogger_H_
#define GreaseLogger_H_

#ifndef GREASE_LIB
#include <v8.h>
#include <node.h>
#include <uv.h>
#include <node_buffer.h>

#include "nan.h"

using namespace node;
using namespace v8;

#else
#include "grease_lib.h"
#include <assert.h>
#endif

#include <sys/types.h>
#include <sys/socket.h>
// #include <linux/if.h>
// #include <linux/if_tun.h>
#ifndef __APPLE__
#include <linux/fs.h>
#endif
// ok in both Linux and OS X:
#include <sys/types.h>  // for OS X writev()
#include <sys/uio.h>    // Linux writev()

#include <syslog.h>  // for LOG_FAC and LOG_PRI macros

#include <unistd.h>
#include <sys/ioctl.h>
#include <sys/stat.h>
#include <fcntl.h>

#include <string.h>
#include <stdlib.h>
#include <stdarg.h>
#include <stdio.h>
#include <uv.h>
#include <time.h>
// Linux thing:
#include <sys/timeb.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/select.h>
// Linux kernel logs:
#include <sys/klog.h>
//#include <sys/syslog.h>


#include <TW/tw_alloc.h>
#include <TW/tw_fifo.h>
#include <TW/tw_khash.h>
#include <TW/tw_circular.h>

#include "grease_client.h"
#include "error-common.h"

#include <gperftools/tcmalloc.h>

//#include <re2/re2.h>
#include <string>
//#define PCRE2_CODE_UNIT_WIDTH 8
//#include <pcre2.h>



#define LMALLOC tc_malloc_skip_new_handler
#define LCALLOC tc_calloc
#define LREALLOC tc_realloc
#define LFREE tc_free

extern "C" char *local_strdup_safe(const char *s);
extern "C" int memcpy_and_json_escape(char *out, const char *in, int in_len, int *out_len);

namespace Grease {

//typedef void (*__GreaseLibCallback) (_errcmn::err_ev *err, void *); // used for internal callbacks in the library, when not using V8

using namespace TWlib;

struct Alloc_LMALLOC {
	static void *malloc (tw_size nbytes) {  return LMALLOC((int) nbytes); }
	static void *calloc (tw_size nelem, tw_size elemsize) { return LCALLOC((size_t) nelem,(size_t) elemsize); }
	static void *realloc(void *d, tw_size s) { return LREALLOC(d,(size_t) s); }
	static void free(void *p) { LFREE(p); }
	static void sync(void *addr, tw_size len, int flags = 0) { } // does nothing - not shared memory
	static void *memcpy(void *d, const void *s, size_t n) { return ::memcpy(d,s,n); };
	static void *memmove(void *d, const void *s, size_t n) { return ::memmove(d,s,n); };
	static int memcmp(void *l, void *r, size_t n) { return ::memcmp(l, r, n); }
	static void *memset(void *d, int c, size_t n) { return ::memset(d,c,n); }
	static const char *ALLOC_NOMEM_ERROR_MESSAGE;
};


typedef TWlib::Allocator<Alloc_LMALLOC> LoggerAlloc;


struct uint32_t_eqstrP {
	  inline int operator() (const uint32_t *l, const uint32_t *r) const
	  {
		  return (*l == *r);
	  }
};

struct TargetId_eqstrP {
	  inline int operator() (const TargetId *l, const TargetId *r) const
	  {
		  return (*l == *r);
	  }
};

struct uint64_t_eqstrP {
	  inline int operator() (const uint64_t *l, const uint64_t *r) const
	  {
		  return (*l == *r);
	  }
};

// http://broken.build/2012/11/10/magical-container_of-macro/
// http://www.kroah.com/log/linux/container_of.html
#ifndef offsetof
	#define offsetof(TYPE, MEMBER) ((size_t) &((TYPE *)0)->MEMBER)
#endif
#ifndef container_of
#define container_of(ptr, type, member) ({ \
                const typeof( ((type *)0)->member ) *__mptr = (ptr); \
                (type *)( (char *)__mptr - offsetof(type,member) );})
#endif

#if UV_VERSION_MAJOR < 1
#define ERROR_UV(msg, code, loop) do {                                              \
  uv_err_t __err = uv_last_error(loop);                                             \
  fprintf(stderr, "%s: [%s: %s]\n", msg, uv_err_name(__err), uv_strerror(__err));   \
} while(0)
#else
#define ERROR_UV(msg, code, loop) do {                                              \
		  fprintf(stderr, "%s: [%s: %s]\n", msg, uv_err_name(code), uv_strerror(code));                \
		} while(0)
#endif


//  assert(0);

#define COMMAND_QUEUE_NODE_SIZE 200
#define INTERNAL_QUEUE_SIZE 200          // must be at least as big as PRIMARY_BUFFER_ENTRIES
#define V8_LOG_CALLBACK_QUEUE_SIZE 10
#define MAX_TARGET_CALLBACK_STACK 20
#define TARGET_CALLBACK_QUEUE_WAIT 2000000  // 2 seconds
#define MAX_ROTATED_FILES 10


#define DEFAULT_TARGET GREASE_DEFAULT_TARGET_ID
#define DEFAULT_ID 0

#define GREASE_ALL_LEVELS 0xFFFFFFFF
#define GREASE_STDOUT 1
#define GREASE_STDERR 2
#define GREASE_BAD_FD -1

#define TTY_NORMAL 0
#define TTY_RAW    1

#define NOTREADABLE 0
#define READABLE    1

#define META_GET_LIST(m,n) ((FilterList *)m._cached_lists[n])
#define META_SET_LIST(m,n,p) do { m._cached_lists[n] = p; } while(0)

// use these to fix "NO BUFFERS" ... greaseLog will never block if it's IO can't keep up with logging
// and instead will drop log info
#define NUM_BANKS 4  // default number of banks in a logTarget
#define LOGGER_DEFAULT_CHUNK_SIZE  1500
#define DEFAULT_BUFFER_SIZE  2000

#define PRIMARY_BUFFER_ENTRIES 100   // this is the amount of entries we hold in memory, which will be logged by the logger thread.
                                     // each entry is DEFAULT_BUFFER_SIZE
                                     // if a log message is > than DEFAULT_BUFFER_SIZE, it logged as an overflow.
#define PRIMARY_BUFFER_SIZE (PRIMARY_BUFFER_ENTRIES*DEFAULT_BUFFER_SIZE)
// Sink settings
#define SINK_BUFFER_SIZE (DEFAULT_BUFFER_SIZE*2)
#define BUFFERS_PER_SINK 4

// this number can be bigger, but why?
// anything larger than this will just be dropped. Prevents a buggy program
// from chewing up gobs of log memory.
#define MAX_LOG_MESSAGE_SIZE 65536



#define DEFAULT_MODE_FILE_TARGET (S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP)
#define DEFAULT_FLAGS_FILE_TARGET (O_APPEND | O_CREAT | O_WRONLY)

//#define LOGGER_HEAVY_DEBUG 1
#define MAX_IDENTICAL_FILTERS 16
#define LOGGER_HEAVY_DEBUG
#ifdef LOGGER_HEAVY_DEBUG
#pragma message "Build is Debug Heavy!!"
// confused? here: https://gcc.gnu.org/onlinedocs/cpp/Variadic-Macros.html
#define HEAVY_DBG_OUT(s,...) fprintf(stderr, "**DEBUG2** " s "\n", ##__VA_ARGS__ )
#define IF_HEAVY_DBG( x ) { x }
#else
#define HEAVY_DBG_OUT(s,...) {}
#define IF_DBG( x ) {}
#endif

const int MAX_IF_NAME_LEN = 16;


class GreaseLogger 
#ifndef GREASE_LIB
	: public Nan::ObjectWrap 
#endif
{
public:
	typedef void (*actionCB)(GreaseLogger *, _errcmn::err_ev &err, void *data);

//	struct start_info {
//		Handle<Function> callback;
//		start_info() : callback() {}
//	};

	class heapBuf final {
	public:
		class heapBufManager {
		public:
			virtual void returnBuffer(heapBuf *b) = 0;
		};
		heapBufManager *return_cb;
		uv_buf_t handle;
		int used;
//		uv_buf_t getUvBuf() {
//			return uv_buf_init(handle.base,handle.len);
//		}
		explicit heapBuf(int n) : return_cb(NULL), used(0) {
			handle.len = n;
			if(n > 0)
				handle.base = (char *) LMALLOC(n);
			else
				handle.base = NULL;
		}
		void sprintf(const char *format, ... ) {
			char buffer[512];
			va_list args;
			va_start (args, format);
			RawLogLen len = (RawLogLen) vsnprintf (buffer,512,format, args);
			va_end (args);
			if(handle.base) LFREE(handle.base);
			handle.len = len;
			handle.base = (char *) LMALLOC(handle.len);
			::memset((void *)handle.base,(int) 0,handle.len);
			::memcpy(handle.base,buffer,len);
			used = handle.len;
		}
		heapBuf(const char *d, int n) : return_cb(NULL), used(0) {
			handle.len = n;
			handle.base = (char *) LMALLOC(n);
			::memcpy(handle.base,d,n);
		}
		heapBuf(heapBuf &&o) {
			handle = o.handle;
			used = o.used; o.used =0;
			o.handle.base = NULL;
			o.handle.len = 0;
			return_cb = o.return_cb; o.return_cb = NULL;
		}
		void malloc(int n) {
			if(handle.base) LFREE(handle.base);
			handle.len = n;
			handle.base = (char *) LMALLOC(handle.len);
			used = 0;
		}

		int memcpy(const char *s, size_t l,const char *append_str=nullptr) {
			if(handle.base) {
				if(append_str) {
					bool append = false;
					int app_len = 0;
					if(l > handle.len) {
						app_len = strlen(append_str);
						l = (int) handle.len-app_len;
						append = true;
						::memcpy((char*)handle.base+l,append_str,app_len);
					}
					::memcpy(handle.base,s,l);
					used = l + app_len;
					return l;
				} else {
					if(l > handle.len) l = (int) handle.len;
					::memcpy(handle.base,s,l);
					used = l;
					return l;
				}
			} else
				return 0;
		}
		int memcpyJsonEscape(const char *s, size_t l,const char *append_str=nullptr) {
			if(handle.base) {
				if(append_str) {
					bool append = false;
					int app_len = 0;
					if(l > handle.len) {
						app_len = strlen(append_str);
						l = (int) handle.len-app_len;
						append = true;
						::memcpy((char*)handle.base+l,append_str,app_len);
					}
					int out_len = handle.len;
					memcpy_and_json_escape(handle.base, s, l, &out_len);
					//					::memcpy(handle.base,s,l);
					used = l + app_len;
					return l;
				} else {
					if(l > handle.len) l = (int) handle.len;
//					::memcpy(handle.base,s,l);
					int out_len = handle.len;
					memcpy_and_json_escape(handle.base, s, l, &out_len);
					used = out_len;
					return l;
				}
			} else
				return 0;
		}
		heapBuf duplicate() {
			heapBuf ret;
			if(handle.base) {
				ret.handle.base = (char *) LMALLOC(handle.len);
				ret.handle.len = handle.len;
				::memcpy(ret.handle.base,handle.base,handle.len);
				ret.used = used;
			}
			ret.return_cb = return_cb;
			return ret;
		}
		bool empty() {
			if(handle.base != NULL) return false; else return true;
		}
		void returnBuf() {
			if(return_cb) return_cb->returnBuffer(this);
		}
		explicit heapBuf() : return_cb(NULL) { handle.base = NULL; handle.len = 0; };
		~heapBuf() {
//			DBG_OUT(" debug FREE %x\n",handle.base);
			if(handle.base) LFREE(handle.base);
		}
		// I find it too confusing to having multiple overloaded '=' operators - so we have this
		void assign(heapBuf &o) {
			returnBuf();
			if(handle.base) LFREE(handle.base);
			handle.base = NULL;
			if(o.handle.base && o.handle.len > 0) {
				handle.base = (char *) LMALLOC(o.handle.len);
				handle.len = o.handle.len;
				::memcpy(handle.base,o.handle.base,handle.len);
			}
		}
		heapBuf& operator=(heapBuf&& o) {
			used = o.used; o.used =0;
			handle = o.handle;
			o.handle.base = NULL;
			o.handle.len = 0;
			return_cb = o.return_cb; o.return_cb = NULL;
			return *this;
		}
	};

	class singleLog final {
	private:
		int _ref_cnt;
	protected:
		singleLog(int len, const logMeta &m) : _ref_cnt(NOT_REFED), meta(), buf(len) {
			meta.m = m;
		}
	public:
		extra_logMeta meta;
		heapBuf buf;
		static const int NOT_REFED = -2;  // no reference counting
		static const int INIT_REF = 1;  // no reference counting
		singleLog(const char *d, int l, const logMeta &_m) : _ref_cnt(NOT_REFED), meta(), buf(d,l) {
			meta.m = _m;
		}
		singleLog(int len) : _ref_cnt(NOT_REFED), meta(), buf(len) {
			ZERO_LOGMETA(meta.m);
		}
		singleLog() = delete;
		static singleLog *newSingleLogAsJsonEscape(const char *d, int l, const logMeta &m) {
			singleLog *ret = new singleLog(l*2,m);
			ret->buf.memcpyJsonEscape(d,l);
			return ret;
		}

		static singleLog *heapSingleLog(int len) {
			singleLog *ret = new singleLog(len);
			ret->_ref_cnt = INIT_REF;
			return ret;
		}
		static singleLog *heapSingleLog(const char *d, int l, const logMeta &_m) {
			singleLog *ret = new singleLog(d,l,_m);
			ret->_ref_cnt = INIT_REF;
			return ret;
		}
		void clear() {
			buf.used = 0;
			_ref_cnt = 1;
			ZERO_LOGMETA(meta.m);
		}
		void incRef() {
			if(_ref_cnt != NOT_REFED) {
				DBG_OUT("incRef: %p (%d)\n", this, _ref_cnt);
				_ref_cnt++;
			}
		}
		void decRef() {
			if(_ref_cnt != NOT_REFED) {
				DBG_OUT("decRef: %p (%d)\n", this, _ref_cnt);
				_ref_cnt--;
				if(_ref_cnt < 1) {
					DBG_OUT("ref ... delete: %p (%d)\n", this, _ref_cnt);
					delete this;
				}
			}
		}
		int getRef() {
			return _ref_cnt;
		}
	};

	static const int MAX_OVERFLOW_BUFFERS = 4;

	class overflowWriteOut final {
	public:
		int N;
		singleLog *bufs[MAX_OVERFLOW_BUFFERS];
	protected:
		void decRef() {
			for(int n=0;n<MAX_OVERFLOW_BUFFERS;n++)
				if(bufs[n]) bufs[n]->decRef();
		}
	public:
		overflowWriteOut() : N(0) {
			for(int n=0;n<MAX_OVERFLOW_BUFFERS;n++)
				bufs[n] = NULL;
		}
		void addBuffer(singleLog *l) {
			assert(N < MAX_OVERFLOW_BUFFERS);
			bufs[N] = l;
			N++;
		}
		int totalSize() {
			int ret = 0;
			for(int n=0;n<MAX_OVERFLOW_BUFFERS;n++)
				if(bufs[n]) {
					ret += bufs[n]->buf.handle.len;
				}
			return ret;
		}
		void copyAllTo(char *d) {
			char *walk = d;
			for(int n=0;n<MAX_OVERFLOW_BUFFERS;n++)
				if(bufs[n]) {
					memcpy(d,bufs[n]->buf.handle.base,bufs[n]->buf.handle.len);
					walk += bufs[n]->buf.handle.len;
				}
		}
		~overflowWriteOut() {
			decRef();
		}
	};

	class logLabel final {
	public:
		heapBuf buf;
		logLabel(const char *s, const char *format = nullptr) : buf() {
			if(format != nullptr)
				buf.sprintf(format,s);
			else
				buf.sprintf("%s",s);
		}
		size_t length() {
			return buf.handle.len;
		}
		logLabel() : buf() {}
		bool empty() { return buf.empty(); }
		void setUTF8(const char *s, int len) {
			buf.malloc(len+1);
			memset(buf.handle.base,0,(size_t) len+1);
			buf.memcpy(s,(size_t) len);
		}
		logLabel& operator=(logLabel &o) {
			buf.assign(o.buf);
			return *this;
		}
		/**
		 * Creates a log label from a utf8 string with specific len. The normal cstor
		 * can be used to do this also, but this ensures the entire string is encoded (if there are bugs
		 * with your sprintf, etc for UTF8)
		 * @param s
		 * @param len
		 * @return
		 */
		static logLabel *fromUTF8(const char *s, int len) {
			logLabel *l = new logLabel();
			l->buf.malloc((size_t) len+1);
			memset(l->buf.handle.base,0,(size_t) len+1);
			l->buf.memcpy(s,len);
			return l;
		}
	};

	typedef TWlib::TW_KHash_32<uint32_t, logLabel *, TWlib::TW_Mutex, uint32_t_eqstrP, TWlib::Allocator<LoggerAlloc> > LabelTable;

	class delim_data final {
	public:
		heapBuf delim;
		heapBuf delim_output;
		delim_data() : delim(), delim_output() {}
		delim_data(delim_data&& o)
			: delim(std::move(o.delim)),
			  delim_output(std::move(o.delim_output)) {}

		void setDelim(char *s,int l) {
			delim.malloc(l);
			delim.memcpy(s,l);
		}
		void setOutputDelim(char *s,int l) {
			delim_output.malloc(l);
			delim_output.memcpy(s,l);
		}
		delim_data duplicate() {
			delim_data d;
			d.delim = delim.duplicate();
			d.delim_output = delim_output.duplicate();
			return d;
		}
		delim_data& operator=(delim_data&& o) {
			this->delim = std::move(o.delim);
			this->delim_output = std::move(o.delim_output);
			return *this;
		}
	};

#ifndef GREASE_LIB
	class target_start_info final {
	public:
		bool needsAsyncQueue; // if the callback must be called in the v8 thread
		_errcmn::err_ev err;      // used if above is true
		actionCB cb;
//		start_info *system_start_info;
		Nan::Callback *targetStartCB;
		TargetId targId;
		target_start_info() : needsAsyncQueue(false), err(), cb(NULL), targetStartCB(NULL), targId(0) {}
		target_start_info& operator=(target_start_info&& o) {
			if(o.err.hasErr())
				err = std::move(o.err);
			cb = o.cb; o.cb = NULL;
			targetStartCB = o.targetStartCB; o.targetStartCB = NULL;
			targId = o.targId;
			return *this;
		}
	};
#else
	class target_start_info final {
	public:
		bool needsAsyncQueue; // if the callback must be called in the v8 thread
		_errcmn::err_ev err;      // used if above is true
		actionCB cb;
		int optsId; // used for the caller to track which target got started

		GreaseLibCallback targetStartCB;
		TargetId targId;
		target_start_info() : needsAsyncQueue(false), err(), cb(NULL), targetStartCB(NULL), targId(0) {} // targetStartCB(NULL),
		target_start_info& operator=(target_start_info&& o) {
			if(o.err.hasErr())
				err = std::move(o.err);
			cb = o.cb; o.cb = NULL;
			targetStartCB = o.targetStartCB; o.targetStartCB = NULL;
			targId = o.targId;
			return *this;
		}
	};
#endif

	struct Opts_t {
		uv_mutex_t mutex;
		bool show_errors;
		bool callback_errors;
		int bufferSize;  // size of each buffer
		int chunkSize;
		uint32_t levelFilterOutMask;
		bool defaultFilterOut;
		Opts_t() : show_errors(false), callback_errors(false), levelFilterOutMask(0), defaultFilterOut(false) {
			uv_mutex_init(&mutex);
		}

		void lock() {
			uv_mutex_lock(&mutex);
		}

		void unlock() {
			uv_mutex_unlock(&mutex);
		}
	};
	Opts_t Opts;


protected:
	static GreaseLogger *LOGGER;  // this class is a Singleton

	// these are the primary buffers for all log messages. Logs are put here before targets are found,
	// or anything else happens (other than sift())
	TWlib::tw_safeCircular<singleLog  *, LoggerAlloc > masterBufferAvail;    // <-- available buffers (starts out full)

	uv_thread_t logThreadId;
	uv_async_t asyncV8LogCallback;  // used to wakeup v8 to call log callbacks (see v8LogCallbacks)
	uv_async_t asyncTargetCallback; // used to wakeup v8 to call callbacks on target starts

	uv_async_t asyncRefLogger;      // used to signal whether the node/v8 thread can exit or not (used as V8 ref/unref keep-alive handle)
	uv_async_t asyncUnrefLogger;    // used to signal whether the node/v8 thread can exit or not

	uv_mutex_t mutexRefLogger;
	uint32_t needV8;                // 0 means can exit, > 0 mean can't
	uint32_t needGreaseThread;
	// when understanding the uv_async stuff, it's important to read this note: https://nikhilm.github.io/uvbook/threads.html#inter-thread-communication
	// and realize that multiple calls to async_send only guarantee at least _one_ call of the callback

	void refGreaseInGrease() {
		uv_mutex_lock(&mutexRefLogger);
		DBG_OUT("[ask for ref] needGreaseThread=%d",needGreaseThread+1);
		if(needGreaseThread == 0) {
			uv_ref((uv_handle_t *)&flushTimer);  // we use the flush timer to keep the grease thread up...
			startFlushTimer();
		}
		needGreaseThread++;
		uv_mutex_unlock(&mutexRefLogger);
	}

	void unrefGreaseInGrease() {
		uv_mutex_lock(&mutexRefLogger);
		assert(needGreaseThread > 0);
		DBG_OUT("[unref] needGreaseThread=%d",needGreaseThread-1);
		needGreaseThread--;
		if(needGreaseThread < 1) {
			uv_unref((uv_handle_t *)&flushTimer);
			stopFlushTimer();
		}
		uv_mutex_unlock(&mutexRefLogger);
	}

#ifndef GREASE_LIB
	void refFromV8() {
		uv_mutex_lock(&mutexRefLogger);
		if(needV8 == 0) {
			DBG_OUT("[ask for ref] needV8=%d",needV8);
			uv_async_send(&asyncRefLogger);
		}
		needV8++;
		uv_mutex_unlock(&mutexRefLogger);
	}

	// to be called from non-V8 thread
	void unrefFromV8() {
		uv_mutex_lock(&mutexRefLogger);
		assert(needV8 > 0);  // FIXME
		needV8--;
		if(needV8 < 1) {
			uv_async_send(&asyncUnrefLogger);
		}
		uv_mutex_unlock(&mutexRefLogger);
	}

	// to be called from V8 thread
	void unrefFromV8_inV8() {
		uv_mutex_lock(&mutexRefLogger);
		assert(needV8 > 0);  // FIXME
		needV8--;
		DBG_OUT("[unref] needV8=%d",needV8);
		if(needV8 == 0) {
			uv_unref((uv_handle_t *)&asyncRefLogger);
		}
		int n = uv_loop_alive(uv_default_loop());
		DBG_OUT("v8 loop_alive=%d\n",n);

 		uv_mutex_unlock(&mutexRefLogger);
	}

#else
	void refFromV8() {
	}

	// to be called from non-V8 thread
	void unrefFromV8() {
	}

	// to be called from V8 thread
	void unrefFromV8_inV8() {
	}
#endif

#if UV_VERSION_MAJOR > 0
	static void refCb_Logger(uv_async_t* handle) {
#else
	static void refCb_Logger(uv_async_t* handle, int status) {
#endif
	  /* After closing the async handle, it will no longer keep the loop alive. */
//		uv_mutex_lock(&LOGGER->mutexRefLogger);
		uv_ref((uv_handle_t *)&LOGGER->asyncRefLogger);
//		uv_mutex_unlock(&LOGGER->mutexRefLogger);
	}

#if UV_VERSION_MAJOR > 0
	static void unrefCb_Logger(uv_async_t* handle) {
#else
	static void unrefCb_Logger(uv_async_t* handle, int status) {
#endif
//		uv_mutex_lock(&LOGGER->mutexRefLogger);
		uv_unref((uv_handle_t *)&LOGGER->asyncRefLogger);
//		uv_mutex_unlock(&LOGGER->mutexRefLogger);
	}

	TWlib::tw_safeCircular<target_start_info  *, LoggerAlloc > targetCallbackQueue;
	_errcmn::err_ev err;

	// Definitions:
	// Target - a final place a log entry goes: TTY, file, etc.x
	// Filter - a condition, which if matching, will cause a log to go to a target
	// FilterHash - a number used to point to one or more filters
	// Log entry - the log entry (1) - synonymous with a single call to log() or logSync()
	// Meta - the data in a log Entry beyond the message itself

	// Default target - where stuff goes if no matching filter is found
	//

    //
	// filter id -> { [N1:0 0:N2 N1:N2], [level mask], [target] }

	// table:tag -> N1  --> [filter id list]
	// table:origin -> N2  --> [filter id list]
	// search N1:0 0:N2 N1:N2 --> [filter id list]
	// filterMasterTable: uint64_t -> [ filter id list ]

	// { tag: "mystuff", origin: "crazy.js", level" 0x02 }


	uv_mutex_t nextIdMutex;
	uv_mutex_t nextOptsIdMutex; // used for the 'optsId' which helps calling code track callbacks, on AddTarget
	uint32_t nextOptsId;
	FilterId nextFilterId;
	TargetId nextTargetId ;
	SinkId nextSinkId;


	class Filter final {
	public:
		bool _disabled;
		FilterId id;
		LevelMask levelMask;
		TargetId targetId;
		logLabel preFormat;
		logLabel postFormatPreMsg;
		logLabel postFormat;
		Filter() : preFormat(), postFormat() {
			id = 0;  // an ID of 0 means - empty Filter
			levelMask = 0;
			targetId = 0;
			_disabled = false;
		};
		Filter(FilterId id, LevelMask mask, TargetId t) : _disabled(false), id(id), levelMask(mask), targetId(t), preFormat(), postFormatPreMsg(), postFormat() {}
//		Filter(Filter &&o) : id(o.id), levelMask(o.mask), targetId(o.t) {};
		Filter(Filter &o) :  _disabled(false), id(o.id), levelMask(o.levelMask), targetId(o.targetId) {};
		Filter& operator=(Filter& o) {
			id = o.id;
			levelMask = o.levelMask;
			targetId = o.targetId;
			preFormat = o.preFormat;
			postFormatPreMsg = o.postFormatPreMsg;
			postFormat = o.postFormat;
			return *this;
		}
	};

	class FilterList final {
	public:
		Filter list[MAX_IDENTICAL_FILTERS]; // a non-zero entry is valid. when encountering the first zero, the rest array elements are skipped
		LevelMask bloom; // this is the bloom filter for the list. If this does not match, the list does not log this level
		bool filterOut;  // if true, then this entire list is filtered out - and will not be logged.
		FilterList() : bloom(0), filterOut(false) {}
		inline bool valid(LevelMask m) {
			return (bloom & m);
		}
		inline bool add(Filter &filter) {
			bool ret = false;
			for(int n=0;n<MAX_IDENTICAL_FILTERS;n++) {
				if(list[n].id == 0) {
					bloom |= filter.levelMask;
					list[n] = filter;
					ret = true;
					break;
				}
			}
			return ret;
		}

		inline bool find(FilterId lookup_id, Filter *&filter) {
			bool ret = false;
			for(int n=0;n<MAX_IDENTICAL_FILTERS;n++) {
				if(list[n].id == lookup_id) {
					filter = &list[n];
					ret = true;
					break;
				}
			}
			return ret;
		}
	};


	class Sink {
	protected:
#ifndef GREASE_LIB
		Persistent<Function> onNewConnCB;
#endif		
	public:

		Sink() 
#ifndef GREASE_LIB
		: onNewConnCB() 
#endif
		{}

		static void parseAndLog() {

		}


		bool bind() {
			return false;
		}
		void start() {

		}
		void shutdown() {

		}
	};


	/**
	 * NOT COMPLETE
	 */
	class PipeSink final : public Sink, virtual public heapBuf::heapBufManager  {
	protected:
		TWlib::tw_safeCircular<heapBuf *, LoggerAlloc > buffers;
		struct PipeClient {
			char temp[SINK_BUFFER_SIZE];
			enum _state {
				NEED_PREAMBLE,
				IN_PREAMBLE,
				IN_LOG_ENTRY    // have log_entry_size
			};
			int temp_used;
			int state_remain;
			int log_entry_size;
			_state state;
			uv_pipe_t client;
			PipeSink *self;
			PipeClient() = delete;
			PipeClient(PipeSink *_self) :
				temp_used(0),
				state_remain(GREASE_CLIENT_HEADER_SIZE),  // the initial state requires preamble + size(uint32_t)
				log_entry_size(0),
				state(NEED_PREAMBLE), self(_self) {
				uv_pipe_init(_self->loop, &client, 0);
				client.data = this;
			}
			void resetState() {
				state = NEED_PREAMBLE;
				state_remain = GREASE_CLIENT_HEADER_SIZE;
				temp_used = 0;
				log_entry_size = 0;
			}
			void close() {

			}
			static void on_close(uv_handle_t *t) {
				PipeClient *c = (PipeClient *) t->data;
				delete c;
			}
			static void on_read(uv_stream_t* handle, ssize_t nread, uv_buf_t buf) {
				PipeClient *c = (PipeClient *) handle->data;
				if(nread == -1) {
					// time to shutdown - client left...
					uv_close((uv_handle_t *)&c->client, PipeClient::on_close);
					DBG_OUT("PipeSink client disconnect.\n");
				} else {
					int walk = 0;
					while(walk < buf.len) {
						switch(c->state) {
							case NEED_PREAMBLE:
								if(buf.len >= GREASE_CLIENT_HEADER_SIZE) {
									if(IS_SINK_PREAMBLE(buf.base)) {
										c->state = IN_LOG_ENTRY;
										GET_SIZE_FROM_PREAMBLE(buf.base,c->log_entry_size);
									} else {
										DBG_OUT("PipeClient: Bad state. resetting.\n");
										c->resetState();
									}
									walk += GREASE_CLIENT_HEADER_SIZE;
								} else {
									c->state_remain = SIZEOF_SINK_LOG_PREAMBLE - buf.len;
									memcpy(c->temp+c->temp_used, buf.base+walk,buf.len);
									c->state = IN_PREAMBLE;
									walk += buf.len;
								}
								break;
							case IN_PREAMBLE:
							{
								int n = c->state_remain;
								if(buf.len < n) n = buf.len;
								memcpy(c->temp+c->temp_used,buf.base,n);
								walk += n;
								c->state_remain = c->state_remain - n;
								if(c->state_remain == 0) {
									if(IS_SINK_PREAMBLE(c->temp)) {
										GET_SIZE_FROM_PREAMBLE(c->temp,c->log_entry_size);
										c->temp_used = 0;
										c->state = IN_LOG_ENTRY;
									} else {
										DBG_OUT("PipeClient: Bad state. resetting.\n");
										c->resetState();
									}
								}
								break;
							}
							case IN_LOG_ENTRY:
							{
								if(c->temp_used == 0) { // we aren't using the buffer,
									if((buf.len - walk) >= GREASE_TOTAL_MSG_SIZE(c->log_entry_size)) { // let's see if we have everything already
										int r;
										if((r = c->self->owner->logFromRaw(buf.base,GREASE_TOTAL_MSG_SIZE(c->log_entry_size)))
												!= GREASE_OK) {
											ERROR_OUT("Grease logFromRaw failure: %d\n", r);
										}
										walk += c->log_entry_size;
										c->resetState();
									} else {
										memcpy(c->temp,buf.base+walk,buf.len-walk);
										c->temp_used = buf.len;
										walk += buf.len; // end loop
									}
								} else {
									int need = GREASE_TOTAL_MSG_SIZE(c->log_entry_size) - c->temp_used;
									if(need <= buf.len-walk) {
										memcpy(c->temp + c->temp_used,buf.base+walk,need);
										walk += need;
										c->temp_used += need;
										int r;
										if((r = c->self->owner->logFromRaw(c->temp,GREASE_TOTAL_MSG_SIZE(c->log_entry_size)))
												!= GREASE_OK) {
											ERROR_OUT("Grease logFromRaw failure (2): %d\n", r);
										}
										c->resetState();
									} else {
										memcpy(c->temp + c->temp_used,buf.base+walk,buf.len-walk);
										walk += buf.len-walk;
										c->temp_used += buf.len-walk;
									}
								}
								break;
							}
						}
					}
				}
			}
		};

	public:
		uv_loop_t *loop;
		GreaseLogger *owner;
		char *path;
		SinkId id;
		uv_pipe_t pipe; // on Unix this is a AF_UNIX/SOCK_STREAM, on Windows its a Named Pipe
		PipeSink() = delete;
		PipeSink(GreaseLogger *o, char *_path, SinkId _id, uv_loop_t *l) :
				buffers(BUFFERS_PER_SINK),
				loop(l), owner(o), path(NULL), id(_id) {
			uv_pipe_init(l,&pipe,0);
			pipe.data = this;
			if(_path)
				path = local_strdup_safe(_path);
			for (int n=0;n<BUFFERS_PER_SINK;n++) {
				heapBuf *b = new heapBuf(SINK_BUFFER_SIZE);
				buffers.add(b);
			}
		}
		void returnBuffer(heapBuf *b) {
			buffers.add(b);
		}
		bool bind() {
			assert(path);
			uv_pipe_bind(&pipe,path);
			return true;
		}
		//uv_buf_t (*uv_alloc_cb)(uv_handle_t* handle, size_t suggested_size);
		static uv_buf_t alloc_cb(uv_handle_t* handle, size_t suggested_size) {
			uv_buf_t buf;
			heapBuf *b = NULL;
			PipeClient *client = (PipeClient *) handle->data;
			if(client->self->buffers.remove(b)) {          // grab an existing buffer
				buf.base = b->handle.base;
				buf.len = b->handle.len;
				b->return_cb = (heapBuf::heapBufManager *) client->self;  // assign class/callback
			} else {
				ERROR_OUT("PipeSink: alloc_cb failing. no sink buffers.\n");
				buf.len = 0;
				buf.base = NULL;
			}
			return buf;
		}
		static void on_new_conn(uv_stream_t* server, int status) {
			PipeSink *sink = (PipeSink *) server->data;
			if(status == 0 ) {
				PipeClient *client = new PipeClient(sink);
				int r;
#if UV_VERSION_MAJOR > 0
				if((r = uv_accept(server, (uv_stream_t *)&client->client)) < 0) {
					ERROR_OUT("PipeSink: Failed accept() %s\n", uv_strerror(r));
					delete sink;
				} else {
//					if(uv_read_start((uv_stream_t *)&client->client,PipeSink::alloc_cb, PipeClient::on_read)) {
//
//					}
				}

#else
				if((r = uv_accept(server, (uv_stream_t *)&client->client))==0) {
					ERROR_OUT("PipeSink: Failed accept()\n", uv_strerror(uv_last_error(sink->loop)));
					delete sink;
				} else {
					if(uv_read_start((uv_stream_t *)&client->client,PipeSink::alloc_cb, PipeClient::on_read)) {

					}
				}
#endif
			} else {
				ERROR_OUT("Sink: on_new_conn: Error on status: %d\n",status);
			}
		}
		void start() {
			pipe.data = this;
			uv_listen((uv_stream_t*)&pipe,16,&on_new_conn);
		}

		void stop() {

		}
		~PipeSink() {
			if(path) ::free(path);
			heapBuf *b;
			while(buffers.remove(b)) {
				delete b;
			}
		}

	};

	static constexpr char *re_capture_syslog = "^\\s*(?:\\<([0-9]*)\\>)?\\s*([a-zA-Z]{3}\\s+[0-9]{1,2}\\s+[0-9]{2}\\:[0-9]{2}\\:[0-9]{2})\\s+([\\S\\s]*)$";

	class SyslogDgramSink final : public Sink, virtual public heapBuf::heapBufManager  {
	protected:
		uv_thread_t listener_thread;
		char *path;

	public:

		uv_loop_t *loop;
		GreaseLogger *owner;
		SinkId id;
//		uv_pipe_t pipe; // on Unix this is a AF_UNIX/SOCK_STREAM, on Windows its a Named Pipe
		uv_mutex_t control_mutex;
		bool stop_thread;
		int socket_fd;
		int wakeup_pipe[2];
		static const int PIPE_WAIT = 1;
		static const int PIPE_WAKEUP = 0;
		static const int DESIRED_SOCKET_SIZE = 65536;
		struct sockaddr_un sink_dgram_addr;
		bool ready;
		SyslogDgramSink() = delete;
		SyslogDgramSink(GreaseLogger *o, char *_path, SinkId _id, uv_loop_t *l) :
//				buffers(BUFFERS_PER_SINK),
				path(NULL), loop(l), owner(o),  id(_id),
				stop_thread(false),
				socket_fd(0),
				wakeup_pipe(), // {-1,-1}
				ready(false) {
//			uv_pipe_init(l,&pipe,0);
//			pipe.data = this;
			wakeup_pipe[0] = -1; wakeup_pipe[1] = -1;
			if(_path && strlen(_path) > 0) {
				path = local_strdup_safe(_path);
			}
			else
				DBG_OUT("UnixDgramSink: No path set. Will fail.\n");
//			for (int n=0;n<BUFFERS_PER_SINK;n++) {
//				heapBuf *b = new heapBuf(SINK_BUFFER_SIZE);
////				buffers.add(b);
//			}
		}
		void returnBuffer(heapBuf *b) {
//			buffers.add(b);
		}
		bool bind() {
			if(path) {
				socket_fd = socket(AF_UNIX, SOCK_DGRAM, 0);
				if(socket_fd < 0) {
					ERROR_PERROR("SyslogDgramSink: Failed to create SOCK_DGRAM socket.\n", errno);
				} else {

					if(pipe(wakeup_pipe) < 0) {
						ERROR_PERROR("SyslogDgramSink: Failed to pipe() for SOCK_DGRAM socket.\n", errno);
					} else {
						fcntl(wakeup_pipe[PIPE_WAIT], F_SETFL, O_NONBLOCK);
						::memset(&sink_dgram_addr,0,sizeof(sink_dgram_addr));
						sink_dgram_addr.sun_family = AF_UNIX;
						unlink(path); // get rid of it if it already exists
						strcpy(sink_dgram_addr.sun_path,path);
						if(::bind(socket_fd, (const struct sockaddr *) &sink_dgram_addr, sizeof(sink_dgram_addr)) < 0) {
							ERROR_PERROR("SyslogDgramSink: Failed to bind() SOCK_DGRAM socket.\n", errno);
							close(socket_fd);
						} else {
							// set permissions:
							if(chmod(path, 0666) < 0) {
								ERROR_PERROR("SyslogDgramSink: could not set permissions to 0666\n",errno);
							}

							ready = true;
						}
					}
				}
			} else {
				ERROR_OUT("SyslogDgramSink: No path set.");
			}
			return ready;
		}


		static void listener_work(void *self) {
			SyslogDgramSink *sink = (SyslogDgramSink *) self;

			fd_set readfds;

			char dump[5];

			const int nbuffers = 10;
			char *raw_buffer[nbuffers];
			struct iovec iov[nbuffers];
			struct msghdr message;

			socklen_t optsize;

			char *temp_buffer_entry = (char *) malloc(GREASE_MAX_MESSAGE_SIZE);


			// using MSG_DONTWAIT instead
//			int flags = fcntl(socket_fd, F_GETFL, 0);
//			if (flags < 0) {
//				ERROR_PERROR("UnixDgramSink: Error getting socket flags\n",errno);
//			}
//			flags = flags|O_NONBLOCK;
//			if(fcntl(socket_fd, F_SETFL, flags) < 0) {
//				ERROR_PERROR("UnixDgramSink: Error setting socket non-blocking flags\n",errno);
//			}

			// discover socket max recieve size (this will be the max for a non-fragmented log message
			int rcv_buf_size = 65536;
			setsockopt(sink->socket_fd, SOL_SOCKET, SO_RCVBUF, &rcv_buf_size, (unsigned int) sizeof(int));

			getsockopt(sink->socket_fd, SOL_SOCKET, SO_RCVBUF, &rcv_buf_size, &optsize);
			// http://stackoverflow.com/questions/10063497/why-changing-value-of-so-rcvbuf-doesnt-work
			if(rcv_buf_size < 100) {
				ERROR_OUT("SyslogDgramSink: Failed to start reader thread - SO_RCVBUF too small\n");
			} else {
				DBG_OUT("SyslogDgramSink: SO_RCVBUF is %d\n", rcv_buf_size);
			}

			for(int n=0;n<nbuffers;n++) {
				raw_buffer[n] = (char *) ::malloc(rcv_buf_size+1);
			}

			enum {
				NEED_PREAMBLE,
				IN_PREAMBLE,
				IN_LOG
			} state;
			state = NEED_PREAMBLE;

			FD_ZERO(&readfds);
			FD_SET(sink->wakeup_pipe[PIPE_WAIT], &readfds);
			FD_SET(sink->socket_fd, &readfds);

			int n = sink->socket_fd + 1;
			if(sink->wakeup_pipe[PIPE_WAIT] > n)
				n = sink->wakeup_pipe[PIPE_WAIT]+1;

//			RE2 re_dissect_syslog(re_capture_syslog);

			GreaseLogger *l = GreaseLogger::setupClass();
			GreaseLogger::singleLog *entry = NULL;

			int recv_cnt = 0;
			int err_cnt = 0;
			while(sink->ready && !sink->stop_thread) {
				int err = select(n,&readfds,NULL,NULL,NULL); // block and wait...
				if(err == -1) {
					ERROR_PERROR("SyslogDgramSink: error on select() \n", errno);
				} else if(err > 0) {
					if(FD_ISSET(sink->wakeup_pipe[PIPE_WAIT], &readfds)) {
						while(read(sink->wakeup_pipe[PIPE_WAIT], dump, 1) == 1) {}
					}
					if(FD_ISSET(sink->socket_fd, &readfds)) {

						// ////////////////////////////////
						// Business work of sink.........
						// ////////////////////////////////
						message.msg_name=&sink->sink_dgram_addr;
						message.msg_namelen=sizeof(struct sockaddr_un);
						message.msg_iov=iov;
						message.msg_iovlen=nbuffers;
						message.msg_control=NULL;
						message.msg_controllen=0;
						message.msg_flags = 0;

						for(int n=0;n<nbuffers;n++) {
							memset(raw_buffer[n],0,rcv_buf_size);
							iov[n].iov_base = raw_buffer[n];
							iov[n].iov_len = rcv_buf_size;
						}

						if((recv_cnt = recvmsg(sink->socket_fd, &message, MSG_DONTWAIT)) < 0) {
							if(errno != EAGAIN || errno != EWOULDBLOCK) {
								ERROR_PERROR("SyslogDgramSink: Error on recvfrom() ", errno);
								err_cnt++;
							}
						}

						// TODO:
						// use setsockopt SO_PASSCRED to get the actual calling PID through a recvmsg() as an aux message
						// This way you can determine the process ID which can be mapped to the origin label

//						char header_temp[GREASE_CLIENT_HEADER_SIZE];
//						int header_temp_walk = 0;
						int temp_buffer_walk = 0;
						int walk = 0;
						int walk_buf = 0;
						int remain = recv_cnt;
						int remain_in_buf = (recv_cnt > rcv_buf_size) ? rcv_buf_size : recv_cnt;
						int iov_n = 0;
						char *current_buffer = NULL;
						uint32_t entry_size;
						state = NEED_PREAMBLE;


						GreaseLogger::syslog_parse_state state = SYSLOG_BEGIN;

						while(iov_n < nbuffers && remain > 0) {
							if(iov[iov_n].iov_len > 0) {

#ifdef ERRCMN_DEBUG_BUILD
								// if (!re_dissect_syslog.ok()) {
								// 	DBG_OUT("OOPS!!!!!!! - regex is not compiling!");
								// }
								char *buf = (char *) malloc(iov[iov_n].iov_len+1);
								::memcpy(buf,iov[iov_n].iov_base,iov[iov_n].iov_len);
								*(buf+iov[iov_n].iov_len) = 0;
								DBG_OUT("syslog in %d>> %s\n",iov_n, buf);
								free(buf);
#endif

// 								tempPiece.set((char *) iov[iov_n].iov_base,iov[iov_n].iov_len);
// 								if(RE2::FullMatch(tempPiece,re_dissect_syslog,&fac_pri,&cap_date,&cap_msg)) {
// //									if(cap_date.length() > 3) {
// //										// we figure out the log time here, via strptime() - but honestly, its gonna be when it was sent - so don't bother
// //									}
// 									pri = LOG_PRI(fac_pri);
// 									if( pri < 8) {
// 										meta_syslog.level = GREASE_SYSLOGPRI_TO_LEVEL_MAP[pri];
// 									} else {
// 										meta_syslog.level = GREASE_LEVEL_LOG;
// 										DBG_OUT("out of bounds LOG_PRI ");
// 									}
// 									fac = LOG_FAC(fac_pri);
// //									DBG_OUT("fac_pri %d  %d\n",fac_pri,fac);
// 									if( fac < sizeof(GREASE_SYSLOGFAC_TO_TAG_MAP)) {
// 										meta_syslog.tag = GREASE_SYSLOGFAC_TO_TAG_MAP[fac];
// 									} else {
// 										meta_syslog.tag = GREASE_TAG_SYSLOG;
// 										DBG_OUT("out of bounds LOG_FAC %d",fac);
// 									}


// 									sink->owner->logP(&meta_syslog,cap_msg.c_str(),cap_msg.length());
// 								} else {

// 									DBG_OUT("NO MATCH @ RE2 for syslog input");
// 									// regex did not match. - just log the whole message
// 									sink->owner->logP(&meta_syslog,(char *) iov[iov_n].iov_base,iov[iov_n].iov_len);

// 								}

								int r = recv_cnt;
								if(l->_grabInLogBuffer(entry) == GREASE_OK) {									
									if(GreaseLogger::parse_single_syslog_to_singleLog((char *) iov[iov_n].iov_base, r, state, entry)) {
										entry->incRef();
										l->_submitBuffer(entry);										
									} else {
										if(state == SYSLOG_INVALID) {
											DBG_OUT("Invalid syslog state! (iovec recvmsg() call)");
										} else {
											DBG_OUT("Incomplete syslog on iovec recvmsg() call");
										}
										l->_returnBuffer(entry);
									}
								} else {
									DBG_OUT("SyslogDgramSink::listener_work() failed to _grabInLogBuffer() - ouch.");
								}

								remain -= iov[iov_n].iov_len;
								iov_n++;

							} else {
								DBG_OUT("syslog in %d ZERO length",iov_n );
								break;
							}
						}


					}
				} // else timeout
			}

			free(temp_buffer_entry);
		}

		void start() {
			uv_thread_create(&listener_thread,SyslogDgramSink::listener_work,(void *) this);
		}

		void stop() {
			uv_mutex_lock(&control_mutex);
			stop_thread = true;
			uv_mutex_unlock(&control_mutex);
			wakeup_thread();
		}


		~SyslogDgramSink() {
			if(path) ::free(path);
			if(wakeup_pipe[0] < 0)
				close(wakeup_pipe[0]);
			if(wakeup_pipe[1] < 0)
				close(wakeup_pipe[1]);
		}

	protected:
		void wakeup_thread() {
			if(wakeup_pipe[PIPE_WAKEUP] > -1) {
				write(wakeup_pipe[PIPE_WAKEUP], "x", 1);
			}
		}
	};  // end SyslogDatagramSink


//	static constexpr char *re_capture_kernelog = "\\<([0-9])\\>\\[[0-9]+\\.[0-9]+\\]([^\\n]+)";
//	static constexpr char *re_default_kernlog_path = "/proc/kmsg";

	/* Close the log.  Currently a NOP. */
	#define KSYSLOG_ACTION_CLOSE          0
	/* Open the log. Currently a NOP. */
	#define KSYSLOG_ACTION_OPEN           1
	/* Read from the log. */
	#define KSYSLOG_ACTION_READ           2
	/* Read all messages remaining in the ring buffer. (allowed for non-root) */
	#define KSYSLOG_ACTION_READ_ALL       3
	/* Read and clear all messages remaining in the ring buffer */
	#define KSYSLOG_ACTION_READ_CLEAR     4
	/* Clear ring buffer. */
	#define KSYSLOG_ACTION_CLEAR          5
	/* Disable printk's to console */
	#define KSYSLOG_ACTION_CONSOLE_OFF    6
	/* Enable printk's to console */
	#define KSYSLOG_ACTION_CONSOLE_ON     7
	/* Set level of messages printed to console */
	#define KSYSLOG_ACTION_CONSOLE_LEVEL  8
	/* Return number of unread characters in the log buffer */
	#define KSYSLOG_ACTION_SIZE_UNREAD    9
	/* Return size of the log buffer */
	#define KSYSLOG_ACTION_SIZE_BUFFER   10

	class KernelProcKmsgSink final : public Sink, virtual public heapBuf::heapBufManager  {
	protected:

		uv_thread_t listener_thread;
//		char *path;

	public:

		uv_loop_t *loop;
		GreaseLogger *owner;
		SinkId id;

		uv_mutex_t control_mutex;
		bool stop_thread;
		int kernlog_fd;
		int kernbufsize;
		bool valid;
		int wakeup_pipe[2];
		static const int PIPE_WAIT = 1;
		static const int PIPE_WAKEUP = 0;
//		static const int DESIRED_SOCKET_SIZE = 65536;
		struct sockaddr_un sink_dgram_addr;
		bool ready;
		KernelProcKmsgSink() = delete;
		KernelProcKmsgSink(GreaseLogger *o, SinkId _id, uv_loop_t *l) :
//				buffers(BUFFERS_PER_SINK),
				loop(l), owner(o),  id(_id),
				stop_thread(false),
				kernlog_fd(0),
				kernbufsize(2^14),
				valid(false),
				wakeup_pipe(), // {-1,-1}
				ready(false) {
//			uv_pipe_init(l,&pipe,0);
//			pipe.data = this;
			wakeup_pipe[0] = -1; wakeup_pipe[1] = -1;
//			if(_path && strlen(_path) > 0) {
//				path = local_strdup_safe(_path);
//			}
//			else
//				DBG_OUT("KernelProcKmsgSink: No path set. Will fail.\n");
//			for (int n=0;n<BUFFERS_PER_SINK;n++) {
//				heapBuf *b = new heapBuf(SINK_BUFFER_SIZE);
////				buffers.add(b);
//			}
		}
		void returnBuffer(heapBuf *b) {
//			buffers.add(b);
		}
		bool bind() {
			kernbufsize = ::klogctl(KSYSLOG_ACTION_SIZE_BUFFER, NULL, 0);
			if(kernbufsize > 0) {
				valid = true;
			}
			if(pipe(wakeup_pipe) < 0) {
				ERROR_PERROR("KernelProcKmsgSink: Failed to create wakeup pipe().\n", errno);
			} else {
				fcntl(wakeup_pipe[PIPE_WAIT], F_SETFL, O_NONBLOCK);
			}

			return ready;
		}

#define SINK_KLOG_TV_USEC_START 250000

		static void listener_work(void *self) {
			KernelProcKmsgSink *sink = (KernelProcKmsgSink *) self;

			fd_set readfds;

			char dump[5];

			struct timeval timeout;
			timeout.tv_sec = 0;
			timeout.tv_usec = SINK_KLOG_TV_USEC_START; // 0.25 seconds
			int buf_size = sink->kernbufsize*2;

			if(!sink->valid) {
				ERROR_OUT("KernelProcKmsgSink: NOT VALID. Can't start thread.");
				return;
			}

			char *temp_buffer_entry = (char *) malloc(buf_size);
			int buf_remain = buf_size;
			GreaseLogger::klog_parse_state parse_state = LEVEL_BEGIN;
			char *buf_curpos = temp_buffer_entry;
			FD_ZERO(&readfds);
			FD_SET(sink->wakeup_pipe[PIPE_WAIT], &readfds);
//			FD_SET(sink->socket_fd, &readfds);
			DECL_LOG_META(meta_klog, GREASE_TAG_KERNEL, GREASE_LEVEL_LOG, 0 ); // static meta struct we will use
			int Z = 0;
			int reads = 0;

			GreaseLogger *l = GreaseLogger::setupClass();
			GreaseLogger::singleLog *entry = NULL;


			//			int err = select(n,&readfds,NULL,NULL,NULL); // block and wait...
			while(1) {
				Z++;
				DBG_OUT("TOp - KLOG %d",Z);
				int err = select(sink->wakeup_pipe[PIPE_WAIT] + 1, &readfds, NULL, NULL, &timeout);
				if(err == -1) {
					ERROR_PERROR("select() error",errno);
				} else if(err == 0) {
					DBG_OUT("TOp - timeout - KLOG");
					bool again = true;
					reads = 0;
					while(again) {

						int readn = klogctl(KSYSLOG_ACTION_SIZE_UNREAD,temp_buffer_entry,buf_size);
						if (readn > 0) {
							readn = klogctl(KSYSLOG_ACTION_READ, temp_buffer_entry, buf_size-1);
							temp_buffer_entry[readn] = 0; // make last char NULL - for debug
							DBG_OUT("READ Klog: %s",temp_buffer_entry);
							buf_remain = buf_size -1;
							buf_curpos = temp_buffer_entry;
							parse_state = LEVEL_BEGIN;
							while(1) {

								if(l->_grabInLogBuffer(entry) == GREASE_OK) {

									if(GreaseLogger::parse_single_klog_to_singleLog(buf_curpos, buf_remain, parse_state, entry, buf_curpos)) {
										DBG_OUT("PARSED Klog: %s",buf_curpos);
										entry->incRef();
										l->_submitBuffer(entry);
									} else {
										if(parse_state == INVALID) {
											DBG_OUT("INVALID Klog: %s",buf_curpos);
											l->_returnBuffer(entry);
											break;
										} else {
											DBG_OUT("INCOMPLETE Klog: %s",buf_curpos);
											// TODO: fixme to continue this buffer
											l->_returnBuffer(entry);
											break;
										}
									}

								} else {
									ERROR_OUT("Failed to get buffer: _greaseLib_handle_stdoutFd_cb");
									break;
								}

							}
							reads++;
						} else {
							again = false;
						}
					}
					if(reads == 0) {
						timeout.tv_usec = SINK_KLOG_TV_USEC_START * 4;
					} else
					if(reads == 1) {
						timeout.tv_usec = SINK_KLOG_TV_USEC_START * 2;
					} else {
						timeout.tv_usec = SINK_KLOG_TV_USEC_START;
					}
					// timeout
				} else {
					DBG_OUT("TOp - LOG - PIPE");
					if(FD_ISSET(sink->wakeup_pipe[PIPE_WAIT], &readfds)) {
						while(read(sink->wakeup_pipe[PIPE_WAIT], dump, 1) == 1) {}
					}
				}
				uv_mutex_lock(&sink->control_mutex);
				if(sink->stop_thread) {
					uv_mutex_unlock(&sink->control_mutex);
					break;
				} else {
					uv_mutex_unlock(&sink->control_mutex);
				}
			}

			free(temp_buffer_entry);
		}

		void start() {
			uv_thread_create(&listener_thread,KernelProcKmsgSink::listener_work,(void *) this);
		}

		void stop() {
			uv_mutex_lock(&control_mutex);
			stop_thread = true;
			uv_mutex_unlock(&control_mutex);
			wakeup_thread();
		}


		~KernelProcKmsgSink() {
			if(wakeup_pipe[0] < 0)
				close(wakeup_pipe[0]);
			if(wakeup_pipe[1] < 0)
				close(wakeup_pipe[1]);
		}

	protected:
		void wakeup_thread() {
			if(wakeup_pipe[PIPE_WAKEUP] > -1) {
				write(wakeup_pipe[PIPE_WAKEUP], "x", 1);
			}
		}
	};  // end Klog class



	/**
	 * In modern kernels, after 3.5, you can read the kernel log directly from
	 * /dev/kmsg, which allows use to use select and not the lousy, polling on a timer.
	 *
	 * https://stackoverflow.com/questions/32615442/kernel-log-file-descriptor-for-use-with-select
	 * Refer to the above, and the newer dmesg source.
	 * https://github.com/karelzak/util-linux/blob/master/sys-utils/dmesg.c
	 */
	class KernelProcKmsg2Sink final : public Sink, virtual public heapBuf::heapBufManager  {
	protected:
		static constexpr char *KERNLOG_PATH = "/dev/kmsg";
		uv_thread_t listener_thread;
//		char *path;

	public:

		uv_loop_t *loop;
		GreaseLogger *owner;
		SinkId id;

		uv_mutex_t control_mutex;
		bool stop_thread;
		int kernlog_fd;
		int kernbufsize;
		bool valid;
		int wakeup_pipe[2];
		static const int PIPE_WAIT = 1;
		static const int PIPE_WAKEUP = 0;
//		static const int DESIRED_SOCKET_SIZE = 65536;
		struct sockaddr_un sink_dgram_addr;
		bool ready;
		KernelProcKmsg2Sink() = delete;
		KernelProcKmsg2Sink(GreaseLogger *o, SinkId _id, uv_loop_t *l) :
//				buffers(BUFFERS_PER_SINK),
				loop(l), owner(o),  id(_id),
				stop_thread(false),
				kernlog_fd(0),
				kernbufsize(2^14),
				valid(false),
				wakeup_pipe(), // {-1,-1}
				ready(false) {
//			uv_pipe_init(l,&pipe,0);
//			pipe.data = this;
			wakeup_pipe[0] = -1; wakeup_pipe[1] = -1;
//			if(_path && strlen(_path) > 0) {
//				path = local_strdup_safe(_path);
//			}
//			else
//				DBG_OUT("KernelProcKmsgSink: No path set. Will fail.\n");
//			for (int n=0;n<BUFFERS_PER_SINK;n++) {
//				heapBuf *b = new heapBuf(SINK_BUFFER_SIZE);
////				buffers.add(b);
//			}
		}
		void returnBuffer(heapBuf *b) {
//			buffers.add(b);
		}
		bool bind() {
			kernbufsize = ::klogctl(KSYSLOG_ACTION_SIZE_BUFFER, NULL, 0);
			if(kernbufsize > 0) {
				valid = true;
			}
			if(pipe(wakeup_pipe) < 0) {
				ERROR_PERROR("KernelProcKmsgSink: Failed to create wakeup pipe().\n", errno);
			} else {
				fcntl(wakeup_pipe[PIPE_WAIT], F_SETFL, O_NONBLOCK);
			}

			return ready;
		}

#define SINK_KLOG_TV_USEC_START 250000

		static int open_kmsg(const char *path, int &_errno) {
			int mode = O_RDONLY | O_NONBLOCK;
			int fd = open(path, mode);
			if (fd < 0) {
				_errno = errno;
				return -1;
			} else {
				lseek(fd, 0, SEEK_END); // get last message
				return fd;
			}
		}

		static void listener_work(void *self) {
			KernelProcKmsgSink *sink = (KernelProcKmsgSink *) self;

			fd_set readfds;

			char dump[5];

//			struct timeval timeout;
//			timeout.tv_sec = 0;
//			timeout.tv_usec = SINK_KLOG_TV_USEC_START; // 0.25 seconds
			int buf_size = sink->kernbufsize;

			if(!sink->valid) {
				ERROR_OUT("KernelProcKmsg2Sink: NOT VALID. Can't start thread.");
				return;
			}



			char *temp_buffer_entry = (char *) malloc(buf_size);
			int remaining_to_parse = buf_size;
			GreaseLogger::klog_parse_state parse_state = LEVEL_BEGIN;
			char *buf_curpos = temp_buffer_entry;
			FD_ZERO(&readfds);
			FD_SET(sink->wakeup_pipe[PIPE_WAIT], &readfds);
			int _errno = 0;
			int kmsg_fd = open_kmsg(KERNLOG_PATH, _errno);
			if(kmsg_fd > 0) {
				FD_SET(kmsg_fd,&readfds);
			} else {
				ERROR_OUT("KernelProcKmsg2Sink: FATAL for thread. Can't open %s\n",KERNLOG_PATH);
				close(sink->wakeup_pipe[0]);
				close(sink->wakeup_pipe[1]);
				sink->ready = false;
				sink->valid = false;
				return;
			}

			int last_fd = kmsg_fd + 1;
			if(sink->wakeup_pipe[PIPE_WAIT] > last_fd)
				last_fd = sink->wakeup_pipe[PIPE_WAIT]+1;

			//			FD_SET(sink->socket_fd, &readfds);
			DECL_LOG_META(meta_klog, GREASE_TAG_KERNEL, GREASE_LEVEL_LOG, 0 ); // static meta struct we will use
			int Z = 0;
			int reads = 0;

			GreaseLogger *l = GreaseLogger::setupClass();
			GreaseLogger::singleLog *entry = NULL;

			char *read_into = temp_buffer_entry;
			int buffer_remain = buf_size;
			//			int err = select(n,&readfds,NULL,NULL,NULL); // block and wait...
			while(1) {
				Z++;
				DBG_OUT("TOp - KLOG %d",Z);
				lseek(kmsg_fd, 0, SEEK_END); // get last message
				int err = select(last_fd, &readfds, NULL, NULL, NULL);
				DBG_OUT("INNER KLOG 1 -- %d",Z);
				if(err == -1) {
					ERROR_PERROR("select() error",errno);
				} else {
					if(FD_ISSET(kmsg_fd, &readfds)) {
						DBG_OUT("INNER KLOG 1.1 %d",Z);
						// ok - the kmsg is readable. Do it.
						if( parse_state != LEVEL_BEGIN ||  // if _not_ LEVEL_BEGIN, then we have  fragment
							l->_grabInLogBuffer(entry) == GREASE_OK) {
							DBG_OUT("INNER KLOG 2 %d",Z);
							int readn = ::read(kmsg_fd,read_into,(int) buf_size - (read_into - temp_buffer_entry));
							remaining_to_parse = buffer_remain;
							if (readn > 0) {

								read_into[readn] = 0;
								DBG_OUT("KLOG Got read %s",read_into);
								if(GreaseLogger::parse_single_devklog_to_singleLog(read_into, readn, parse_state, entry, buf_curpos)) {
									DBG_OUT("PARSED Klog: %s",buf_curpos);
									entry->incRef();
									l->_submitBuffer(entry);
									read_into = temp_buffer_entry;
									buffer_remain = buf_size;
									entry = NULL;
								} else {
									if(parse_state == INVALID) {
										DBG_OUT("INVALID Klog: %s",buf_curpos);
										l->_returnBuffer(entry);
										read_into = temp_buffer_entry;
										buffer_remain = buf_size;
										parse_state = LEVEL_BEGIN;
										entry = NULL;
										break;
									} else {
										DBG_OUT("INCOMPLETE Klog: %s",buf_curpos);
										read_into = buf_curpos;
//										l->_returnBuffer(entry);
										break;
									}
								}
							} else {
								l->_returnBuffer(entry);
								entry = NULL;
							}

						} else {
							ERROR_OUT("KernelProcKmsg2Sink: Failed to get buffer: _greaseLib_handle_stdoutFd_cb");
							break;
						}
					}
					if(FD_ISSET(sink->wakeup_pipe[PIPE_WAIT], &readfds)) {
						while(read(sink->wakeup_pipe[PIPE_WAIT], dump, 1) == 1) {}
					}
				}
				uv_mutex_lock(&sink->control_mutex);
				if(sink->stop_thread) {
					uv_mutex_unlock(&sink->control_mutex);
					if(entry) l->_returnBuffer(entry);
					break;
				} else {
					uv_mutex_unlock(&sink->control_mutex);
				}
			}
//					DBG_OUT("TOp - timeout - KLOG");
//					bool again = true;
//					reads = 0;
//					while(again) {
//
//						int readn = klogctl(KSYSLOG_ACTION_SIZE_UNREAD,temp_buffer_entry,buf_size);
//						if (readn > 0) {
//							readn = klogctl(KSYSLOG_ACTION_READ, temp_buffer_entry, buf_size-1);
//							temp_buffer_entry[readn] = 0; // make last char NULL - for debug
//							DBG_OUT("READ Klog: %s",temp_buffer_entry);
//							buf_remain = buf_size -1;
//							buf_curpos = temp_buffer_entry;
//							parse_state = LEVEL_BEGIN;
//							while(1) {
//
//								if(l->_grabInLogBuffer(entry) == GREASE_OK) {
//
//									if(GreaseLogger::parse_single_klog_to_singleLog(buf_curpos, buf_remain, parse_state, entry, buf_curpos)) {
//										DBG_OUT("PARSED Klog: %s",buf_curpos);
//										entry->incRef();
//										l->_submitBuffer(entry);
//									} else {
//										if(parse_state == INVALID) {
//											DBG_OUT("INVALID Klog: %s",buf_curpos);
//											l->_returnBuffer(entry);
//											break;
//										} else {
//											DBG_OUT("INCOMPLETE Klog: %s",buf_curpos);
//											// TODO: fixme to continue this buffer
//											l->_returnBuffer(entry);
//											break;
//										}
//									}
//
//								} else {
//									ERROR_OUT("KernelProcKmsg2Sink:Failed to get buffer: _greaseLib_handle_stdoutFd_cb");
//									break;
//								}
//
//							}
//							reads++;
//						} else {
//							again = false;
//						}
//					}
//					if(reads == 0) {
//						timeout.tv_usec = SINK_KLOG_TV_USEC_START * 4;
//					} else
//					if(reads == 1) {
//						timeout.tv_usec = SINK_KLOG_TV_USEC_START * 2;
//					} else {
//						timeout.tv_usec = SINK_KLOG_TV_USEC_START;
//					}
//					// timeout
//				} else {
//					DBG_OUT("TOp - LOG - PIPE");
//					if(FD_ISSET(sink->wakeup_pipe[PIPE_WAIT], &readfds)) {
//						while(read(sink->wakeup_pipe[PIPE_WAIT], dump, 1) == 1) {}
//					}
//				}
//			}

			free(temp_buffer_entry);
		}

		void start() {
			uv_thread_create(&listener_thread,KernelProcKmsg2Sink::listener_work,(void *) this);
		}

		void stop() {
			uv_mutex_lock(&control_mutex);
			stop_thread = true;
			uv_mutex_unlock(&control_mutex);
			wakeup_thread();
		}


		~KernelProcKmsg2Sink() {
			if(wakeup_pipe[0] < 0)
				close(wakeup_pipe[0]);
			if(wakeup_pipe[1] < 0)
				close(wakeup_pipe[1]);
		}

	protected:
		void wakeup_thread() {
			if(wakeup_pipe[PIPE_WAKEUP] > -1) {
				write(wakeup_pipe[PIPE_WAKEUP], "x", 1);
			}
		}
	};  // end Klog class




	/**
	 * The difference between using a uv_pipe (implemented by libuv as a unix socket,
	 * and a Unix datagram socket, is that message boundaries get preserved.
	 * This really simplifies things.
	 *
	 * Read more here:
	 * http://stackoverflow.com/questions/4669710/atomic-write-on-an-unix-socket
	 */
	class UnixDgramSink final : public Sink, virtual public heapBuf::heapBufManager  {
	protected:
//		TWlib::tw_safeCircular<heapBuf *, LoggerAlloc > buffers;
//		struct UnixDgramClient {
////			char temp[SINK_BUFFER_SIZE];
//			enum _state {
//				NEED_PREAMBLE,
//				IN_PREAMBLE,
//				IN_LOG_ENTRY    // have log_entry_size
//			};
//			int temp_used;
//			int state_remain;
//			int log_entry_size;
//			_state state;
//			uv_pipe_t client;
//			UnixDgramSink *self;
//			UnixDgramClient() = delete;
//			UnixDgramClient(UnixDgramSink *_self) :
//				temp_used(0),
//				state_remain(GREASE_CLIENT_HEADER_SIZE),  // the initial state requires preamble + size(uint32_t)
//				log_entry_size(0),
//				state(NEED_PREAMBLE), self(_self) {
//				uv_pipe_init(_self->loop, &client, 0);
//				client.data = this;
//			}
//			void resetState() {
//				state = NEED_PREAMBLE;
//				state_remain = GREASE_CLIENT_HEADER_SIZE;
//				temp_used = 0;
//				log_entry_size = 0;
//			}
//			void close() {
//
//			}
//			static void on_close(uv_handle_t *t) {
//				PipeClient *c = (PipeClient *) t->data;
//				delete c;
//			}
//			static void on_read(uv_stream_t* handle, ssize_t nread, uv_buf_t buf) {
//				PipeClient *c = (PipeClient *) handle->data;
//				if(nread == -1) {
//					// time to shutdown - client left...
//					uv_close((uv_handle_t *)&c->client, PipeClient::on_close);
//					DBG_OUT("UnixDgramSink client disconnect.\n");
//				} else {
//					int walk = 0;
//					while(walk < buf.len) {
//						switch(c->state) {
//							case NEED_PREAMBLE:
//								if(buf.len >= GREASE_CLIENT_HEADER_SIZE) {
//									if(IS_SINK_PREAMBLE(buf.base)) {
//										c->state = IN_LOG_ENTRY;
//										GET_SIZE_FROM_PREAMBLE(buf.base,c->log_entry_size);
//									} else {
//										DBG_OUT("PipeClient: Bad state. resetting.\n");
//										c->resetState();
//									}
//									walk += GREASE_CLIENT_HEADER_SIZE;
//								} else {
//									c->state_remain = SIZEOF_SINK_LOG_PREAMBLE - buf.len;
//									memcpy(c->temp+c->temp_used, buf.base+walk,buf.len);
//									c->state = IN_PREAMBLE;
//									walk += buf.len;
//								}
//								break;
//							case IN_PREAMBLE:
//							{
//								int n = c->state_remain;
//								if(buf.len < n) n = buf.len;
//								memcpy(c->temp+c->temp_used,buf.base,n);
//								walk += n;
//								c->state_remain = c->state_remain - n;
//								if(c->state_remain == 0) {
//									if(IS_SINK_PREAMBLE(c->temp)) {
//										GET_SIZE_FROM_PREAMBLE(c->temp,c->log_entry_size);
//										c->temp_used = 0;
//										c->state = IN_LOG_ENTRY;
//									} else {
//										DBG_OUT("PipeClient: Bad state. resetting.\n");
//										c->resetState();
//									}
//								}
//								break;
//							}
//							case IN_LOG_ENTRY:
//							{
//								if(c->temp_used == 0) { // we aren't using the buffer,
//									if((buf.len - walk) >= GREASE_TOTAL_MSG_SIZE(c->log_entry_size)) { // let's see if we have everything already
//										int r;
//										if((r = c->self->owner->logFromRaw(buf.base,GREASE_TOTAL_MSG_SIZE(c->log_entry_size)))
//												!= GREASE_OK) {
//											ERROR_OUT("Grease logFromRaw failure: %d\n", r);
//										}
//										walk += c->log_entry_size;
//										c->resetState();
//									} else {
//										memcpy(c->temp,buf.base+walk,buf.len-walk);
//										c->temp_used = buf.len;
//										walk += buf.len; // end loop
//									}
//								} else {
//									int need = GREASE_TOTAL_MSG_SIZE(c->log_entry_size) - c->temp_used;
//									if(need <= buf.len-walk) {
//										memcpy(c->temp + c->temp_used,buf.base+walk,need);
//										walk += need;
//										c->temp_used += need;
//										int r;
//										if((r = c->self->owner->logFromRaw(c->temp,GREASE_TOTAL_MSG_SIZE(c->log_entry_size)))
//												!= GREASE_OK) {
//											ERROR_OUT("Grease logFromRaw failure (2): %d\n", r);
//										}
//										c->resetState();
//									} else {
//										memcpy(c->temp + c->temp_used,buf.base+walk,buf.len-walk);
//										walk += buf.len-walk;
//										c->temp_used += buf.len-walk;
//									}
//								}
//								break;
//							}
//						}
//					}
//				}
//			}
//		};

		uv_thread_t listener_thread;
		char *path;

	public:

		uv_loop_t *loop;
		GreaseLogger *owner;
		SinkId id;
//		uv_pipe_t pipe; // on Unix this is a AF_UNIX/SOCK_STREAM, on Windows its a Named Pipe
		uv_mutex_t control_mutex;
		bool stop_thread;
		int socket_fd;
		int wakeup_pipe[2];
		static const int PIPE_WAIT = 1;
		static const int PIPE_WAKEUP = 0;
		static const int DESIRED_SOCKET_SIZE = 65536;
		struct sockaddr_un sink_dgram_addr;
		bool ready;
		UnixDgramSink() = delete;
		UnixDgramSink(GreaseLogger *o, char *_path, SinkId _id, uv_loop_t *l) :
//				buffers(BUFFERS_PER_SINK),
				path(NULL), loop(l), owner(o),  id(_id),
				stop_thread(false),
				socket_fd(0),
				wakeup_pipe(), // {-1,-1}
				ready(false) {
//			uv_pipe_init(l,&pipe,0);
//			pipe.data = this;
			wakeup_pipe[0] = -1; wakeup_pipe[1] = -1;
			if(_path && strlen(_path) > 0) {
				path = local_strdup_safe(_path);
			}
			else
				DBG_OUT("UnixDgramSink: No path set. Will fail.\n");
//			for (int n=0;n<BUFFERS_PER_SINK;n++) {
//				heapBuf *b = new heapBuf(SINK_BUFFER_SIZE);
////				buffers.add(b);
//			}
		}
		void returnBuffer(heapBuf *b) {
//			buffers.add(b);
		}
		bool bind() {
			if(path) {
				socket_fd = socket(AF_UNIX, SOCK_DGRAM, 0);
				if(socket_fd < 0) {
					ERROR_PERROR("UnixDgramSink: Failed to create SOCK_DGRAM socket.\n", errno);
				} else {
					if(pipe(wakeup_pipe) < 0) {
						ERROR_PERROR("UnixDgramSink: Failed to pipe() for SOCK_DGRAM socket.\n", errno);
					} else {
						fcntl(wakeup_pipe[PIPE_WAIT], F_SETFL, O_NONBLOCK);
						::memset(&sink_dgram_addr,0,sizeof(sink_dgram_addr));
						sink_dgram_addr.sun_family = AF_UNIX;
						unlink(path); // get rid of it if it already exists
						strcpy(sink_dgram_addr.sun_path,path);
						if(::bind(socket_fd, (const struct sockaddr *) &sink_dgram_addr, sizeof(sink_dgram_addr)) < 0) {
							ERROR_PERROR("UnixDgramSink: Failed to bind() SOCK_DGRAM socket.\n", errno);
							close(socket_fd);
						} else {
							// set permissions:
							if(chmod(path, 0666) < 0) {
								ERROR_PERROR("UnixDgramSink: could not set permissions to 0666\n",errno);
							}

							ready = true;
						}
					}
				}
			} else {
				ERROR_OUT("UnixDgramSink: No path set.");
			}
			return ready;
		}


		//uv_buf_t (*uv_alloc_cb)(uv_handle_t* handle, size_t suggested_size);
//		static uv_buf_t alloc_cb(uv_handle_t* handle, size_t suggested_size) {
//			uv_buf_t buf;
//			heapBuf *b = NULL;
//			PipeClient *client = (PipeClient *) handle->data;
//			if(client->self->buffers.remove(b)) {          // grab an existing buffer
//				buf.base = b->handle.base;
//				buf.len = b->handle.len;
//				b->return_cb = (heapBuf::heapBufManager *) client->self;  // assign class/callback
//			} else {
//				ERROR_OUT("UnixDgramSink: alloc_cb failing. no sink buffers.\n");
//				buf.len = 0;
//				buf.base = NULL;
//			}
//			return buf;
//		}
//		static void on_new_conn(uv_stream_t* server, int status) {
//			UnixDgramSink *sink = (UnixDgramSink *) server->data;
//			if(status == 0 ) {
//				PipeClient *client = new PipeClient(sink);
//				int r;
//				if((r = uv_accept(server, (uv_stream_t *)&client->client))==0) {
//					ERROR_OUT("UnixDgramSink: Failed accept()\n", uv_strerror(uv_last_error(sink->loop)));
//					delete sink;
//				} else {
//					if(uv_read_start((uv_stream_t *)&client->client,UnixDgramSink::alloc_cb, PipeClient::on_read)) {
//
//					}
//				}
//			} else {
//				ERROR_OUT("Sink: on_new_conn: Error on status: %d\n",status);
//			}
//		}

		static void listener_work(void *self) {
			UnixDgramSink *sink = (UnixDgramSink *) self;

			fd_set readfds;

			char dump[5];

			const int nbuffers = 10;
			char *raw_buffer[nbuffers];
			struct iovec iov[nbuffers];
			struct msghdr message;

			socklen_t optsize;

			char *temp_buffer_entry = (char *) malloc(GREASE_MAX_MESSAGE_SIZE);


			// using MSG_DONTWAIT instead
//			int flags = fcntl(socket_fd, F_GETFL, 0);
//			if (flags < 0) {
//				ERROR_PERROR("UnixDgramSink: Error getting socket flags\n",errno);
//			}
//			flags = flags|O_NONBLOCK;
//			if(fcntl(socket_fd, F_SETFL, flags) < 0) {
//				ERROR_PERROR("UnixDgramSink: Error setting socket non-blocking flags\n",errno);
//			}

			// discover socket max recieve size (this will be the max for a non-fragmented log message
			int rcv_buf_size = 65536;
			setsockopt(sink->socket_fd, SOL_SOCKET, SO_RCVBUF, &rcv_buf_size, sizeof(int));

			getsockopt(sink->socket_fd, SOL_SOCKET, SO_RCVBUF, &rcv_buf_size, &optsize);
			// http://stackoverflow.com/questions/10063497/why-changing-value-of-so-rcvbuf-doesnt-work
			if(rcv_buf_size < 100) {
				ERROR_OUT("UnixDgramSink: Failed to start reader thread - SO_RCVBUF too small\n");
			} else {
				DBG_OUT("UnixDgramSink: SO_RCVBUF is %d\n", rcv_buf_size);
			}

			for(int n=0;n<nbuffers;n++) {
				raw_buffer[n] = (char *) malloc(rcv_buf_size);
			}

			enum {
				NEED_PREAMBLE,
				IN_PREAMBLE,
				IN_LOG
			} state;
			state = NEED_PREAMBLE;

			FD_ZERO(&readfds);
			FD_SET(sink->wakeup_pipe[PIPE_WAIT], &readfds);
			FD_SET(sink->socket_fd, &readfds);

			int n = sink->socket_fd + 1;
			if(sink->wakeup_pipe[PIPE_WAIT] > n)
				n = sink->wakeup_pipe[PIPE_WAIT]+1;


			int recv_cnt = 0;
			int err_cnt = 0;
			while(sink->ready && !sink->stop_thread) {
				int err = select(n,&readfds,NULL,NULL,NULL); // block and wait...
				if(err == -1) {
					ERROR_PERROR("UnixDgramSink: error on select() \n", errno);
				} else if(err > 0) {
					if(FD_ISSET(sink->wakeup_pipe[PIPE_WAIT], &readfds)) {
						while(read(sink->wakeup_pipe[PIPE_WAIT], dump, 1) == 1) {}
					}
					if(FD_ISSET(sink->socket_fd, &readfds)) {

						// ////////////////////////////////
						// Business work of sink.........
						// ////////////////////////////////
						message.msg_name=&sink->sink_dgram_addr;
						message.msg_namelen=sizeof(struct sockaddr_un);
						message.msg_iov=iov;
						message.msg_iovlen=nbuffers;
						message.msg_control=NULL;
						message.msg_controllen=0;
						message.msg_flags = 0;

						for(int n=0;n<nbuffers;n++) {
							memset(raw_buffer[n],0,rcv_buf_size);
							iov[n].iov_base = raw_buffer[n];
							iov[n].iov_len = rcv_buf_size;
						}

						if((recv_cnt = recvmsg(sink->socket_fd, &message, MSG_DONTWAIT)) < 0) {
							if(errno != EAGAIN || errno != EWOULDBLOCK) {
								ERROR_PERROR("UnixDgramSink: Error on recvfrom() ", errno);
								err_cnt++;
							}
						}

//						DBG_OUT("recv_cnt = %d\n",recv_cnt);
//						DBG_OUT("msg_iovlen = %d\n", message.msg_iovlen);
//						for(int n=0;n<nbuffers;n++) {
//							DBG_OUT("msg_iov[%d].iov_len = %d\n", n, iov[n].iov_len);
//							DBG_OUT("iov.base[%d] = %s\n",n, iov[n].iov_base);
//
//						}


						char header_temp[GREASE_CLIENT_HEADER_SIZE];
						int header_temp_walk = 0;
						int temp_buffer_walk = 0;
						int walk = 0;
						int walk_buf = 0;
						int remain = recv_cnt;
						int remain_in_buf = (recv_cnt > rcv_buf_size) ? rcv_buf_size : recv_cnt;
						int iov_n = 0;
						char *current_buffer = NULL;
						uint32_t entry_size;
						state = NEED_PREAMBLE;

#define SD_WAL_BUF_P (current_buffer + (walk_buf))
#define SD_WALK(N) do{walk += N; walk_buf += N; remain = remain - N; remain_in_buf = remain_in_buf - N;}while(0)

						while(walk < recv_cnt) {
							if(walk_buf >= recv_cnt) {
								walk_buf = 0;
								iov_n++;
								if(iov_n >= nbuffers) {
									ERROR_OUT("UnixDgramSink: Overflow on dgram buffers - too big of message\n");
									break;
								}
								remain_in_buf = (remain > rcv_buf_size) ? rcv_buf_size : remain;
							}
							current_buffer = (char *) iov[iov_n].iov_base;
							switch(state) {
								case NEED_PREAMBLE:
									if(remain >= GREASE_CLIENT_PING_SIZE) {
										if(IS_SINK_PING(SD_WAL_BUF_P)) {
											SD_WALK(GREASE_CLIENT_HEADER_SIZE);
											DBG_OUT(">>>>>>> GOT PING.\n");
											// send ping ack...
											if(sendto(sink->socket_fd, (char *) &__grease_sink_ping_ack, GREASE_CLIENT_PING_SIZE, 0,
													(struct sockaddr *) message.msg_name, message.msg_namelen) < 0) {
												ERROR_PERROR("UnixDgramSink (ping ack): Error on sendto() ", errno);
											}
											continue;
										}
										if(IS_SINK_PREAMBLE(SD_WAL_BUF_P)) {
											state = IN_LOG;
											GET_SIZE_FROM_PREAMBLE(SD_WAL_BUF_P,entry_size);
											DBG_OUT(">>>> dgram sink see %d bytes",entry_size);
										} else {
											DBG_OUT("UnixDgram Client: Bad state. resetting. %d\n",sizeof(uint32_t));
											state = NEED_PREAMBLE;
										}
										SD_WALK(GREASE_CLIENT_HEADER_SIZE);
									} else {
										memcpy(header_temp + header_temp_walk, SD_WAL_BUF_P,remain);
										SD_WALK(remain);
										remain = 0;
										state = IN_PREAMBLE;
									}
									break;
								case IN_PREAMBLE:
								{
									int nibble = GREASE_CLIENT_HEADER_SIZE - header_temp_walk;
									if(remain_in_buf < nibble) n = remain_in_buf;
									memcpy(header_temp + header_temp_walk, SD_WAL_BUF_P,nibble);
									SD_WALK(nibble);
									if(header_temp_walk >= GREASE_CLIENT_HEADER_SIZE) {
										if(IS_SINK_PREAMBLE(header_temp)) {
											GET_SIZE_FROM_PREAMBLE(header_temp,entry_size);
											DBG_OUT(">>>> dgram sink see %d bytes",entry_size);
											header_temp_walk = 0;
											state = IN_LOG;
										} else {
											DBG_OUT("PipeClient: Bad state. resetting. (2)\n");
											state = NEED_PREAMBLE;
										}
									}
									break;
								}
								case IN_LOG:
								{
									if(temp_buffer_walk == 0) { // we aren't using the buffer,
										if(remain_in_buf >= entry_size) { // let's see if we have everything already
											int r;
											if((r = sink->owner->logFromRaw(SD_WAL_BUF_P,entry_size))
													!= GREASE_OK) {
												ERROR_OUT("Grease logFromRaw failure: %d\n", r);
											}
											SD_WALK(entry_size);
											state = NEED_PREAMBLE;
										} else {
											memcpy(temp_buffer_entry,SD_WAL_BUF_P,remain_in_buf);
											temp_buffer_walk = remain_in_buf;
											SD_WALK(remain_in_buf);
										}
									} else {
										int need = entry_size - temp_buffer_walk;
										memcpy(temp_buffer_entry + temp_buffer_walk,SD_WAL_BUF_P,need);
										if(need <= remain_in_buf) {
											int r;
											if((r = sink->owner->logFromRaw(temp_buffer_entry,entry_size))
													!= GREASE_OK) {
												ERROR_OUT("Grease logFromRaw failure (2): %d\n", r);
											}
											state = NEED_PREAMBLE;
											temp_buffer_walk =0;
										} else {
											temp_buffer_walk += need;
										}
										SD_WALK(need);
									}
									break;
								}
							}
						}



					}
				} // else timeout
			}

			free(temp_buffer_entry);
		}

		void start() {
			uv_thread_create(&listener_thread,UnixDgramSink::listener_work,(void *) this);
		}

		void stop() {
			uv_mutex_lock(&control_mutex);
			stop_thread = true;
			uv_mutex_unlock(&control_mutex);
			wakeup_thread();
		}


		~UnixDgramSink() {
			if(path) ::free(path);
			if(wakeup_pipe[0] < 0)
				close(wakeup_pipe[0]);
			if(wakeup_pipe[1] < 0)
				close(wakeup_pipe[1]);
		}

	protected:
		void wakeup_thread() {
			if(wakeup_pipe[PIPE_WAKEUP] > -1) {
				write(wakeup_pipe[PIPE_WAKEUP], "x", 1);
			}
		}
	};

	struct logBuf {
		uv_buf_t handle;
		uv_mutex_t mutex;
		int id;
		const uint32_t space;
		delim_data delim;
//		int delimLen;
		bool tempBuffer; // a flag which lets you mark the buffer as temporary. Temporary buffers get deleted/freed when done
		logBuf(int s, int id, delim_data _delim) : id(id), space(s), delim(std::move(_delim)), tempBuffer(false) {
			handle.base = (char *) LMALLOC(space);
			handle.len = 0;
			uv_mutex_init(&mutex);
		}
		logBuf() = delete;
		~logBuf() {
			if(handle.base) LFREE(handle.base);
		}
		void copyIn(const char *s, int n, bool use_delim = true) {
			if(n > 0) {
				uv_mutex_lock(&mutex);
				assert(n <= space);
				if(handle.len > space) {
					CRITICAL_FAILURE("handle.len > space!!! - copy will fail.");
					handle.len = space;
					uv_mutex_unlock(&mutex);
 					return;
				}
//				assert(handle.len <= space);
				memcpy((void *) (handle.base + handle.len), s, n);
				handle.len += n;
				if(!delim.delim.empty() && use_delim) {
					::memcpy((void *) (handle.base + handle.len), delim.delim.handle.base, delim.delim.handle.len);
					handle.len += delim.delim.handle.len;
				}
				uv_mutex_unlock(&mutex);
			}
		}
		void copyInJsonEscape(const char *s, int n, bool use_delim) {
			if(n > 0) {
				uv_mutex_lock(&mutex);
				assert(n <= space);
				assert(handle.len <= space);
				//int memcpy_and_json_escape(char *out, char *in, int in_len, int *out_len)
				int out_len = space;
				memcpy_and_json_escape((handle.base + handle.len), s, n, &out_len);
				handle.len += out_len;
				if(!delim.delim.empty() && use_delim) {
					::memcpy((void *) (handle.base + handle.len), delim.delim.handle.base, delim.delim.handle.len);
					handle.len += delim.delim.handle.len;
				}
				uv_mutex_unlock(&mutex);
			}
		}
		void clear() {
			uv_mutex_lock(&mutex);
			handle.len = 0;
			uv_mutex_unlock(&mutex);
		}
		bool isEmpty() {
			uv_mutex_lock(&mutex);
			bool ret = (handle.len == 0);
			uv_mutex_unlock(&mutex);
			return ret;
		}
		int remain() {
			int ret = 0;
			uv_mutex_lock(&mutex);
			ret = space - handle.len;
			if(!delim.delim.empty()) ret = ret - delim.delim.handle.len;
			uv_mutex_unlock(&mutex);
			if(ret < 0) ret = 0;
			return ret;
		}
	};


	enum nodeCommand {
		NOOP,
		GLOBAL_FLUSH,  // flush all targets
		SHUTDOWN
//		ADD_TARGET,
//		CHANGE_FILTER
	};

	enum internalCommand {
		INTERNALNOOP,
		NEW_LOG,
		TARGET_ROTATE_BUFFER,   // we had to rotate buffers on a target
		WRITE_TARGET_OVERFLOW,   // too big to fit in buffer... will flush existing target, and then make large write.
		INTERNAL_SHUTDOWN
	};
//	class data {
//	public:
//		int x;
//		data() : x(0) {}
//		data(data &) = delete;
//		data(data &&o) : x(o.x) { o.x = 0; }
//		data& operator=(data&& other) {
//		     x = other.x;
//		     other.x = 0;
//		     return *this;
//		}
//	};

	struct internalCmdReq final {
		internalCommand c;
		uint32_t d;
		void *aux;
		int auxInt;
		internalCmdReq(internalCommand _c, uint32_t _d = 0): c(_c), d(_d), aux(NULL), auxInt(0) {}
		internalCmdReq(internalCmdReq &) = delete;
		internalCmdReq() : c(INTERNALNOOP), d(0), aux(NULL), auxInt(0) {}
		internalCmdReq(internalCmdReq &&o) : c(o.c), d(o.d), aux(o.aux), auxInt(o.auxInt) {};
		internalCmdReq& operator=(internalCmdReq&& other) {
			c = other.c;
			d = other.d;
			aux = other.aux;
			auxInt = other.auxInt;
			return *this;
		}
	};

	struct logReq final {
		uv_work_t work;
		int _errno; // the errno that happened on read if an error occurred.
//		v8::Persistent<Function> onSendSuccessCB;
//		v8::Persistent<Function> onSendFailureCB;
//		v8::Persistent<Object> buffer; // Buffer object passed in
//		char *_backing; // backing of the passed in Buffer
//		int len;
		GreaseLogger *self;
		// need Buffer
		logReq(GreaseLogger *i) : _errno(0),
//				onSendSuccessCB(), onSendFailureCB(), buffer(), _backing(NULL), len(0),
				self(i) {
			work.data = this;
		}
		logReq() = delete;
	};

	struct nodeCmdReq final {
//		uv_work_t work;
		nodeCommand cmd;
		int _errno; // the errno that happened on read if an error occurred.

#ifndef GREASE_LIB
		Nan::Callback *callback;
#endif
//		v8::Persistent<Function> onFailureCB;
//		v8::Persistent<Object> buffer; // Buffer object passed in
//		char *_backing; // backing of the passed in Buffer
//		int len;
		GreaseLogger *self;
		// need Buffer
		nodeCmdReq(nodeCommand c, GreaseLogger *i) : cmd(c), _errno(0),
#ifndef GREASE_LIB
				callback(),
#endif
//				onSuccessCB(), onFailureCB(),
//				buffer(), _backing(NULL), len(0),
				self(i) {
//			work.data = this;
		}
		nodeCmdReq(nodeCmdReq &) = delete;
		nodeCmdReq() : cmd(nodeCommand::NOOP), _errno(0), 
#ifndef GREASE_LIB
		callback(), 
#endif		
		self(NULL) {}
		nodeCmdReq(nodeCmdReq &&o) :cmd(o.cmd), _errno(o._errno), 
#ifndef GREASE_LIB
		callback(NULL), 
#endif		
		self(o.self) {
#ifndef GREASE_LIB
			if(o.callback) {
				callback = o.callback; o.callback = NULL;
			}
#endif			
		};
		nodeCmdReq& operator=(nodeCmdReq&& o) {
			cmd = o.cmd;
			_errno = o._errno;
#ifndef GREASE_LIB
			if(o.callback) {
				callback = o.callback; o.callback = NULL;
			}
#endif			
			return *this;
		}
	};

	static const char empty_label[];  // a marker for labels and other string with no value

	class logTarget {
	protected:
		bool _disabled;
	public:
		const uint32_t JSON_ESCAPE_STRINGS = GREASE_JSON_ESCAPE_STRINGS;
		// this strips string of their SGR and similar escpae codes
		const uint32_t STRIP_ANSI_ESCAPE_CODES = GREASE_STRIP_ANSI_ESCAPE_CODES;

		typedef void (*targetReadyCB)(bool ready, _errcmn::err_ev &err, logTarget *t);
		targetReadyCB readyCB;
		target_start_info *readyData;
//		logBuf *_buffers[NUM_BANKS];
		logBuf **_buffers;
		bool logCallbackSet; // if true, logCallback (below) is set (we prefer to not touch any v8 stuff, when outside the v8 thread)
#ifndef GREASE_LIB
		Nan::Callback *logCallback;
#else
		GreaseLibTargetCallback logCallback;
#endif
		delim_data delim;
		uint32_t numBanks;

		// Queue flow:
		// 1) un-used logBuf taken out of availBuffers, filled with data
		// 2) placed in writeOutBuffers for write out
		// 2) logBuf gets written out (usually in a separate thread)
		// 3) after being wirrten out, if no 'logCallbac' is assigned, logBuf is returned to availBuffers, OR
		//    if logCallback is set, then it is queued in waitingOnCBBuffers
		// 4) when v8 thread becomes active, callback is called, then logBuf is returned to availBuffers
		TWlib::tw_safeCircular<logBuf *, LoggerAlloc > availBuffers; // buffers which are available for writing
		TWlib::tw_safeCircular<logBuf *, LoggerAlloc > writeOutBuffers; // buffers which are available for writing
		TWlib::tw_safeCircular<logBuf *, LoggerAlloc > waitingOnCBBuffers; // buffers which are available for writing
		_errcmn::err_ev err;
		int _log_fd;

		uv_mutex_t writeMutex;
//		int logto_n;    // log to use to buffer log.X calls. The 'current Buffer' index in _buffers
		logBuf *currentBuffer;
		int bankSize;
		GreaseLogger *owner;
		uint32_t myId;
		uint32_t flags;

		bool isDisableWrites() {
			bool ret = false;
			uv_mutex_lock(&writeMutex);
			ret = _disabled;
			uv_mutex_unlock(&writeMutex);
			return ret;
		}

		void disableWrites(bool v) {
			uv_mutex_lock(&writeMutex);
			_disabled = v;
			uv_mutex_unlock(&writeMutex);
		}

		void setFlag(uint32_t flag) {
			flags |= flag;
		}


		logTarget(int buffer_size, uint32_t id, GreaseLogger *o,
				targetReadyCB cb, delim_data _delim, target_start_info *readydata, uint32_t num_banks = NUM_BANKS);
		logTarget() = delete;

		logLabel timeFormat;
		logLabel tagFormat;
		logLabel originFormat;
		logLabel levelFormat;
		logLabel preFormat;
		logLabel postFormat;
		logLabel preMsgFormat; // new one, this is printed right before the given message, but after above data

		void setTimeFormat(const char *s, int len) {
			uv_mutex_lock(&writeMutex);
			timeFormat.setUTF8(s,len);
			uv_mutex_unlock(&writeMutex);
		}
		void setTagFormat(const char *s, int len) {
			uv_mutex_lock(&writeMutex);
			tagFormat.setUTF8(s,len);
			uv_mutex_unlock(&writeMutex);
		}
		void setOriginFormat(const char *s, int len) {
			uv_mutex_lock(&writeMutex);
			originFormat.setUTF8(s,len);
			uv_mutex_unlock(&writeMutex);
		}
		void setLevelFormat(const char *s, int len) {
			uv_mutex_lock(&writeMutex);
			levelFormat.setUTF8(s,len);
			uv_mutex_unlock(&writeMutex);
		}
		void setPreFormat(const char *s, int len) {
			uv_mutex_lock(&writeMutex);
			preFormat.setUTF8(s,len);
			uv_mutex_unlock(&writeMutex);
		}
		void setPostFormat(const char *s, int len) {
			uv_mutex_lock(&writeMutex);
			postFormat.setUTF8(s,len);
			uv_mutex_unlock(&writeMutex);
		}
		void setPreMsgFormat(const char *s, int len) {
			uv_mutex_lock(&writeMutex);
			preMsgFormat.setUTF8(s,len);
			uv_mutex_unlock(&writeMutex);
		}

		size_t putsHeader(char *mem, size_t remain, const logMeta &m, Filter *filter) {
			static __thread struct timeb _timeb;
			size_t space = remain;
			logLabel *label;
			int n = 0;
			if(filter && !filter->preFormat.empty()) {
				n = snprintf(mem + (remain-space),space,filter->preFormat.buf.handle.base);
				space = space - n;
			}
			if(preFormat.length() > 0 && space > 0) {
				n = snprintf(mem + (remain-space),space,preFormat.buf.handle.base);
				space = space - n;
			}
			if(timeFormat.length() > 0 && space > 0) {
//				time_t curr = time(NULL);
				ftime(&_timeb);
				n = snprintf(mem+(remain-space),space,timeFormat.buf.handle.base,_timeb.time,_timeb.millitm);
				space = space - n;
			}
			if(levelFormat.length() > 0 && space > 0) {
				if(owner->levelLabels.find(m.level,label) && label->length() > 0) {
					n = snprintf(mem + (remain-space),space,levelFormat.buf.handle.base,label->buf.handle.base);
					space = space - n;
				}
			}
			if(tagFormat.length() > 0 && space > 0) {
				if(m.tag > 0 && owner->tagLabels.find(m.tag,label) && label->length() > 0) {
					n = snprintf(mem + (remain-space),space,tagFormat.buf.handle.base,label->buf.handle.base);
					space = space - n;
				} else {
					n = snprintf(mem+(remain-space),space,tagFormat.buf.handle.base,empty_label);
					space = space - n;
				}
			}
			if(originFormat.length() > 0 && space > 0) {
				if(m.origin > 0 && owner->originLabels.find(m.origin,label) && label->length() > 0) {
					n = snprintf(mem+(remain-space),space,originFormat.buf.handle.base,label->buf.handle.base);
					space = space - n;
				} else {
					n = snprintf(mem+(remain-space),space,originFormat.buf.handle.base,empty_label);
					space = space - n;
				}
			}
			if(preMsgFormat.length() > 0 && space > 0) {
				n = snprintf(mem + (remain-space),space,preMsgFormat.buf.handle.base);
				space = space - n;
			}
			if(filter && !filter->postFormatPreMsg.empty()) {
				n = snprintf(mem + (remain-space),space,filter->postFormatPreMsg.buf.handle.base);
				space = space - n;
			}

			// returns the amount of space used.
			return remain-space;
		}

		size_t putsFooter(char *mem, size_t remain, Filter *filter) {
			size_t space = remain;
			logLabel *label;
			int n = 0;
			if(filter && !filter->postFormat.empty()) {
				n = snprintf(mem,space,filter->postFormat.buf.handle.base);
				space = space - n;
			}
			if(postFormat.length() > 0 && space > 0) {
				n = snprintf(mem,space,postFormat.buf.handle.base);
				space = space - n;
			}
			// returns the amount of space used.
			return remain-space;
		}


		class writeCBData final {
		protected:
			writeCBData(logTarget *_t, overflowWriteOut *_b) : overflow(_b), t(_t), b(NULL), nocallback(false) {}
		public:
			overflowWriteOut *overflow;
			logTarget *t;
			logBuf *b;
			bool nocallback;
			writeCBData() : overflow(NULL), t(NULL), b(NULL) {}
			writeCBData(logTarget *_t, logBuf *_b) : overflow(NULL), t(_t), b(_b), nocallback(false) {}
			static writeCBData *newAsOverflow(logTarget *t, overflowWriteOut *b) {
				return new writeCBData(t,b);
			}
			writeCBData& operator=(writeCBData&& o) {
				overflow = o.overflow;
				o.overflow = NULL;
				b = o.b;
				t = o.t;
				return *this;
			}
			void freeOverflow() {
				if(overflow) delete overflow;
				overflow = NULL;
			}
			~writeCBData() {
				if(overflow) delete overflow;
			}
		};

		// called when the V8 callback is done
		void finalizeV8Callback(logBuf *b) {
			b->clear();
			if(!availBuffers.addIfRoom(b)) {
				DBG_OUT("ERROR ERROR availBuffers was full (makes no sense) \n");
				ERROR_OUT("ERROR ERROR availBuffers was full (makes no sense) \n");
			}
		}
#ifndef GREASE_LIB
		void returnBuffer(logBuf *b, bool sync = false, bool nocallback = false) {
			bool isempty = b->isEmpty();
			if(!isempty)
				owner->unrefGreaseInGrease();
			if(!nocallback && logCallbackSet && !isempty) {
				writeCBData cbdat(this,b);
				if(sync) {
					GreaseLogger::_doV8Callback(cbdat);
				} else {
					if(!owner->callerLogCallbacks.addMvIfRoom(cbdat)) {
						if(owner->Opts.show_errors)
							ERROR_OUT(" !!! v8LogCallbacks is full! Can't rotate. Callback will be skipped.");
						b->clear();
						availBuffers.add(b);
					} else {
						owner->refFromV8();
						uv_async_send(&owner->asyncV8LogCallback);
					}
				}
			} else {
				b->clear();
				availBuffers.add(b);
			}
		}

		void returnBuffer(overflowWriteOut *b, bool sync = false, bool nocallback = false) {
			owner->unrefGreaseInGrease();
			if(!nocallback && logCallbackSet && b) {
				writeCBData cbdat;
				cbdat.t = this;
				cbdat.overflow = b;
				if(!owner->callerLogCallbacks.addMvIfRoom(cbdat)) {
					if(owner->Opts.show_errors)
						ERROR_OUT(" !!! v8LogCallbacks is full! Can't rotate. Callback will be skipped.");
				}
				if(!sync) {
					owner->refFromV8();
					uv_async_send(&owner->asyncV8LogCallback);
				}
				else
					GreaseLogger::_doV8Callback(cbdat);
			} else {
				if(b) delete b;
			}
		}
#else
		void returnBuffer(logBuf *b, bool sync = false, bool nocallback = false) {
			bool isempty = b->isEmpty();
			if(!isempty)
				owner->unrefGreaseInGrease();
			if(!nocallback && logCallbackSet && !isempty) {
				writeCBData cbdat(this,b);
				if(sync) {
					GreaseLogger::_doLibCallback(cbdat);
				} else {
					if(!owner->callerLogCallbacks.addMvIfRoom(cbdat)) {
						DBG_OUT(" !!! v8LogCallbacks is full! Can't rotate. Callback will be skipped.");
						if(owner->Opts.show_errors)
							ERROR_OUT(" !!! v8LogCallbacks is full! Can't rotate. Callback will be skipped.");
						b->clear();
						availBuffers.add(b);
					} else {
						owner->refFromV8();
						uv_async_send(&owner->asyncV8LogCallback);
					}
				}
			} else {
				b->clear();
				availBuffers.add(b);
			}
		}

		void returnBuffer(overflowWriteOut *b, bool sync = false, bool nocallback = false) {
			owner->unrefGreaseInGrease();
			if(!nocallback && logCallbackSet && b) {
				writeCBData cbdat;
				cbdat.t = this;
				cbdat.overflow = b;
				if(!owner->callerLogCallbacks.addMvIfRoom(cbdat)) {
					DBG_OUT(" !!! v8LogCallbacks is full! Can't rotate. Callback will be skipped.");
					if(owner->Opts.show_errors)
						ERROR_OUT(" !!! v8LogCallbacks is full! Can't rotate. Callback will be skipped.");
				}
				if(!sync) {
					owner->refFromV8();
					uv_async_send(&owner->asyncV8LogCallback);
				}
				else
					GreaseLogger::_doLibCallback(cbdat);
			} else {
				if(b) delete b;
			}
		}
#endif

#ifndef GREASE_LIB
		void setCallback(Local<Function> &func) {
			uv_mutex_lock(&writeMutex);
			if(!func.IsEmpty()) {
				logCallbackSet = true;
				logCallback = new Nan::Callback(func);
			}
			uv_mutex_unlock(&writeMutex);
		}
#else
		void setCallback(GreaseLibTargetCallback cb) {
			uv_mutex_lock(&writeMutex);
			if(cb) {
				logCallbackSet = true;
				logCallback = cb;
			}
			uv_mutex_unlock(&writeMutex);
		}
#endif

		bool rotate() {
			bool ret = false;
			logBuf *next = NULL;
			uv_mutex_lock(&writeMutex);
			if(currentBuffer->handle.len > 0) {
				ret = availBuffers.remove(next);  // won't block
				if(ret) {
					ret = writeOutBuffers.addIfRoom(currentBuffer); // won't block
					if(ret) {
						currentBuffer = next;
					} else {
						if(owner->Opts.show_errors)
							ERROR_OUT(" !!! writeOutBuffers is full! Can't rotate. Data will be lost.");
						if(!availBuffers.addIfRoom(next)) {  // won't block
							if(owner->Opts.show_errors)
								ERROR_OUT(" !!!!!!!! CRITICAL ERROR - can't put Buffer back in availBuffers. Losing Buffer ?!@?!#!@");
						}
						currentBuffer->clear();
					}
				} else {
					DBG_OUT(" !!! Can't rotate. NO BUFFERS - Overwriting buffer!! Data will be lost. [target %d]", myId);
					if(owner->Opts.show_errors)
						ERROR_OUT(" !!! Can't rotate. NO BUFFERS - Overwriting buffer!! Data will be lost. [target %d]", myId);
					currentBuffer->clear();
				}
			} else {
				DBG_OUT("Skipping rotation. No data in current buffer.");
			}
//			if(!ret) {
//				ERROR_OUT(" !!! Can't rotate. Overwriting buffer!! Data will be lost.");
//			}
			uv_mutex_unlock(&writeMutex);
			return ret;
		}

		void write(const char *s, uint32_t len, const logMeta &m, Filter *filter) {  // called from grease thread...
			static __thread char header_buffer[GREASE_MAX_PREFIX_HEADER]; // used to make header of log line. for speed it's static. this is __thread
			                                                              // so when it is called from multiple threads buffers don't conflict
			                                                              // for a discussion of thread_local (a C++ 11 thing) and __thread (a gcc thing)
			                                                              // see here: http://stackoverflow.com/questions/12049095/c11-nontrivial-thread-local-static-variable
			static __thread char footer_buffer[GREASE_MAX_PREFIX_HEADER];
			static __thread int len_header_buffer;
			static __thread int len_footer_buffer;

			bool dropout = false;
			uv_mutex_lock(&writeMutex);
			dropout = _disabled;
			uv_mutex_unlock(&writeMutex);
			if(dropout) return;    // if the logTarget is disabled, ignore the write()

			// its huge. do an overflow request now, and move on.
			if(bankSize < (len+GREASE_MAX_PREFIX_HEADER)) {
				singleLog *B = NULL;
				if(flags & JSON_ESCAPE_STRINGS)
					B = singleLog::newSingleLogAsJsonEscape(s,len,m);
				else
					B = new singleLog(s,len,m);
				internalCmdReq req(WRITE_TARGET_OVERFLOW,myId);
				req.aux = B;
				if(owner->internalCmdQueue.addMvIfRoom(req))
					uv_async_send(&owner->asyncInternalCommand);
				else {
					if(owner->Opts.show_errors)
						ERROR_OUT("Overflow. Dropping WRITE_TARGET_OVERFLOW. Command overflow.");
					delete B;
				}
				return;
			}

			uv_mutex_lock(&writeMutex);
			len_header_buffer = putsHeader(header_buffer,(size_t) GREASE_MAX_PREFIX_HEADER,m,filter);
			len_footer_buffer = putsFooter(footer_buffer,(size_t) GREASE_MAX_PREFIX_HEADER-len_header_buffer,filter);
			uv_mutex_unlock(&writeMutex);
			bool no_footer = true;
			if(len_footer_buffer > 0) no_footer = false;

			if(currentBuffer->remain() >= (int) (len+len_header_buffer+len_footer_buffer)) {
				if(currentBuffer->isEmpty())
					owner->refGreaseInGrease();
				uv_mutex_lock(&writeMutex);
				currentBuffer->copyIn(header_buffer,len_header_buffer, false); // false means skip delimiter
				if(flags & JSON_ESCAPE_STRINGS)
					currentBuffer->copyInJsonEscape(s,len,no_footer); // false means skip delimiter
				else
					currentBuffer->copyIn(s,len,no_footer);
				if(!no_footer)
					currentBuffer->copyIn(footer_buffer,len_footer_buffer);
				uv_mutex_unlock(&writeMutex);
			} else {
				bool rotated = false;
				internalCmdReq req(TARGET_ROTATE_BUFFER,myId); // just rotated off a buffer
				rotated = rotate();
				if(rotated) {
					if(owner->internalCmdQueue.addMvIfRoom(req)) {
						int id = 0;
						uv_mutex_lock(&writeMutex);
						owner->refGreaseInGrease(); // new buffer for use, do the ref
						currentBuffer->copyIn(header_buffer,len_header_buffer, false);
						if(flags & JSON_ESCAPE_STRINGS)
							currentBuffer->copyInJsonEscape(s,len,no_footer); // false means skip delimiter
						else
							currentBuffer->copyIn(s,len,no_footer);
						if(!no_footer)
							currentBuffer->copyIn(footer_buffer,len_footer_buffer);
						id = currentBuffer->id;
						uv_mutex_unlock(&writeMutex);
						DBG_OUT("Request ROTATE [%d] [target %d]", id, myId);
						uv_async_send(&owner->asyncInternalCommand);
					} else {
						if(owner->Opts.show_errors)
							ERROR_OUT("Overflow. Dropping TARGET_ROTATE_BUFFER");
					}
				}
			}

		}

		virtual void writeAsyncOverflow(overflowWriteOut *b, bool nocallbacks) {
			delete b;
		}

		void writeOverflow(singleLog *log, Filter *filter) {  // called from grease thread...

			bool dropout = false;
			uv_mutex_lock(&writeMutex);
			dropout = _disabled;
			uv_mutex_unlock(&writeMutex);
			if(dropout) return;    // if the logTarget is disabled, ignore the write()


			overflowWriteOut *bufs = new overflowWriteOut();

			singleLog *hdr = singleLog::heapSingleLog(GREASE_MAX_PREFIX_HEADER);
			singleLog *ftr = singleLog::heapSingleLog(GREASE_MAX_PREFIX_HEADER);

			uv_mutex_lock(&writeMutex);
			int len_header_buffer = putsHeader(hdr->buf.handle.base,(size_t) GREASE_MAX_PREFIX_HEADER,log->meta.m,filter);
			int len_footer_buffer = putsFooter(ftr->buf.handle.base,(size_t) GREASE_MAX_PREFIX_HEADER-len_header_buffer,filter);
			uv_mutex_unlock(&writeMutex);


			hdr->buf.handle.len = len_header_buffer;
			ftr->buf.handle.len = len_footer_buffer;
			if(len_header_buffer > 0) {
				bufs->addBuffer(hdr);
			} else
				delete hdr;
			bufs->addBuffer(log);
			if(len_footer_buffer > 0) {
				bufs->addBuffer(ftr);
			} else
				delete ftr;
			singleLog *delim_buf = singleLog::heapSingleLog((int) delim.delim.handle.len);
			if(delim.delim.handle.len > 0) {
				delim_buf->buf.memcpy(delim.delim.handle.base,(int) delim.delim.handle.len);
				bufs->addBuffer(delim_buf);
			} else
				delete delim_buf;

			owner->refGreaseInGrease();
			writeAsyncOverflow(bufs,false);
		}


		void flushAll(bool _rotate = true, bool nocallbacks = false) {
			if(_rotate) rotate();
			while(1) {
				logBuf *b = NULL;
				if(writeOutBuffers.remove(b)) {
					flush(b, nocallbacks);
				} else
					break;
			}
			sync();
		}

		void flushAllSync(bool _rotate = true, bool nocallbacks = false) {
			HEAVY_DBG_OUT("flushAllSync()");
			if(_rotate) rotate();
			while(1) {
				logBuf *b = NULL;
				if(writeOutBuffers.remove(b)) {
					flushSync(b,nocallbacks);
				} else
					break;
			}
			sync();
		}

		// called from Logger thread ONLY
		virtual void flush(logBuf *b, bool nocallbacks = false) {}; // flush buffer 'n'. This is ansynchronous
		virtual void flushSync(logBuf *b, bool nocallbacks = false) {}; // flush buffer 'n'. This is ansynchronous
		virtual void writeSync(const char *s, int len, const logMeta &m) {}; // flush buffer 'n'. This is synchronous. Writes now - skips buffering
		virtual void close() {};
		virtual void sync() {};
		virtual ~logTarget();
	};


	class ttyTarget final : public logTarget {
	public:
		int ttyFD;
		uv_tty_t tty;
		static void write_cb(uv_write_t* req, int status) {
//			HEAVY_DBG_OUT("write_cb");
			writeCBData *d = (writeCBData *) req->data;
			d->t->returnBuffer(d->b,false,d->nocallback);
			delete req;
		}
		static void write_overflow_cb(uv_write_t* req, int status) {
			HEAVY_DBG_OUT("overflow_cb");
			writeCBData *d = (writeCBData *) req->data;
			d->t->returnBuffer(d->overflow,false,d->nocallback);
			d->overflow = NULL;
			delete d;
			delete req;
		}


		uv_write_t outReq;  // since this function is no re-entrant (below) we can do this
		void flush(logBuf *b,bool nocallbacks=false) override { // this will always be called by the logger thread (via uv_async_send)
			uv_write_t *req = new uv_write_t;
			writeCBData *d = new writeCBData;
			d->t = this;
			d->b = b;
			d->nocallback = nocallbacks;
			req->data = d;
			uv_mutex_lock(&b->mutex);  // this lock should be ok, b/c write is async
			HEAVY_DBG_OUT("TTY: flush() %d bytes", b->handle.len);
			uv_write(req, (uv_stream_t *) &tty, &b->handle, 1, write_cb);
			uv_mutex_unlock(&b->mutex);
//			b->clear(); // NOTE - there is a risk that this could get overwritten before it is written out.
		}
		void flushSync(logBuf *b,bool nocallbacks=false) override { // this will always be called by the logger thread (via uv_async_send)
			uv_write_t *req = new uv_write_t;
			writeCBData *d = new writeCBData;
			d->t = this;
			d->b = b;
			d->nocallback = nocallbacks;
			req->data = d;
			uv_mutex_lock(&b->mutex);  // this lock should be ok, b/c write is async
			HEAVY_DBG_OUT("TTY: flushSync() %d bytes", b->handle.len);
			returnBuffer(b,true,nocallbacks);
			uv_write(req, (uv_stream_t *) &tty, &b->handle, 1, NULL);
			uv_mutex_unlock(&b->mutex);
//			b->clear(); // NOTE - there is a risk that this could get overwritten before it is written out.
		}
		void writeAsyncOverflow(overflowWriteOut *b,bool nocallbacks ) override {
			uv_write_t *req = new uv_write_t;

			uv_buf_t bufs[MAX_OVERFLOW_BUFFERS];
			for(int p=0;p<b->N;p++) {
				bufs[p] = b->bufs[p]->buf.handle;
			}

			writeCBData *d = writeCBData::newAsOverflow(this,b);
			req->data = d;

			uv_write(req, (uv_stream_t *) &tty, bufs, b->N, write_overflow_cb);
		}
		void writeSync(const char *s, int l, const logMeta &m) override {
			uv_write_t *req = new uv_write_t;
			uv_buf_t buf;
			buf.base = const_cast<char *>(s);
			buf.len = l;
			uv_write(req, (uv_stream_t *) &tty, &buf, 1, NULL);
		}
		ttyTarget(int buffer_size, uint32_t id, GreaseLogger *o, targetReadyCB cb, delim_data _delim, target_start_info *readydata = NULL,  char *ttypath = NULL, uint32_t num_banks = NUM_BANKS)
		   : logTarget(buffer_size, id, o, cb, std::move(_delim), readydata, num_banks), ttyFD(0)  {
			_errcmn::err_ev err;

			if(ttypath) {
				HEAVY_DBG_OUT("TTY: at open(%s) ", ttypath);
				ttyFD = ::open(ttypath, O_WRONLY, 0);
			}
			else {
				HEAVY_DBG_OUT("TTY: at open(/dev/tty) ", ttypath);
				ttyFD = ::open("/dev/tty", O_WRONLY, 0);
			}

			HEAVY_DBG_OUT("TTY: past open() ");

			if(ttyFD >= 0) {
				tty.loop = o->loggerLoop;
				int r = uv_tty_init(o->loggerLoop, &tty, ttyFD, READABLE);

				if (r) ERROR_UV("initing tty", r, o->loggerLoop);

				// enable TYY formatting, flow-control, etc.
//				r = uv_tty_set_mode(&tty, TTY_NORMAL);
//				DBG_OUT("r = %d\n",r);
//				if (r) ERROR_UV("setting tty mode", r, o->loggerLoop);

				if(!r)
					readyCB(true,err,this);
				else {
					err.setError(r);
					readyCB(false,err, this);
				}
			} else {
				err.setError(errno,"Failed to open TTY");
				readyCB(false,err, this);
			}

		}

	protected:
		// make a TTY from an existing file descriptor
		ttyTarget(int fd, int buffer_size, uint32_t id, GreaseLogger *o, targetReadyCB cb, delim_data _delim, target_start_info *readydata = NULL)
		   : logTarget(buffer_size, id, o, cb, std::move(_delim), readydata), ttyFD(0)  {
//			outReq.cb = write_cb;
			_errcmn::err_ev err;
//			if(ttypath)
//				ttyFD = ::open(ttypath, O_WRONLY, 0);
//			else
//				ttyFD = ::open("/dev/tty", O_WRONLY, 0);
			ttyFD = fd;

			if(ttyFD >= 0) {
				tty.loop = o->loggerLoop;
				int r = uv_tty_init(o->loggerLoop, &tty, ttyFD, READABLE);

				if (r) ERROR_UV("initing tty", r, o->loggerLoop);

				// enable TYY formatting, flow-control, etc.
//				r = uv_tty_set_mode(&tty, TTY_NORMAL);
//				DBG_OUT("r = %d\n",r);
//				if (r) ERROR_UV("setting tty mode", r, o->loggerLoop);

				if(!r)
					readyCB(true,err,this);
				else {
					err.setError(r);
					readyCB(false,err, this);
				}
			} else {
				err.setError(errno,"Failed to open TTY");
				readyCB(false,err, this);
			}

		}
	public:
		static ttyTarget *makeTTYTargetFromFD(int fd, int buffer_size, uint32_t id, GreaseLogger *o, targetReadyCB cb, delim_data _delim, target_start_info *readydata = NULL) {
			return new  ttyTarget( fd, buffer_size, id, o, cb,  std::move(_delim), readydata );
		}
	};


	class fileTarget final : public logTarget {
	public:
		class rotationOpts final {
		public:
			bool enabled;
			bool rotate_on_start;     // rotate files on start (default false)
			uint32_t rotate_past;     // rotate when past rotate_past size in bytes, 0 - no setting
			uint32_t max_files;       // max files, including current, 0 - no setting
			uint64_t max_total_size;  // the max size to maintain, 0 - no setting
			uint32_t max_file_size;
			rotationOpts() : enabled(false), rotate_on_start(false), rotate_past(0), max_files(0), max_total_size(0), max_file_size(0) {}
			rotationOpts(rotationOpts& o) : enabled(o.enabled), rotate_on_start(o.rotate_on_start),
					rotate_past(o.rotate_past), max_files(o.max_files), max_total_size(o.max_total_size), max_file_size(o.max_file_size) {}
			rotationOpts& operator=(rotationOpts& o) {
				enabled = true;
				rotate_on_start = o.rotate_on_start; rotate_past = o.rotate_past;
				max_files = o.max_files; max_total_size = o.max_total_size;
				max_file_size = o.max_file_size;
				return *this;
			}
			bool validate() {
				if (max_files > MAX_ROTATED_FILES) {
					max_files = MAX_ROTATED_FILES;
				}
				return true;
			}
		};
	protected:


		// this tracks the number of submitted async write calls. When it's zero
		int submittedWrites; // ...it's safe to rotate
		rotationOpts filerotation;
		uint32_t current_size;
		uint64_t all_files_size;
	public:
		uv_rwlock_t wrLock;
		uv_file fileHandle;
		uv_fs_t fileFs;
		char *myPath;
		int myMode; int myFlags;

		static void on_open(uv_fs_t *req) {
			uv_fs_req_cleanup(req);
			fileTarget *t = (fileTarget *) req->data;
			_errcmn::err_ev err;
			if (req->result >= 0) {
				HEAVY_DBG_OUT("file: on_open() -> FD is %d", req->result);
				t->fileHandle = req->result;
				t->readyCB(true,err,t);
			} else {
				ERROR_OUT("Error opening file %s\n", t->myPath);
#if (UV_VERSION_MAJOR > 0)
				ERROR_UV("Error opening file.\n", req->result, NULL);
				err.setError(req->result);
#else
				err.setError(req->errorno);
#endif
				t->readyCB(false,err,t);
			}
		}

		static void on_open_for_rotate(uv_fs_t *req) {
			uv_fs_req_cleanup(req);
			fileTarget *t = (fileTarget *) req->data;
			_errcmn::err_ev err;
			if (req->result >= 0) {
				HEAVY_DBG_OUT("file: on_open() -> FD is %d", req->result);
				t->fileHandle = req->result;
//				t->readyCB(true,err,t);
			} else {
				ERROR_OUT("Error opening file %s\n", t->myPath);
#if (UV_VERSION_MAJOR > 0)
				ERROR_UV("Error opening file.\n", req->result, NULL);
				err.setError(req->result);
#else
				err.setError(req->errorno);
#endif
//				t->readyCB(false,err,t);
			}
		}

		static void check_and_maybe_rotate(fileTarget &ft, uint32_t *sz=nullptr) {
			uv_rwlock_wrlock(&ft.wrLock);
			ft.submittedWrites--;
			if(sz)
				ft.current_size += *sz;

			HEAVY_DBG_OUT("rotate: sub: %d\n",ft.submittedWrites);

			if(ft.submittedWrites < 1 && ft.filerotation.enabled) {
				HEAVY_DBG_OUT("rotate: current_size: %d\n",ft.current_size);
				if(ft.current_size > ft.filerotation.max_file_size) {
					HEAVY_DBG_OUT("rotate: past max file size\n");
					uv_fs_close(ft.owner->loggerLoop, &ft.fileFs, ft.fileHandle, NULL);
					HEAVY_DBG_OUT("rotate: now rotate_files()\n");
					ft.rotate_files();

					int r = uv_fs_open(ft.owner->loggerLoop, &ft.fileFs, ft.myPath, ft.myFlags, ft.myMode, NULL); // use default loop because we need on_open() cb called in event loop of node.js
#if (UV_VERSION_MAJOR > 0)
//					DBG_OUT("uv_fs_open: return was: %d : %d\n", r); // poorly documented, but as sync func, r is the FD
					if(r >= 0)
						on_open_for_rotate(&ft.fileFs); // and the FD is also in ft.fileFs struct.
#else
					if(r != -1)
						on_open_for_rotate(&ft.fileFs);
#endif
					else {
						ERROR_PERROR("Error opening log file: %s\n", errno, ft.myPath);
					}
				}
			}
			uv_rwlock_wrunlock(&ft.wrLock);
		}

		static void write_cb(uv_fs_t* req) {
			HEAVY_DBG_OUT("file: write_cb()");
#if (UV_VERSION_MAJOR > 0)
			if(req->result < 0) {
				ERROR_UV("file: write_cb() ", req->result, NULL);
			}
#else
			if(req->errorno) {
				ERROR_PERROR("file: write_cb() ",req->errorno);
			}
#endif
			writeCBData *d = (writeCBData *) req->data;
			fileTarget *ft = (fileTarget *) d->t;

			check_and_maybe_rotate(*ft);

			d->t->returnBuffer(d->b,false,d->nocallback);
			uv_fs_req_cleanup(req);
			delete req;
		}
		static void write_overflow_cb(uv_fs_t* req) {
			HEAVY_DBG_OUT("overflow_cb");
#if (UV_VERSION_MAJOR > 0)
			if(req->result < 0) {
				ERROR_UV("file: write_overflow_cb() ", req->result, NULL);
			}
#else
			if(req->errorno) {
				ERROR_PERROR("file: write_overflow_cb() ",req->errorno);
			}
#endif
			singleLog *b = (singleLog *) req->data;
			b->decRef();
			uv_fs_req_cleanup(req);

//			delete req;
		}

		uv_write_t outReq;  // since this function is no re-entrant (below) we can do this
		void flush(logBuf *b, bool nocallbacks = false) override { // this will always be called by the logger thread (via uv_async_send)
			uv_fs_t *req = new uv_fs_t;
			writeCBData *d = new writeCBData;
			d->t = this;
			d->b = b;
			d->nocallback = nocallbacks;
			req->data = d;
			uv_mutex_lock(&b->mutex);  // this lock should be ok, b/c write is async
			HEAVY_DBG_OUT("file: flush() %d bytes", b->handle.len);
			current_size += b->handle.len;
			// NOTE: we aren't using this like a normal wrLock. What it's for is to prevent
			// a file rotation while writing
			uv_rwlock_wrlock(&wrLock);
			submittedWrites++;
#if (UV_VERSION_MAJOR > 0)
			// libuv 1.x switch to a pwritev style function...
			uv_buf_t bz[1];
			bz[0] = b->handle;
			uv_fs_write(owner->loggerLoop, req, fileHandle, bz, 1, -1, write_cb);
#else
			uv_fs_write(owner->loggerLoop, req, fileHandle, (void *) b->handle.base, b->handle.len, -1, write_cb);
#endif

			uv_rwlock_wrunlock(&wrLock);
//			write_cb(&req);  // debug, skip actual write
			uv_mutex_unlock(&b->mutex);
//			b->clear(); // NOTE - there is a risk that this could get overwritten before it is written out.
			int n = uv_loop_alive(LOGGER->loggerLoop);
			DBG_OUT("loop_alive=%d\n",n);

		}

		int sync_n;
//		uv_fs_t reqSync[4];
		static void sync_cb(uv_fs_t* req) {
			uv_fs_req_cleanup(req);
#if (UV_VERSION_MAJOR > 0)
			HEAVY_DBG_OUT("file: sync() %d", req->result);
#else
			HEAVY_DBG_OUT("file: sync() %d", req->errorno);
#endif
			delete req;
		}
		void sync() override {
//			sync_n++;
//			if(sync_n >= 4) sync_n = 0;
			uv_fs_t *req = new uv_fs_t;
			req->data = this;
			uv_fs_fsync(owner->loggerLoop, req, fileHandle, sync_cb);
		}
		void flushSync(logBuf *b, bool nocallbacks = false) override { // this will always be called by the logger thread (via uv_async_send)
			uv_fs_t req;
			uv_mutex_lock(&b->mutex);  // this lock should be ok, b/c write is async
			HEAVY_DBG_OUT("file: flushSync() %d bytes", b->handle.len);
#if (UV_VERSION_MAJOR > 0)
			// libuv 1.x switch to a pwritev style function...
			uv_buf_t bz[1];
			bz[0] = b->handle;
			uv_fs_write(owner->loggerLoop, &req, fileHandle, bz, 1, -1, NULL);
#else
			uv_fs_write(owner->loggerLoop, &req, fileHandle, (void *) b->handle.base, b->handle.len, -1, NULL);
#endif
			check_and_maybe_rotate(*this);
			//			uv_write(req, (uv_stream_t *) &tty, &b->handle, 1, NULL);
			uv_mutex_unlock(&b->mutex);
			returnBuffer(b,true,nocallbacks);
			uv_fs_req_cleanup(&req);
//			b->clear(); // NOTE - there is a risk that this could get overwritten before it is written out.
		}

		void writeAsyncOverflow(overflowWriteOut *b,bool nocallbacks) override {
//			uv_fs_t *req = new uv_fs_t;
//			req->data = b;
			// APIUPDATE libuv - will change with newer libuv
			// uv_fs_write(uv_loop_t* loop, uv_fs_t* req, uv_file file, void* buf, size_t length, int64_t offset, uv_fs_cb cb);

//			uv_write_t *req = new uv_write_t;
//			req->data = b;
//
//			uv_buf_t bufs[MAX_OVERFLOW_BUFFERS];
//			for(int p=0;p<b->N;p++) {
//				bufs[p] = b->bufs[p]->buf.handle;
//			}

			// FIXME: in new version of libuv we need to just use uv_fs_write() which uses writev()

			struct iovec iov[MAX_OVERFLOW_BUFFERS];
			int need = 0;
			for(int p=0;p<b->N;p++) {
				if(b->bufs[p] && b->bufs[p]->buf.handle.base) {
					iov[p].iov_base = b->bufs[p]->buf.handle.base;
					iov[p].iov_len = b->bufs[p]->buf.handle.len;
					need += b->bufs[p]->buf.handle.len;
				}
			}

			uv_rwlock_wrlock(&wrLock);
			submittedWrites++;
			uv_rwlock_wrunlock(&wrLock);

			int got = ::writev(fileHandle, iov, b->N);

			if(got < 0) {
				ERROR_PERROR("file: write_overflow_cb() ", errno);
				uv_rwlock_wrlock(&wrLock);
				submittedWrites--;
				uv_rwlock_wrunlock(&wrLock);
			} else {
				if(got < need) {
					HEAVY_DBG_OUT("file target: writev() only wrote %d bytes. whatev. dropping it.", got);
				}
				check_and_maybe_rotate(*this, (uint32_t *) &got);
			}



			// if is callback, then do that

			returnBuffer(b,false,nocallbacks);

//			// else delete
//			delete b;

//			uv_fs_write( owner->loggerLoop, req, fileHandle, (uv_stream_t *) &tty, bufs, b->N, write_overflow_cb);
//
//
//			// TODO - make format string first...
//			uv_fs_write(owner->loggerLoop, req, fileHandle, (void *) b->buf.handle.base, b->buf.handle.len, -1, write_overflow_cb);
		}
		void writeSync(const char *s, int l, const logMeta &m) override {
			uv_fs_t req;
			uv_buf_t buf;
			buf.base = const_cast<char *>(s);
			buf.len = l;
#if (UV_VERSION_MAJOR > 0)
			// libuv 1.x switch to a pwritev style function...
			uv_buf_t bz[1];
			bz[0] = buf;
			uv_fs_write(owner->loggerLoop, &req, fileHandle, bz, 1, -1, NULL);
#else
			uv_fs_write(owner->loggerLoop, &req, fileHandle, (void *) s, l, -1, NULL);
#endif
			current_size += l;

			//			uv_write(req, (uv_stream_t *) &tty, &buf, 1, NULL);
		}
	protected:


		class rotatedFile final {
		public:
			char *path;
			uint32_t size;
			uint32_t num;
			bool ownMem;
			rotatedFile(char *base, int n) : path(NULL), size(0), ownMem(false), num(n) {
				path = rotatedFile::strRotateName(base,n);
				ownMem = true;
			}
			rotatedFile(internalCmdReq &) = delete;
			rotatedFile(rotatedFile && o) : path(o.path), size(o.size), num(o.num), ownMem(true) {
				o.path = NULL;
				o.ownMem = false;
			}
			rotatedFile() : path(NULL), size(0), num(0), ownMem(false) {};
			rotatedFile& operator=(rotatedFile& o) {
				if(path && ownMem)
					LFREE(path);
				path = o.path;
				size = o.size;
				num = o.num;
				ownMem = false;
				return *this;
			}
			rotatedFile& operator=(rotatedFile&& o) {
				if(path && ownMem)
					LFREE(path);
				path = o.path; o.path = NULL;
				size = o.size;
				num = o.num;
				ownMem = o.ownMem; o.ownMem = false;
				return *this;
			}
			~rotatedFile() {
				if(path && ownMem)
					LFREE(path);
			}
			static char *strRotateName(char *s, int n) {
				char *ret = NULL;
				if(s && n > 0) { // n < MAX_ROTATED_FILES &&
					int origL = strlen(s);
					ret = (char *) LMALLOC(origL+10);
					memset(ret,0,origL+10);
					memcpy(ret,s,origL);
					snprintf(ret+origL,9,".%d",n);
				}
				return ret;
			}
		};

		uv_mutex_t rotateFileMutex;
		TWlib::tw_safeCircular<rotatedFile, LoggerAlloc > rotatedFiles;

		void rotate_files() {
			rotatedFile f;
			uv_mutex_lock(&rotateFileMutex);  // not needed
			TWlib::tw_safeCircular<rotatedFile, LoggerAlloc > tempFiles(MAX_ROTATED_FILES,true);
			int n = rotatedFiles.remaining();
			while (rotatedFiles.removeMv(f)) {
				uv_fs_t req;
				if(filerotation.max_files && ((n+1) > filerotation.max_files)) { // remove file
					if(!f.path) {
						ERROR_OUT("file rotation - internal error - NULL path\n");
					} else {
						int r = uv_fs_unlink(owner->loggerLoop,&req,f.path,NULL);
						DBG_OUT("rotate_files::::::::::::::::: fs_unlink %s\n",f.path);
#if (UV_VERSION_MAJOR > 0)
						if(r < 0) {
							if(r != UV_ENOENT) {
								ERROR_OUT("file rotation - remove %s: %s\n", f.path, uv_strerror(r));
							}
						}
#else
						if(r) {
							uv_err_t e = uv_last_error(owner->loggerLoop);
							if(e.code != UV_ENOENT) {
								ERROR_OUT("file rotation - remove %s: %s\n", f.path, uv_strerror(e));
							}
						}
#endif
					}
				} else {  // move file to new name
					if (!f.path) {
						ERROR_OUT("file rotation - internal error (2) - NULL path\n");
					} else {
//						char *newname = rotatedFile::strRotateName(myPath,f.num+1);
						rotatedFile new_f(myPath,f.num+1);
						DBG_OUT("rotate_files::::::::::::::::: fs_rename\n");
						int r = uv_fs_rename(owner->loggerLoop, &req, f.path, new_f.path, NULL);
#if (UV_VERSION_MAJOR > 0)
						if(r < 0) {
							ERROR_OUT("file rotation - rename %s: %s\n", f.path, uv_strerror(r));
#else
						if(r) {
							uv_err_t e = uv_last_error(owner->loggerLoop);
							if(e.code != UV_OK) {
								ERROR_OUT("file rotation - rename %s: %s\n", f.path, uv_strerror(e));
							}
#endif
//							if(e.code != UV_ENOENT) {
//								ERROR_OUT("file rotation - rename %s: %s\n", f.path, uv_strerror(e));
//							}
						}
						tempFiles.addMv(new_f);
					}
				}
				n--;
			}
			uv_fs_t req;
			rotatedFile new_f(myPath,1);

			int r = uv_fs_rename(owner->loggerLoop, &req, myPath, new_f.path, NULL);
#if (UV_VERSION_MAJOR > 0)
			if(r < 0) {
				ERROR_OUT("file rotation - rename %s: %s\n", new_f.path, uv_strerror(r));
#else
			if(r) {
				uv_err_t e = uv_last_error(owner->loggerLoop);
				if(e.code != UV_OK) {
					ERROR_OUT("file rotation - rename %s: %s\n", new_f.path, uv_strerror(e));
				}
#endif
			} else
				tempFiles.addMv(new_f);
			rotatedFiles.transferFrom(tempFiles);
			current_size = 0; // we will use a new file, size is 0
			uv_mutex_unlock(&rotateFileMutex);
		}

		bool check_files() {
			all_files_size = 0;
			// find all rotated files
			bool needs_rotation = false;
			uv_fs_t req;
			rotatedFiles.clear();
			HEAVY_DBG_OUT("rotation: check_files(%s)\n",myPath);
			// find existing file...
			int r = uv_fs_stat(owner->loggerLoop, &req, myPath, NULL);
#if (UV_VERSION_MAJOR > 0)
			if(r < 0) {
				ERROR_OUT("file rotation: %s\n", uv_strerror(r));
#else
			if(r) {
				uv_err_t e = uv_last_error(owner->loggerLoop);
				if(e.code != UV_ENOENT) {
					ERROR_OUT("file rotation: %s\n", uv_strerror(e));
				}
#endif
			} else {
				// if the current file is too big - we need to rotate.
				HEAVY_DBG_OUT("rotation: current file %s is %d bytes\n",myPath,req.statbuf.st_size);
				if(filerotation.max_file_size && filerotation.max_file_size < req.statbuf.st_size) {
					needs_rotation = true;
					HEAVY_DBG_OUT("rotation: current files needs rotation.\n");					
				}

				all_files_size += req.statbuf.st_size;
				current_size = req.statbuf.st_size;
			}

			int n = filerotation.max_files;
			while(n > 0) {  // find all files...
				uv_fs_t req;
				rotatedFile f(myPath,n);
				if(!f.path) {
					HEAVY_DBG_OUT("rotation: f.path NULL - should not happen\n");
					break; // either something went wrong, or we are past the max rotation of files
				}
				HEAVY_DBG_OUT("file rotation: fs_stat %s\n",f.path);
				int r = uv_fs_stat(owner->loggerLoop, &req, f.path, NULL);
#if (UV_VERSION_MAJOR > 0)
				if(r < 0) {
					if(r != UV_ENOENT) {
						ERROR_OUT("file rotation: [path:%s]: %s\n", f.path, uv_strerror(r));
					}
//					break;
#else
				if(r) {
					uv_err_t e = uv_last_error(owner->loggerLoop);
					if(e.code && e.code != UV_ENOENT) {
						ERROR_OUT("file rotation [path:%s]: %s\n", f.path, uv_strerror(e));
					}
					HEAVY_DBG_OUT("rotation: end check_files()\n")
					break;
#endif
				} else {
					f.size = req.statbuf.st_size;
					HEAVY_DBG_OUT("rotation: found file: %s of size %d\n",f.path,f.size);
					rotatedFiles.addMv(f);
					all_files_size += req.statbuf.st_size;

				}
				n--;
			}
			HEAVY_DBG_OUT("rotation: all_files_size: %d\n",all_files_size);
//			rotatedFiles.reverse();
			if(all_files_size > filerotation.max_total_size)
				needs_rotation = true;
			return needs_rotation;
		}


		void post_cstor(int buffer_size, uint32_t id, GreaseLogger *o, int flags, int mode, char *path, target_start_info *readydata, targetReadyCB cb) {
			uv_mutex_init(&rotateFileMutex);
			uv_rwlock_init(&wrLock);
			readydata->needsAsyncQueue = true;
			myPath = local_strdup_safe(path);
			if(!path) {
				ERROR_OUT("Need a file path!!");
				_errcmn::err_ev err;
				err.setError(GREASE_UNKNOWN_NO_PATH);
				readyCB(false,err,this);
			} else {
				filerotation.validate();
				fileFs.data = this;

				if(filerotation.enabled) {
					bool needs_rotate = check_files();
					HEAVY_DBG_OUT("rotate_files: total files = %d total size = %d\n",rotatedFiles.remaining()+1,all_files_size);
					if(needs_rotate || filerotation.rotate_on_start) {
						HEAVY_DBG_OUT("Needs rotate....\n");
						if (filerotation.rotate_on_start) {
							// ok - need to preload the list of existing log files.							
						}
						rotate_files();
					}


					// check for existing rotated files, based on path name
					// mv existing file to new name, if rotateopts says so
					// setup rotatedFiles array
				}

				int r = uv_fs_open(owner->loggerLoop, &fileFs, path, flags, mode, on_open); // use default loop because we need on_open() cb called in event loop of node.js
#if (UV_VERSION_MAJOR > 0)
				if (r < 0) ERROR_UV("initing file", r, NULL);
#else
				if (r) ERROR_UV("initing file", r, owner->loggerLoop);
#endif
			}
		}

	public:
		fileTarget(int buffer_size, uint32_t id, GreaseLogger *o, int flags, int mode, char *path, delim_data _delim, target_start_info *readydata, targetReadyCB cb,
				rotationOpts rotateopts, uint32_t num_banks = NUM_BANKS) :
				logTarget(buffer_size, id, o, cb, std::move(_delim), readydata, num_banks),
				submittedWrites(0), rotatedFiles(MAX_ROTATED_FILES,true), current_size(0), all_files_size(0),
				myPath(NULL), myMode(mode), myFlags(flags), filerotation(rotateopts), sync_n(0) {
			post_cstor(buffer_size, id, o, flags, mode, path, readydata, cb);
		}

		fileTarget(int buffer_size, uint32_t id, GreaseLogger *o, int flags, int mode, char *path, delim_data _delim, target_start_info *readydata, targetReadyCB cb, uint32_t num_banks = NUM_BANKS) :
			logTarget(buffer_size, id, o, cb, std::move(_delim), readydata, num_banks),
			submittedWrites(0), rotatedFiles(MAX_ROTATED_FILES,true), current_size(0), all_files_size(0),
			myPath(NULL), myMode(mode), myFlags(flags), filerotation(), sync_n(0) {
			post_cstor(buffer_size, id, o, flags, mode, path, readydata, cb);
		}


	};



	class callbackTarget final : public logTarget {
	public:
		callbackTarget(int buffer_size, uint32_t id, GreaseLogger *o,
				targetReadyCB cb, delim_data _delim, target_start_info *readydata, uint32_t num_banks = NUM_BANKS) :
					logTarget(buffer_size, id, o, cb, std::move(_delim), readydata, num_banks)
	{
			_errcmn::err_ev err;
			if(cb) cb(true,err,this);
	}

		callbackTarget() = delete;

		// 			d->t->returnBuffer(d->b);

		void flush(logBuf *b,bool nocallbacks = false) override {
			returnBuffer(b,false,nocallbacks);
		}; // flush buffer 'n'. This is ansynchronous
		void flushSync(logBuf *b, bool nocallbacks = false) override {
			returnBuffer(b,true,nocallbacks);
		}; // flush buffer 'n'. This is ansynchronous
		void writeAsyncOverflow(overflowWriteOut *b,bool nocallbacks) override {
			int n = 0;
			if(b->N < 1) {
				delete b;
				return;
			}
			if(!nocallbacks && logCallbackSet && b) {
				for(int p=0; p<b->N;p++) {
					n += b->bufs[p]->buf.handle.len;
				}

//				logBuf *outbuf = new logBuf(n,0,std::move(delim));
//				// call returnBufferOverflow
//				for(int p=0; p<b->N-1;p++) {
//					outbuf->copyIn(b->bufs[p]->buf.handle.base,b->bufs[p]->buf.handle.len,false);
//				}
//				outbuf->copyIn(b->bufs[b->N-1]->buf.handle.base,b->bufs[b->N-1]->buf.handle.len,true);
//
//

				writeCBData cbdat;
				cbdat.t = this;
				cbdat.overflow = b;

				if(!owner->callerLogCallbacks.addMvIfRoom(cbdat)) {
					if(owner->Opts.show_errors)
						ERROR_OUT(" !!! v8LogCallbacks is full! Can't rotate. Callback will be skipped.");
				}
				uv_async_send(&owner->asyncV8LogCallback);
			}
		};
		void writeSync(const char *s, int len) {
			// call v8 callback now
		}; // flush buffer 'n'. This is synchronous. Writes now - skips buffering

	};

	static void targetReady(bool ready, _errcmn::err_ev &err, logTarget *t);

	TWlib::tw_safeCircular<GreaseLogger::internalCmdReq, LoggerAlloc > internalCmdQueue;
	TWlib::tw_safeCircular<GreaseLogger::nodeCmdReq, LoggerAlloc > nodeCmdQueue;
	TWlib::tw_safeCircular<GreaseLogger::logTarget::writeCBData, LoggerAlloc > callerLogCallbacks; // formerly v8LogCallbacks

	uv_mutex_t modifyFilters; // if the table is being modified, lock first
	typedef TWlib::TW_KHash_32<uint64_t, FilterList *, TWlib::TW_Mutex, uint64_t_eqstrP, TWlib::Allocator<LoggerAlloc> > FilterHashTable;
	typedef TWlib::TW_KHash_32<uint32_t, Filter *, TWlib::TW_Mutex, uint32_t_eqstrP, TWlib::Allocator<LoggerAlloc> > FilterTable;
	typedef TWlib::TW_KHash_32<TargetId, logTarget *, TWlib::TW_Mutex, TargetId_eqstrP, TWlib::Allocator<LoggerAlloc> > TargetTable;
	typedef TWlib::TW_KHash_32<uint32_t, Sink *, TWlib::TW_Mutex, uint32_t_eqstrP, TWlib::Allocator<LoggerAlloc> > SinkTable;


	FilterHashTable filterHashTable;  // look Filters by tag:origin

	bool sift(logMeta &f); // returns true, then the logger should log it	TWlib::TW_KHash_32<uint16_t, int, TWlib::TW_Mutex, uint16_t_eqstrP, TWlib::Allocator<TWlib::Alloc_Std>  > magicNumTable;
	bool siftP(logMeta *f); // returns true, then the logger should log it	TWlib::TW_KHash_32<uint16_t, int, TWlib::TW_Mutex, uint16_t_eqstrP, TWlib::Allocator<TWlib::Alloc_Std>  > magicNumTable;
//	uint32_t levelFilterOutMask;  // mandatory - all log messages have a level. If the bit is 1, the level will be logged.
//
//	bool defaultFilterOut;


	// http://stackoverflow.com/questions/1277627/overhead-of-pthread-mutexes
	uv_mutex_t modifyTargets; // if the table is being modified, lock first

	logTarget *getTarget(uint32_t key) {
		logTarget *ret = NULL;
		uv_mutex_lock(&modifyTargets);
		targets.find(key,ret);
		uv_mutex_unlock(&modifyTargets);
		return ret;
	}

	logTarget *defaultTarget;
	Filter defaultFilter;
	TargetTable targets;
	SinkTable sinks;

	LabelTable tagLabels;
	LabelTable originLabels;
	LabelTable levelLabels;


	uv_async_t asyncInternalCommand;
	uv_async_t asyncExternalCommand;
	uv_timer_t flushTimer;

#if (UV_VERSION_MAJOR > 0)
	static void handleInternalCmd(uv_async_t *handle); // from in the class
	static void handleExternalCmd(uv_async_t *handle); // from node
	static void flushTimer_cb(uv_timer_t* handle);  // on a timer. this flushed buffers after a while
#else
	static void handleInternalCmd(uv_async_t *handle, int status /*UNUSED*/); // from in the class
	static void handleExternalCmd(uv_async_t *handle, int status /*UNUSED*/); // from node
	static void flushTimer_cb(uv_timer_t* handle, int status);  // on a timer. this flushed buffers after a while
#endif
	static void mainThread(void *);

	void startFlushTimer();
	void stopFlushTimer();

	void setupDefaultTarget(actionCB cb, target_start_info *data);

	inline FilterHash filterhash(TagId tag, OriginId origin) {
		// http://stackoverflow.com/questions/13158501/specifying-64-bit-unsigned-integer-literals-on-64-bit-data-models
		return (uint64_t) (((uint64_t) UINT64_C(0xFFFFFFFF00000000) & ((uint64_t) origin << 32)) | tag);
	}

	inline void getHashes(TagId tag, OriginId origin, FilterHash *hashes) {
		hashes[0] = filterhash(tag,origin);
		hashes[1] = hashes[0] & UINT64_C(0xFFFFFFFF00000000);
		hashes[2] = hashes[0] & UINT64_C(0x00000000FFFFFFFF);
	}

	void _findOrAddFilterTable(OriginId origin, TagId tag) {
		uv_mutex_lock(&nextIdMutex);
		uv_mutex_unlock(&nextIdMutex);

	}

	bool _addFilter(TargetId t, OriginId origin, TagId tag, LevelMask level, FilterId &id,
			logLabel *preformat = nullptr, logLabel *postformatpremsg = nullptr, logLabel *postformat = nullptr) {
		uint64_t hash = filterhash(tag,origin);
		bool ret = false;
		uv_mutex_lock(&nextIdMutex);
		id = nextFilterId++;
		uv_mutex_unlock(&nextIdMutex);

		FilterList *list = NULL;
		uv_mutex_lock(&modifyFilters);
		if(!filterHashTable.find(hash,list)) {
			list = new FilterList();
			DBG_OUT("new FilterList: hash %llu", hash);
			filterHashTable.addNoreplace(hash, list);
		}
		Filter f(id,level,t);
		if(preformat && !preformat->empty())
			f.preFormat = *preformat;
		if(postformatpremsg && !postformatpremsg->empty())
			f.postFormatPreMsg = *postformatpremsg;
		if(postformat && !postformat->empty())
			f.postFormat = *postformat;

		DBG_OUT("new Filter(%x,%x,%x) to list [hash] %llu", id,level,t, hash);
		// TODO add to list
		ret = list->add(f);
		uv_mutex_unlock(&modifyFilters);
		if(!ret) {
			ERROR_OUT("Max filters for this tag/origin combination");
		}
		return ret;
	}

	bool _lookupFilter(OriginId origin, TagId tag, FilterId id, Filter *&filter) {
		uint64_t hash = filterhash(tag,origin);
		bool ret = false;

		FilterList *list = NULL;
		Filter *found = NULL;

		if(filterHashTable.find(hash,list)) {
			list->find(id,found);
		}

		if(found) {
			filter = found;
			ret = true;
		}
		return ret;
	}



	uv_loop_t *loggerLoop;  // grease uses its own libuv event loop (not node's)

	int _log(const logMeta &meta, const char *s, int len); // internal log cmd
	int _logSync(const logMeta &meta, const char *s, int len); // internal log cmd

	// these two function should be used in close proximity
	int _grabInLogBuffer(singleLog* &buf);
	int _submitBuffer(singleLog *buf);
	int _returnBuffer(singleLog *buf);

	void start(actionCB cb, target_start_info *data);
	int logFromRaw(char *base, int len);


// used for parsing syslogs
	enum syslog_parse_state {
		SYSLOG_BEGIN,
		SYSLOG_IN_FAC,
		SYSLOG_IN_DATE_MO,
		SYSLOG_IN_DATE_DAY,
		SYSLOG_IN_DATE_TIME,
		SYSLOG_IN_MESSAGE,
		SYSLOG_INVALID,
		SYSLOG_END_LOG
	};
	static bool parse_single_syslog_to_singleLog(char *start, int &remain, syslog_parse_state &begin_state, singleLog *entry); //, char *&moved);

// used for parsing kernel logs
	enum klog_parse_state {
		LEVEL_BEGIN,  // <
		IN_LEVEL,     // 0-0
//			LEVEL_END,    // >
		TIME_STAMP_BEGIN,   // [
		IN_TIME_STAMP,      // 0-9, .  // not captured
//			TIME_STAMP_END,     // ]
		BODY_BEGIN,         // white space
		IN_BODY,            // anything but newlines
		BODY_END,            // \n
		CONT_BODY_BEGIN,     // continuation body.. white space
		IN_CONT_BODY,        // anything but newlines
		CONT_BODY_END,       // \n
		INVALID,             // bad parse - in whcih case we will just put the whole thing in the logger
		END_LOG              //
	};
	static bool parse_single_klog_to_singleLog(char *start, int &remain, klog_parse_state &begin_state, singleLog *entry, char *&moved);
	static bool parse_single_devklog_to_singleLog(char *start, int &remain, klog_parse_state &begin_state, singleLog *entry, char *&moved);

public:
	int log(const logMeta &f, const char *s, int len); // does the work of logging (for users in C++)
	int logP(logMeta *f, const char *s, int len); // does the work of logging (for users in C++)
	int logSync(const logMeta &f, const char *s, int len); // does the work of logging. now. will empty any buffers first.
	static void flushAll(bool nocallbacks = false);
	static void flushAllSync(bool rotate = true,bool nocallbacks = false);

#ifndef GREASE_LIB
	static void Init(v8::Local<v8::Object> exports);

    static NAN_METHOD(New);
    static NAN_METHOD(NewIwnstance);

    static NAN_METHOD(SetGlobalOpts);

    static NAN_METHOD(AddTagLabel);
    static NAN_METHOD(AddOriginLabel);
    static NAN_METHOD(AddLevelLabel);


    static NAN_METHOD(FilterOut);
    static NAN_METHOD(FilterIn);
    static NAN_METHOD(AddFilter);
    static NAN_METHOD(RemoveFilter);
    static NAN_METHOD(ModifyFilter);

    static NAN_METHOD(AddTarget);
    static NAN_METHOD(AddSink);

    static NAN_METHOD(ModifyDefaultTarget);

    static NAN_METHOD(Start);

    // creates a new pseduo-terminal for use with TTY targets
    static NAN_METHOD(createPTS);

    static NAN_METHOD(DisableTarget);
    static NAN_METHOD(EnableTarget);


    static NAN_METHOD(Log);
    static NAN_METHOD(LogSync);
    static NAN_METHOD(Flush);

    static Nan::Persistent<v8::Function> constructor;
#endif
//    static Persistent<ObjectTemplate> prototype;

    // solid reference to TUN / TAP creation is: http://backreference.org/2010/03/26/tuntap-interface-tutorial/


	void setErrno(int _errno, const char *m=NULL) {
		err.setError(_errno, m);
	}

protected:

	// somewhat analagous to the above NAN_METHOD functions for V8
#ifdef GREASE_LIB
	LIB_METHOD_FRIEND(setGlobalOpts, GreaseLibOpts *opts);
	LIB_METHOD_FRIEND(start);
	LIB_METHOD_FRIEND(shutdown);
	LIB_METHOD_SYNC_FRIEND(addOriginLabel, uint32_t val, const char *utf8, int len);
	LIB_METHOD_SYNC_FRIEND(addTagLabel, uint32_t val, const char *utf8, int len);
	LIB_METHOD_SYNC_FRIEND(addLevelLabel, uint32_t val, const char *utf8, int len);
	LIB_METHOD_SYNC_FRIEND(modifyDefaultTarget,GreaseLibTargetOpts *opts);
	LIB_METHOD_FRIEND(addTarget,GreaseLibTargetOpts *opts);
	LIB_METHOD_SYNC_FRIEND(maskOutByLevel, uint32_t val);
	LIB_METHOD_SYNC_FRIEND(unmaskOutByLevel, uint32_t val);
	LIB_METHOD_SYNC_FRIEND(addFilter,GreaseLibFilter *filter);
	LIB_METHOD_SYNC_FRIEND(disableFilter,GreaseLibFilter *filter);
	LIB_METHOD_SYNC_FRIEND(enableFilter,GreaseLibFilter *filter);
	LIB_METHOD_SYNC_FRIEND(addSink,GreaseLibSink *sink);
	LIB_METHOD_SYNC_FRIEND(disableTarget, TargetId id);
	LIB_METHOD_SYNC_FRIEND(enableTarget, TargetId id);
	LIB_METHOD_SYNC_FRIEND(flush, TargetId id);
	friend void ::_greaseLib_handle_stdoutFd_cb(uv_poll_t *handle, int status, int events);
	friend void ::_greaseLib_handle_stderrFd_cb(uv_poll_t *handle, int status, int events);
	friend GreaseLibTargetOpts* ::GreaseLib_new_GreaseLibTargetOpts(void);
	friend GreaseLibTargetOpts* ::GreaseLib_init_GreaseLibTargetOpts(GreaseLibTargetOpts *);
	friend void ::GreaseLib_cleanup_GreaseLibBuf(GreaseLibBuf *b);
#endif


	// calls callbacks when target starts (if target was started in non-v8 thread
	static void callTargetCallback(uv_async_t *h);
	static void callV8LogCallbacks(uv_async_t *h);
#ifndef GREASE_LIB
	static void _doV8Callback(GreaseLogger::logTarget::writeCBData &data); // <--- only call this one from v8 thread!
#else
	static void _doLibCallback(GreaseLogger::logTarget::writeCBData &data); // <--- only call this one from v8 thread!
	static void returnBufferToTarget(GreaseLibBuf *buf);
#endif
	static void start_target_cb(GreaseLogger *l, _errcmn::err_ev &err, void *d);
	static void start_logger_cb(GreaseLogger *l, _errcmn::err_ev &err, void *d);
    GreaseLogger(int buffer_size = DEFAULT_BUFFER_SIZE, int chunk_size = LOGGER_DEFAULT_CHUNK_SIZE, uv_loop_t *userloop = NULL) :
    	nextOptsId(1), nextFilterId(1), nextTargetId(1),
    	Opts(),
//    	bufferSize(buffer_size), chunkSize(chunk_size),
    	masterBufferAvail(PRIMARY_BUFFER_ENTRIES),
	    needV8(0),
	    needGreaseThread(0),
    	targetCallbackQueue(MAX_TARGET_CALLBACK_STACK, true),
    	err(),
    	internalCmdQueue( INTERNAL_QUEUE_SIZE, true ),
    	nodeCmdQueue( COMMAND_QUEUE_NODE_SIZE, true ),
    	callerLogCallbacks( V8_LOG_CALLBACK_QUEUE_SIZE, true ),
//    	levelFilterOutMask(0), // moved to Opts ... tagFilter(), originFilter(),
//    	defaultFilterOut(false),
    	filterHashTable(),
    	defaultTarget(NULL),
    	defaultFilter(),
    	targets(),
    	sinks(),
    	tagLabels(), originLabels(), levelLabels(),
	    loggerLoop(NULL),
		nextSinkId(0)
    	{
    	   	LOGGER = this;
    	    loggerLoop = uv_loop_new();    // we use our *own* event loop (not the node/io.js one)
    	    Opts.bufferSize = buffer_size;
    	    Opts.chunkSize = chunk_size;
    	    if(!userloop) userloop = uv_default_loop();
    	    uv_async_init(userloop, &asyncTargetCallback, callTargetCallback);
    	    uv_async_init(userloop, &asyncV8LogCallback, callV8LogCallbacks);
    	    uv_async_init(userloop, &asyncRefLogger, refCb_Logger);
    	    uv_async_init(userloop, &asyncUnrefLogger, unrefCb_Logger);
    	    uv_unref((uv_handle_t *)&asyncV8LogCallback);
    	    uv_unref((uv_handle_t *)&asyncTargetCallback);
    	    uv_unref((uv_handle_t *)&asyncRefLogger);
    	    uv_unref((uv_handle_t *)&asyncUnrefLogger);
    	    uv_mutex_init(&mutexRefLogger);
    	    uv_mutex_init(&nextIdMutex);
    	    uv_mutex_init(&nextOptsIdMutex);
    		uv_mutex_init(&modifyFilters);
    		uv_mutex_init(&modifyTargets);

    		for(int n=0;n<PRIMARY_BUFFER_ENTRIES;n++) {
    			singleLog *l = new singleLog(buffer_size);
    			masterBufferAvail.add(l);
    		}
    	}

    ~GreaseLogger() {
    	if(defaultTarget) delete defaultTarget;
    	TargetTable::HashIterator iter(targets);
    	logTarget **t; // shut down other targets.
    	while(!iter.atEnd()) {
    		t = iter.data();
    		delete (*t);
    		iter.getNext();
    	}
    	singleLog *l = NULL;
    	while(masterBufferAvail.remove(l)) {
    		delete l;
    	}
    }

public:
	static GreaseLogger *setupClass(int buffer_size = DEFAULT_BUFFER_SIZE, int chunk_size = LOGGER_DEFAULT_CHUNK_SIZE, uv_loop_t *userloop = NULL) {
		if(!LOGGER) {
			LOGGER = new GreaseLogger(buffer_size, chunk_size, userloop);
			atexit(shutdown);
		}
		return LOGGER;
	}


	static void shutdown() {
		GreaseLogger *l = setupClass();
		internalCmdReq req(INTERNAL_SHUTDOWN); // just rotated off a buffer
		l->internalCmdQueue.addMvIfRoom(req);
		uv_async_send(&l->asyncInternalCommand);
	}
};




} // end namepsace


#endif /* GreaseLogger_H_ */
