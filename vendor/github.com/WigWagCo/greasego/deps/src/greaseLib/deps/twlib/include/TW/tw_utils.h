// WigWag LLC
// (c) 2011
// tw_time.h
// Author: ed
// Mar 22, 2011

/*
 * tw_time.h
 *
 *  Created on: Mar 22, 2011
 *      Author: ed
 */

#ifndef TW_TIME_H_
#define TW_TIME_H_


#ifdef _USING_GLIBC_
#include <execinfo.h>
#endif

#include <cstdarg>
#include <errno.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <time.h>
#include <sys/time.h>
#include <stdint.h>

#include <string>

#include <TW/tw_syscalls.h>

using namespace std;

#define MAX_STRING_CNV_BUF 100

namespace TWlib {

struct timeval* usec_to_timeval( int64_t usec, struct timeval* tv );
struct timeval* add_usec_to_timeval( int64_t usec, struct timeval* tv );
struct timespec *timeval_to_timespec(struct timeval *tv, struct timespec *ts);
char *convInt( char *out, int val, size_t max );
char *convIntHex( char *out, unsigned int val, size_t max );
string &hexDumpToString(char *head, int size, string &out);
string &string_printf(string &fillme, const char *fmt, ... );

uint32_t data_hash_Hsieh (const char * data, int len);

/// @class TimeVal
/// A class wrapping a number of time functions, and handling both timespec and timeval values.
class TimeVal {
protected:
	struct timeval _timeval;
	struct timespec _timespec;

public:
	TimeVal() { }

	TimeVal( TimeVal &o ) {
		::memcpy(&_timeval, &o._timeval, sizeof(struct timeval));
		::memcpy(&_timespec, &o._timespec, sizeof(struct timespec));
	}

	// sets a time for an interval
	TimeVal &setInterval( int64_t usec ) {
		::memset(&_timeval,0,sizeof(struct timeval));
		usec_to_timeval( usec, &_timeval );
		return *this;
	}

	// add 'usec' useconds to the TimeVal's existing value.
	TimeVal &addUsec( int64_t usec ) {
		add_usec_to_timeval(usec, &_timeval);
		return *this;
	}

	TimeVal &gettimeofday() {
		::gettimeofday(&_timeval,NULL);
		return *this;
	}

	TimeVal &gettimeofday(struct timezone *z) {
		::gettimeofday(&_timeval,z);
		return *this;
	}

	struct timespec *timespec() {
		timeval_to_timespec( &_timeval, &_timespec );
		return const_cast<struct timespec *>( &_timespec);
	}

	struct timeval *timeval() {
		return const_cast<struct timeval *>( &_timeval);
	}
};


#define _TW_MAX_STACKTRACE 25

/// @class StackDump
/// Debugging tool for grabbing and storing a stack dump.
class StackDump {
protected:
#ifdef _USING_GLIBC_
	StackDump() : size( 0 ) { }
	void *array[_TW_MAX_STACKTRACE];
	size_t size;
#else
	StackDump() { }
#endif

public:

	string &stringify(string &fill) {
	      fill.clear();
	#ifdef _USING_GLIBC_
			// this stuff is only supported in Glibc environments
			char **strings;
			size_t i;

			strings = backtrace_symbols (array, size);

			for (i = 0; i < size; i++) {
			fill.append((char *) strings[i]);
			fill.append(1,'\n');
			}

			free (strings);
	#else
			fill.append("StackDump - trace not supported on platform.\n");
	#endif
			return fill;
	}

	static std::string& stackDump(std::string& fill) {
	      fill.clear();
	#ifdef _USING_GLIBC_
			// this stuff is only supported in Glibc environments
			void *array[25];
			size_t size;
			char **strings;
			size_t i;

			size = backtrace (array, 25);
			strings = backtrace_symbols (array, size);

			for (i = 0; i < size; i++) {
			fill.append((char *) strings[i]);
			fill.append(1,'\n');
			}

			free (strings);
	#else
			fill.append("StackDump::stackDump(string) - stack dump not supported on platform.\n");
	#endif
			return fill;
		}

	~StackDump() {}

	static StackDump *getStackTrace() {
#ifdef _USING_GLIBC_
		StackDump *ret = new StackDump();
		ret->size = backtrace (ret->array, _TW_MAX_STACKTRACE);
		return ret;
#else
		return NULL;
#endif
	}

};


long getLWP();



}
#endif /* TW_TIME_H_ */
