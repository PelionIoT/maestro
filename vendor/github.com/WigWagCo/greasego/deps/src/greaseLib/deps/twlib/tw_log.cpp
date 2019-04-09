// WigWag LLC
// (c) 2010
// tw_log.cpp
// Author: ed

#include <cstdio>
#include <cstdarg>

#include <stdio.h>
#include <string.h>

#include <TW/tw_log.h>
#include <TW/tw_sema.h>

using namespace TWlib;

TWlog_SP TW_log::_instance;
TW_Mutex TW_log::_instanceMutex;
bool TW_log::_logSetup = false;

TW_Mutex TW_printfFacility::_logMutex;

void TW_log::logTo(unsigned int l, const char *s, va_list *vaptr ) {
	if (_logLevel & l) {
		char *out;
		//		printf("acquire mutex" );
		_bufferMutex.acquire();
//		va_list argptr;
//		va_start ( argptr, s);

//		vprintf( s,argptr);
		vsnprintf(_bufferFillMark, _maxSize, s, *vaptr);
//		va_end( argptr );

		if(_facility) {
			_facility->logTo(_buffer);
		} else
			printf("TW_log: NO FACILITY -> %s", _buffer);

		_bufferMutex.release();
//		if (_instance->_attachedToJava) {
//			jstring msg;
//			msg = env->NewStringUTF(_instance->_buffer);
//			//			printf("calling logFunc now: %s\n", _buffer );
//			_instance->_bufferMutex.release();
//			//			printf("released mutex" );//
//
//			env->CallVoidMethod(_instance->_jLogFacility, _instance->_logFunc, (jint) l, msg);
//			env->DeleteLocalRef(msg);
//		} else {
//			syslog(LOG_MAKEPRI(LOG_USER, LOG_WARNING), "%s", _instance->_buffer);
//			_bufferMutex.release();
//		}
	}
}

TW_log::~TW_log() {
	if(_facility) {
		_facility->shutdown();
		delete _facility;
	}
}

void TW_log::useFacility( TW_logFacility *fac ) {
	if(_facility) {
		_facility->shutdown();
		delete _facility;
	}
	_facility = fac;
	if(!_facility->open()) {
		printf("TW_log facility failed to open. !!!!!!! \n");
		delete _facility;
		_facility = NULL;
	}
}

void TW_log::setupPrefix( const char *p ) {
	_bufferMutex.acquire();
	sprintf( _buffer, "%s", p );
	_bufferFillMark  = ((char *) _buffer) + strlen(_buffer);
	_maxSize = TW_MAX_LOG_MESSAGE - strlen( p );
	_bufferMutex.release();
}

/*
// pulled from ACE
void
__ace_assert(const char *file, int line, const TW_TCHAR *expression)
{
  int error = TW_Log_Msg::last_error_adapter ();
  TW_Log_Msg *log = ACE_Log_Msg::instance ();

  log->set (file, line, -1, error, log->restart (),
            log->msg_ostream (), log->msg_callback ());

  log->log (LM_ERROR, TW_TEXT ("ACE_ASSERT: file %N, line %l assertion failed for '%s'.%a\n"), expression, -1);
}

*/


