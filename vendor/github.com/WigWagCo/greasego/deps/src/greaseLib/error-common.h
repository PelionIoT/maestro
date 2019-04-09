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
 * error-common.h
 *
 *  Created on: Oct 31, 2014
 *      Author: ed
 * (c) 2015, Framez Inc
 */
#ifndef ERROR_COMMON_H_
#define ERROR_COMMON_H_

#include <string.h>
#include <stdlib.h>
#include <stdio.h>

#ifndef GREASE_LIB
#include <v8.h>
#include "nan.h"
#else
#include "grease_lib.h"
#endif

#include <errno.h>
#if !defined(_MSC_VER)
#include <unistd.h>
#endif
#include <fcntl.h>
#include <signal.h>
#include <sys/types.h>
#include <sys/stat.h>


// https://gcc.gnu.org/onlinedocs/cpp/Stringification.html
#define xstr(s) str(s)
#define str(s) #s

// concept from node.js src/node_constants.cc
//#define _ERRCMN_DEFINE_CONSTANT(target, constant)
//  (target)->Set(v8::String::NewSymbol(#constant),
//                v8::Number::New(constant),
//                static_cast<v8::PropertyAttribute>(
//                    v8::ReadOnly|v8::DontDelete))

#define _ERRCMN_DEFINE_CONSTANT(target, constant)                         \
  Nan::ForceSet(target,Nan::New(#constant).ToLocalChecked(),               \
                Nan::New((uint32_t)constant),                                \
                static_cast<v8::PropertyAttribute>(                       \
                    v8::ReadOnly|v8::DontDelete))

//// our mode - this is the same thing, with a reverse lookup key also
//#define _ERRCMN_DEFINE_CONSTANT_WREV(target, constant)
//  (target)->Set(v8::String::NewSymbol(#constant),
//                v8::Number::New(constant),
//                static_cast<v8::PropertyAttribute>(
//                    v8::ReadOnly|v8::DontDelete));
//  (target)->Set(v8::String::New( xstr(constant) ),
//                v8::String::New(#constant),
//                static_cast<v8::PropertyAttribute>(
//                    v8::ReadOnly|v8::DontDelete));

#define _ERRCMN_DEFINE_CONSTANT_WREV(target, constant)                    \
  Nan::ForceSet(target,Nan::New(#constant).ToLocalChecked(),                 \
                Nan::New((uint32_t)constant),                                \
                static_cast<v8::PropertyAttribute>(                       \
                    v8::ReadOnly|v8::DontDelete));                        \
  Nan::ForceSet(target, Nan::New( xstr(constant) ).ToLocalChecked(),        \
                Nan::New(#constant).ToLocalChecked(),                         \
                static_cast<v8::PropertyAttribute>(                       \
                    v8::ReadOnly|v8::DontDelete));                        \

// custom error codes should be above this value
#define _ERRCMN_CUSTOM_ERROR_CUTOFF  4000

#define GREASE_UNKNOWN_FAILURE 4001
#define GREASE_UNKNOWN_TTY 4002
#define GREASE_UNKNOWN_NO_PATH 4003

namespace _errcmn {

#ifndef GREASE_LIB
	void DefineConstants(v8::Handle<v8::Object> target);
	v8::Local<v8::Value> errno_to_JS(int _errno, const char *prefix = NULL);
	v8::Local<v8::Value> errno_to_JSError(int _errno, const char *prefix = NULL);
	v8::Handle<v8::Value> err_ev_to_JS(err_ev &e, const char *prefix);
#endif

	char *get_error_str(int _errno);
	void free_error_str(char *b);
	struct err_ev {
		char *errstr;
		int _errno;
		err_ev(void) : errstr(NULL), _errno(0) {};
		void setError(int e,const char *m=NULL);
		err_ev(int e) : err_ev() {
			setError(e);
		}
		err_ev(const err_ev &o) = delete;
		inline err_ev &operator=(const err_ev &o) = delete;
		inline err_ev &operator=(err_ev &&o) {
			this->errstr = o.errstr;  // transfer string to other guy...
			this->_errno = o._errno;
			o.errstr = NULL; o._errno = 0;
			return *this;
		}
		inline void clear() {
			if(errstr) ::free(errstr); errstr = NULL;
			_errno = 0;
		}
		~err_ev() {
			if(errstr) ::free(errstr);
		}
		bool hasErr() { return (_errno != 0); }
#ifdef GREASE_LIB
		GreaseLibError *toGreaseLibError(GreaseLibError *in) {
			if(!in) {
				in = (GreaseLibError *) malloc(sizeof(GreaseLibError));
			}
			if(in) {
				in->_errno = this->_errno;
				::strncpy(in->errstr,this->errstr,GREASE_LIB_MAX_ERR_STRING);
			}
			return in;
		}
#endif
	};
}

#ifndef NO_ERROR_CMN_OUTPUT  // if define this, you must define these below yourself


// ensure we get the XSI compliant strerror_r():
// see: http://man7.org/linux/man-pages/man3/strerror.3.html

#ifdef __cplusplus
extern "C" {
#endif
extern int __xpg_strerror_r (int __errnum, char *__buf, size_t __buflen);
#ifdef __cplusplus
};
#endif


#define ERR_STRERROR_R(ernum,b,len) __xpg_strerror_r(ernum, b, len)

#ifdef ERRCMN_DEBUG_BUILD
#pragma message "Build is Debug"
// confused? here: https://gcc.gnu.org/onlinedocs/cpp/Variadic-Macros.html
#define ERROR_OUT(s,...) fprintf(stderr, "**ERROR (greaseLib)** %s:%d  " s "\n", __BASE_FILE__,__LINE__,  ##__VA_ARGS__ )
//#define ERROR_PERROR(s,...) fprintf(stderr, "*****ERROR***** " s, ##__VA_ARGS__ );
#define ERROR_PERROR(s,E,...) { char *__S=_errcmn::get_error_str(E); fprintf(stderr, "**ERROR** [ %s ] " s "\n", __S, ##__VA_ARGS__ ); _errcmn::free_error_str(__S); }

#define DBG_OUT(s,...) fprintf(stderr, "**DEBUG_greaseLib** " s "\n", ##__VA_ARGS__ )
#define DBG_OUT_LINE(s,...) fprintf(stderr, "**DEBUG_greaseLib** %s:%d " s "\n", __BASE_FILE__,__LINE__,##__VA_ARGS__ )
#define IF_DBG( x ) { x }
#else
#define ERROR_OUT(s,...) fprintf(stderr, "**ERROR (greaseLib)** %s:%d " s "\n", __BASE_FILE__,__LINE__, ##__VA_ARGS__ )//#define ERROR_PERROR(s,...) fprintf(stderr, "*****ERROR***** " s, ##__VA_ARGS__ );
#define ERROR_PERROR(s,E,...) { char *__S=_errcmn::get_error_str(E); fprintf(stderr, "**ERROR** (greaseLib) [ %s ] " s, __S, ##__VA_ARGS__ ); _errcmn::free_error_str(__S); }
#define DBG_OUT(s,...) {}
#define DBG_OUT_LINE(s,...) {}
#define IF_DBG( x ) {}
#endif

#else

#define ERROR_OUT(s,...) {} //#define ERROR_PERROR(s,...) fprintf(stderr, "*****ERROR***** " s, ##__VA_ARGS__ );
#define ERROR_PERROR(s,E,...) {}
#define DBG_OUT(s,...) {}
#define IF_DBG( x ) {}


#endif // NO_ERROR_CMN_OUTPUT

#define CRITICAL_FAILURE(s,...) fprintf(stderr,"**CRITICAL (greaseLib)** %s:%d " s, __FILE__, __LINE__)


#endif /* ERROR_COMMON_H_ */
