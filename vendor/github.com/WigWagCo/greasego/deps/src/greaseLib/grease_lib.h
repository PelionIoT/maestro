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

/*
 * GreaseLogger.h
 *
 * greaselib bindings
 * launch the grease logging process via these library calls
 *
 *  Created on: Nov 23, 2016
 *      Author: ed
 * (c) 2016, WigWag Inc
 */


#ifndef GreaseLib_H_
#define GreaseLib_H_
#include "grease_client.h"
#include <uv.h>
#include <gperftools/tcmalloc.h>
#include <stdint.h>

// typically defined in <linux/limits.h>
#define GREASE_PATH_MAX        4096

#define GREASE_LIB_OK 0
#define GREASE_LIB_NOT_FOUND      0x01E00000
#define GREASE_LIB_INTERNAL_ERROR 0x01E00001

#define LIB_METHOD( name, ... ) int GreaseLib_##name( GreaseLibCallback libCB, ## __VA_ARGS__ )
#define LIB_METHOD_FRIEND( name, ... ) friend int ::GreaseLib_##name( GreaseLibCallback libCB, ## __VA_ARGS__ )

#define LIB_METHOD_SYNC( name, ... ) int GreaseLib_##name( __VA_ARGS__ )
#define LIB_METHOD_SYNC_FRIEND( name, ... ) friend int ::GreaseLib_##name( __VA_ARGS__ )

#ifdef __cplusplus
extern "C" {
#endif

#define GREASE_LIB_MAX_ERR_STRING 256

typedef struct {
	char errstr[GREASE_LIB_MAX_ERR_STRING];
	int _errno;
} GreaseLibError;
typedef void (*GreaseLibCallback) (GreaseLibError *, void *);
typedef void (*GreaseLibTargetCallback) (GreaseLibError *, void *, uint32_t targId);

typedef struct {
	char *data;
	uint64_t size;
	int own;    // if > 0 then cleanup call will free this memory
	int _id; // used internally (sometimes) - should not be changed
	void *_shadow; // this is the original C++ GreaseLogger::lohBuf object, which we can't have a proper point to - so we just do this
} GreaseLibBuf;

void GreaseLib_getVersion(char *s, int len);
void GreaseLib_init_GreaseLibBuf(GreaseLibBuf *b);
GreaseLibBuf *GreaseLib_new_GreaseLibBuf(size_t l);
void GreaseLib_cleanup_GreaseLibBuf(GreaseLibBuf *b); // should be called when the callback is done using the buffer it was handed
GreaseLibBuf *_greaseLib_new_empty_GreaseLibBuf();

#define GREASE_BOOL int
typedef struct {
	uint32_t LevelFilterOut;
	GREASE_BOOL defaultFilterOut;
	GREASE_BOOL show_filters;
	GREASE_BOOL callback_errors;
	GREASE_BOOL show_errors;
} GreaseLibOpts;

LIB_METHOD(setGlobalOpts, GreaseLibOpts *opts);
LIB_METHOD(start);
LIB_METHOD(shutdown);
void GreaseLib_waitOnGreaseShutdown();
LIB_METHOD_SYNC(addTagLabel, uint32_t val, const char *utf8, int len);
LIB_METHOD_SYNC(addOriginLabel, uint32_t val, const char *utf8, int len);
LIB_METHOD_SYNC(addLevelLabel, uint32_t val, const char *utf8, int len);
LIB_METHOD_SYNC(maskOutByLevel, uint32_t val);
LIB_METHOD_SYNC(unmaskOutByLevel, uint32_t val);

LIB_METHOD_SYNC(setupStandardLevels);
LIB_METHOD_SYNC(setupStandardTags);

LIB_METHOD_SYNC(getUnusedOriginId, uint32_t *val);
LIB_METHOD_SYNC(getUnusedTagId, uint32_t *val);

// for internal logging:
LIB_METHOD_SYNC(logCharBuffer, logMeta *f, const char *utf8, int len);

void _greaseLib_handle_stdoutFd_cb(uv_poll_t *handle, int status, int events);
void _greaseLib_handle_stderrFd_cb(uv_poll_t *handle, int status, int events);


#define GREASE_LIB_SET_FILEOPTS_MODE           0x10000000
#define GREASE_LIB_SET_FILEOPTS_FLAGS          0x20000000
#define GREASE_LIB_SET_FILEOPTS_MAXFILES       0x40000000
#define GREASE_LIB_SET_FILEOPTS_MAXFILESIZE    0x80000000
#define GREASE_LIB_SET_FILEOPTS_MAXTOTALSIZE   0x01000000
#define GREASE_LIB_SET_FILEOPTS_ROTATEONSTART  0x02000000  // set if you want files to rotate on start
#define GREASE_LIB_SET_FILEOPTS_ROTATE         0x04000000  // set if you want files to rotate, if not set all other rotate options are skipped

typedef struct {
	uint32_t _enabledFlags;
	uint32_t mode;           // permissions for file (recommend default)
	uint32_t flags;          // file flags (recommend default)
	uint32_t max_files;      // max # of files to maintain (rotation)
	uint32_t max_file_size;  // max size for any one file
	uint64_t max_total_size; // max total size to maintain in rotation
} GreaseLibTargetFileOpts;

typedef struct {
	char *delim;             // points to delimetter UTF8, does not need NULL termination, NULL pointer will use default value
	int len_delim;           // length of above buffer
//	char *delim_output;
//	int len_delim_output;
	char *tty;
	char *file;
	int optsId; // filled in automatically
	GreaseLibTargetCallback targetCB; // used if this is target is a callback
	GreaseLibTargetFileOpts *fileOpts; // NULL if not needed
	char *format_pre;
	int format_pre_len;
	char *format_time;
	int format_time_len;
	char *format_level;
	int format_level_len;
	char *format_tag;
	int format_tag_len;
	char *format_origin;
	int format_origin_len;
	char *format_post;
	int format_post_len;
	char *format_pre_msg;
	int format_pre_msg_len;
	uint32_t num_banks; // number of buffer banks the target has. The default is NUM_BANKS defined in logger.h
	uint32_t flags;
} GreaseLibTargetOpts;

typedef struct {
	int optsId;
	TargetId targId;
} GreaseLibStartedTargetInfo;

// tell the target to JSON escape all strings handed to it
#define GREASE_JSON_ESCAPE_STRINGS 0x00000001
// strip incoming messages of ANSI escape sequences (color, etc)
// https://en.wikipedia.org/wiki/ANSI_escape_code#graphics
#define GREASE_STRIP_ANSI_ESCAPE_CODES  0x00000002

GreaseLibTargetFileOpts *GreaseLib_new_GreaseLibTargetFileOpts();
GreaseLibTargetFileOpts *GreaseLib_init_GreaseLibTargetFileOpts(GreaseLibTargetFileOpts *);
void GreaseLib_cleanup_GreaseLibTargetFileOpts(GreaseLibTargetFileOpts *opts);
void GreaseLib_set_flag_GreaseLibTargetFileOpts(GreaseLibTargetFileOpts *opts,uint32_t flag);


GreaseLibTargetOpts *GreaseLib_new_GreaseLibTargetOpts();
GreaseLibTargetOpts *GreaseLib_init_GreaseLibTargetOpts(GreaseLibTargetOpts *);
void GreaseLib_set_flag_GreaseLibTargetOpts(GreaseLibTargetOpts *opts,uint32_t flag);

void GreaseLib_cleanup_GreaseLibTargetOpts(GreaseLibTargetOpts *opts);
void GreaseLib_set_string_GreaseLibTargetFileOpts(GreaseLibTargetFileOpts *opts,uint32_t flag,const char *s);

LIB_METHOD_SYNC(modifyDefaultTarget,GreaseLibTargetOpts *opts);
LIB_METHOD(addTarget,GreaseLibTargetOpts *opts);

//* addFilter(obj) where
//* obj = {
//*      // at least origin and/or tag must be present
//*      origin: 0x33,    // any positive number > 0
//*      tag: 0x25        // any positive number > 0,
//*      target: 3,       // mandatory
//*      mask:  0x4000000 // optional (default, show everything: 0xFFFFFFF),
//*      format: {        // optional (formatting settings)
//*      	pre: 'targ-pre>', // optional pre string
//*      	post: '<targ-post;
//*      }
//* }

#define GREASE_LIB_SET_FILTER_ORIGIN  0x1
#define GREASE_LIB_SET_FILTER_TAG     0x2
#define GREASE_LIB_SET_FILTER_TARGET  0x4
#define GREASE_LIB_SET_FILTER_MASK    0x8

typedef struct {
	uint32_t _enabledFlags;
	uint32_t origin;
	uint32_t tag;
	uint32_t target;
	uint32_t mask;
	FilterId id;
	char *format_pre;
	int format_pre_len;
	char *format_post;
	int format_post_len;
	char *format_post_pre_msg;
	int format_post_pre_msg_len;
} GreaseLibFilter;

GreaseLibFilter *GreaseLib_new_GreaseLibFilter();
GreaseLibFilter *GreaseLib_init_GreaseLibFilter(GreaseLibFilter *);
void GreaseLib_cleanup_GreaseLibFilter(GreaseLibFilter *filter);
void GreaseLib_setvalue_GreaseLibFilter(GreaseLibFilter *opts,uint32_t flag,uint32_t val);

LIB_METHOD_SYNC(addFilter,GreaseLibFilter *filter);
LIB_METHOD_SYNC(disableFilter,GreaseLibFilter *filter);
LIB_METHOD_SYNC(enableFilter,GreaseLibFilter *filter);
//LIB_METHOD_SYNC(modifyDefaultTarget,GreaseLibTargetOpts *opts);

#define GREASE_LIB_SINK_UNIXDGRAM 0x1
#define GREASE_LIB_SINK_PIPE 0x2
#define GREASE_LIB_SINK_SYSLOGDGRAM 0x3
#define GREASE_LIB_SINK_KLOG 0x4  // kernel ring-buffer sink
#define GREASE_LIB_SINK_KLOG2 0x5  // kernel ring-buffer sink - post kernel 3.5

typedef struct {
	char path[GREASE_PATH_MAX];
	uint32_t sink_type;
	SinkId id;
} GreaseLibSink;

GreaseLibSink *GreaseLib_new_GreaseLibSink(uint32_t sink_type, const char *path);
GreaseLibSink *GreaseLib_init_GreaseLibSink(GreaseLibSink *ret, uint32_t sink_type, const char *path);
void GreaseLib_cleanup_GreaseLibSink(GreaseLibSink *sink);

LIB_METHOD_SYNC(addSink,GreaseLibSink *sink);

LIB_METHOD_SYNC(disableTarget, TargetId id);
LIB_METHOD_SYNC(enableTarget, TargetId id);
LIB_METHOD_SYNC(flush, TargetId id);

extern const TagId GREASE_SYSLOGFAC_TO_TAG_MAP[22];
extern const LevelMask GREASE_KLOGLEVEL_TO_LEVEL_MAP[8];
extern const LevelMask GREASE_KLOG_DEFAULT_LEVEL;



#define GREASE_PROCESS_STDOUT 1
#define GREASE_PROCESS_STDERR 2

typedef void (*GreaseLibProcessClosedRedirect) (GreaseLibError *, int stream_type, int pid);

// These methods tell the greaseLib to poll the stated
// file descriptor for reads, and log these under the given
// origin ID. The log will be tagged stdout or stderr
LIB_METHOD_SYNC(addFDForStdout,int fd, uint32_t originId, GreaseLibProcessClosedRedirect cb);
LIB_METHOD_SYNC(addFDForStderr,int fd, uint32_t originId, GreaseLibProcessClosedRedirect cb);

LIB_METHOD_SYNC(addDefaultRedirectorClosedCB, GreaseLibProcessClosedRedirect cb);

LIB_METHOD_SYNC(removeFDForStdout,int fd);
LIB_METHOD_SYNC(removeFDForStderr,int fd);



#ifdef __cplusplus
}
#endif

#endif
