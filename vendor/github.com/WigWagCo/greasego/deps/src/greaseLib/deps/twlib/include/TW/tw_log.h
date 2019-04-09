// WigWag LLC
// (c) 2010
// tw_log.h
// Author: ed
// Apr 17, 2011

/*
 * tw_log.h
 *
 *  Created on: Apr 17, 2011
 *      Author: ed
 */

#ifndef TW_LOG_H_
#define TW_LOG_H_

#include <stdio.h>
#include <cstdarg>

#include <syslog.h>

#include <TW/tw_defs.h>
//#include <tw_log_msg.h>

#include <TW/tw_autopointer.h>
#include <TW/tw_sema.h>
#include <TW/tw_syscalls.h>

#define TW_MAX_LOG_MESSAGE 1000

#define TW_LOG_ALL 0xFFFF

#define TW_LL_DEBUG1    0x0040
#define TW_LL_NOTIFY    0x0020
#define TW_LL_CONFIG    0x0010
#define TW_LL_WARN      0x0008
#define TW_LL_ERROR     0x0004
#define TW_LL_CRITICAL  0x0002
#define TW_LL_CRASH     0x0001

// get the thread numbers or what not... on linux this is the LWP (light weight process) number
// _TW_getLWPnum() is defined in syscalls-ARCH.c where ARCH is armel, x86, or whatever you use
#define _TW_LWP_            _TW_getLWPnum()


#define TW_PREFIX_DEBUG "DEBUG "
#define TW_PREFIX_NOTIFY "NOTIFY "
#define TW_PREFIX_CONFIG "CONFIG "
#define TW_PREFIX_WARN "WARN "
#define TW_PREFIX_ERROR "ERROR "
#define TW_PREFIX_CRITICAL "CRITICAL "
#define TW_PREFIX_CRASH "CRASH "
#define TW_PRE_LINE "(%s:%d) "
#define TW_PRE_LINE_THREAD "(%s:%d,lwp:%d) "

// TW_OVERRIDE_LOG allows you to disable the logging or replace it.
#ifndef TW_OVERRIDE_LOG
// if TWLOG_PRINT_MODULENAME is defined then every log entry will have the file/line number of where the log call was made
#ifdef TWLOG_PRINT_MODULENAME
#define TW_NOTIFY( s, ... )  TWlib::TW_log::log( TW_LL_NOTIFY, TW_PREFIX_NOTIFY TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_CONFIG( s, ... )  TWlib::TW_log::log( TW_LL_CONFIG, TW_PREFIX_CONFIG TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_WARN( s, ... )    TWlib::TW_log::log( TW_LL_WARN,   TW_PREFIX_WARN TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_ERROR( s, ... )   TWlib::TW_log::log( TW_LL_ERROR,  TW_PREFIX_ERROR TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_CRTIICAL( s, ... )  TWlib::TW_log::log( TW_LL_CRITICAL, TW_PREFIX_CRITICAL TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_CRASH( s, ... )   TWlib::TW_log::log( TW_LL_CRASH, TW_PREFIX_CRASH TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_CRASH_LT( s, ... )   TWlib::TW_log::log( TW_LL_CRASH, TW_PREFIX_CRASH TW_PRE_LINE_THREAD s, __FILE__, __LINE__, _TW_LWP_, __VA_ARGS__ )
#define TW_DEBUG( s, ... )  TWlib::TW_log::log( TW_LL_DEBUG1, TW_PREFIX_DEBUG TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_DEBUG_L( s, ... )  TWlib::TW_log::log( TW_LL_DEBUG1, TW_PREFIX_DEBUG TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_DEBUG_LT( s, ... )  TWlib::TW_log::log( TW_LL_DEBUG1, TW_PREFIX_DEBUG TW_PRE_LINE_THREAD s, __FILE__, __LINE__, _TW_LWP_, __VA_ARGS__ )

#else
#define TW_NOTIFY( s, ... )  TWlib::TW_log::log( TW_LL_NOTIFY, TW_PREFIX_NOTIFY s, __VA_ARGS__ )
#define TW_CONFIG( s, ... )  TWlib::TW_log::log( TW_LL_CONFIG, TW_PREFIX_CONFIG s, __VA_ARGS__ )
#define TW_WARN( s, ... )    TWlib::TW_log::log( TW_LL_WARN,   TW_PREFIX_WARN s, __VA_ARGS__ )
#define TW_ERROR( s, ... )   TWlib::TW_log::log( TW_LL_ERROR,  TW_PREFIX_ERROR s, __VA_ARGS__ )
#define TW_CRTIICAL( s, ... )  TWlib::TW_log::log( TW_LL_CRITICAL, TW_PREFIX_CRITICAL s, __VA_ARGS__ )
#define TW_CRASH( s, ... )   TWlib::TW_log::log( TW_LL_CRASH, TW_PREFIX_CRASH s, __VA_ARGS__ )
#define TW_CRASH_LT( s, ... )   TWlib::TW_log::log( TW_LL_CRASH, TW_PREFIX_CRASH TW_PRE_LINE_THREAD s, __FILE__, __LINE__, _TW_LWP_, __VA_ARGS__ )
#define TW_DEBUG( s, ... )  TWlib::TW_log::log( TW_LL_DEBUG1, TW_PREFIX_DEBUG s, __VA_ARGS__ )
#define TW_DEBUG_L( s, ... )  TWlib::TW_log::log( TW_LL_DEBUG1, TW_PREFIX_DEBUG TW_PRE_LINE s, __FILE__, __LINE__, __VA_ARGS__ )
#define TW_DEBUG_LT( s, ... )  TWlib::TW_log::log( TW_LL_DEBUG1, TW_PREFIX_DEBUG TW_PRE_LINE_THREAD s, __FILE__, __LINE__, _TW_LWP_, __VA_ARGS__ )
#endif

#else

#ifndef TW_NOTIFY
#define TW_NOTIFY {}
#endif
#ifndef TW_CONFIG
#define TW_CONFIG {}
#endif
#ifndef TW_WARN
#define TW_WARN {}
#pragma message "twlib NOTE: TW_WARN is disabled."
#endif
#ifndef TW_ERROR
#define TW_ERROR {}
#pragma message "twlib NOTE: TW_ERROR is disabled. Important messages failures may not print out."
#endif
#ifndef TW_CRASH
#define TW_CRASH {}
#pragma message "twlib NOTE: TW_CRASH is disabled."
#endif
#ifndef TW_CRASH_LT
#define TW_CRASH_LT {}
#pragma message "twlib NOTE: TW_CRASH_LT is disabled."
#endif
#ifndef TW_DEBUG
#define TW_DEBUG {}
#endif
#ifndef TW_DEBUG_LT
#define TW_DEBUG_LT {}
#endif
#ifndef TW_DEBUG_L
#define TW_DEBUG_L {}
#endif

#endif

#ifndef TW_OVERRIDE_LOG
namespace TWlib {


class TW_log;
class TW_logFacility;

typedef TWlib::autoPointer<TW_log> TWlog_SP;


class TW_logFacility {
public:
	virtual bool open() { return true; }
	virtual void logTo( const char *s ) = 0;
	virtual void shutdown() {}
	virtual ~TW_logFacility() {};
};

class TW_printfFacility : public TW_logFacility {
	static TW_Mutex _logMutex; // there is only one stdout, so this is necessary for multiple loggers
	// If -Wformat is specified, also warn if the format string is not a string literal and so cannot be checked, unless the format function takes its format arguments as a va_list.
//#pragma GCC diagnostic ignored  "-Wformat-security"
	static void writeout(const char *s) {
		_logMutex.acquire();
		fputs(s, stdout); // using puts now (in case the string itself has %s, etc...)
		fflush(stdout);
		_logMutex.release();
	}
//#pragma GCC diagnostic pop
public:
	virtual ~TW_printfFacility() {}
	void logTo( const char * s ) {
		writeout(s);
	}
};

class TW_syslogFacility : public TW_logFacility {
public:
	void logTo( const char * s );
	virtual ~TW_syslogFacility() {}
};

/** a thread safe logging class with support for differen 'facilities'
 * @see TW_logFacility
 * */
class TW_log {
protected:
	static TW_Mutex _instanceMutex; // keeps multiple instances from getting created
	static TWlog_SP _instance;
	static bool _logSetup;

	TW_logFacility *_facility;

	unsigned int _logLevel;

	int _maxSize; // TW_MAX_LOG_MESSAGE - strlen( JNI_LOGPREFIX )
	char *_bufferFillMark;
	char _buffer[TW_MAX_LOG_MESSAGE];
	TW_Mutex _bufferMutex;

public:
	TW_log() : _facility(NULL), _logLevel( TW_LOG_ALL ), _maxSize(TW_MAX_LOG_MESSAGE),
	_bufferFillMark(_buffer), _bufferMutex() {
		// ------------------------------------------------------------     DEFAULT FACILITY
		_facility = new TW_printfFacility();
		_facility->open();
	}

	~TW_log();

	void useFacility( TW_logFacility *fac );

	static TWlog_SP& getInstance() { // returns the default logger
		_instanceMutex.acquire();
		if (!TW_log::_instance.attached()) {
			TW_log::_instance.reset(new TW_log());
		}
		_instanceMutex.release();
		return _instance;
	}
	void setupPrefix( const char *p );
//	void *setPrepend( const char *s );
	void setLogLevel( unsigned int level ) { _logLevel = level; }
	unsigned int logLevel() { return _logLevel; }

	void logTo( unsigned int l, const char *s, va_list *vaptr );

	/** logs to the default logger **/
	static void log( unsigned int l, const char *s, ... ) {
		TW_log *L = TW_log::getInstance();
		va_list argptr;
		va_start(argptr, s);
		L->logTo(l,s, &argptr );
		va_end(argptr);
	}

	// convenience functions

//	static void logCritical( const char * format, ... );
//	static void logError( const char *format, ... );
//	static void logWarn( const char *format, ... );
//	static void logNotify( const char *format, ... );
//	static void logConfig( const char *format, ... );
//	static void logDebug( const char *format, ... );

};


}
#endif // TW_OVERRIDE_LOG

#endif /* TW_LOG_H_ */
