// WigWag LLC
// (c) 2011
// tw_sema.h
// Author: ed
// Mar 22, 2011

/*
 * tw_sema.h
 *
 *  Created on: Mar 22, 2011
 *      Author: ed
 */

#ifndef TW_SEMA_H_
#define TW_SEMA_H_

#include <unistd.h>
#include <pthread.h>
#include <time.h>
#include <sys/time.h>

#include <TW/tw_utils.h>

// #define _TW_SEMA_HEAVY_DEBUG

#ifdef _TW_SEMA_HEAVY_DEBUG
#include <stdio.h>
#endif

namespace TWlib {

const int TW_MUTEX_SUCCESS = 0;
const int TW_MUTEX_BUSY = EBUSY;
const int TW_MUTEX_TIMEOUT = ETIMEDOUT;
const int TW_MUTEX_DEADLOCK = EDEADLK;
const int TW_MUTEX_INVALID = EINVAL;
const int TW_MUTEX_EXCEEDED = EAGAIN; // EAGAIN ... This is a POSIX extension
                                      // [XSI]  The mutex could not be acquired because the maximum number of recursive locks for mutex has been exceeded.

// OS X does not have pthread_mutex_timedlock
// this is from here: http://aleksmaus.blogspot.com/2011/12/missing-pthreadmutextimedlock-on.html
#ifdef __APPLE__
int
pthread_mutex_timedlock (pthread_mutex_t *mutex, struct timespec *timeout)
{
 struct timeval timenow;
 struct timespec sleepytime;
 int retcode;
 
 /* This is just to avoid a completely busy wait */
 sleepytime.tv_sec = 0;
 sleepytime.tv_nsec = 10000000; /* 10ms */
 
 while ((retcode = pthread_mutex_trylock (mutex)) == EBUSY) {
  gettimeofday (&timenow, NULL);
  
  if (timenow.tv_sec >= timeout->tv_sec &&
      (timenow.tv_usec * 1000) >= timeout->tv_nsec) {
   return ETIMEDOUT;
  }
  
  nanosleep (&sleepytime, NULL);
 }
 
 return retcode;
}
#endif


class TW_Mutex {
protected:
	pthread_mutex_t localMutex; // thread safety for FIFO
public:
	TW_Mutex() {
		pthread_mutex_init( &localMutex, NULL );
	}
	int acquire() {
		return pthread_mutex_lock( &localMutex );
	}
	int acquire(TimeVal &t) {
		return pthread_mutex_timedlock( &localMutex, t.timespec() );
	}
	int release() {
		return pthread_mutex_unlock( &localMutex );
	}
	// returns TW_MUTEX_SUCCESS if succesful lock, if non-zero a lock was not acquired. This will not block.
	int tryAcquire() {
		return pthread_mutex_trylock( &localMutex );
	}
	~TW_Mutex() {
		pthread_mutex_destroy( &localMutex );
	}
};

// A null class for having no MUTEX protection
class TW_NoMutex {
public:
	TW_NoMutex() { }
	int acquire() { return 0; }
	int acquire(TimeVal &t) { return 0; }
	int release() { return 0; }
	int tryAcquire() { return 0; }
	~TW_NoMutex() { }
};

// A recursive capable TW_Mutex
class TW_RecursiveMutex {
protected:
	pthread_mutex_t localMutex;
	pthread_mutexattr_t attr;
public:
	TW_RecursiveMutex() {
		pthread_mutexattr_init(&attr);
		pthread_mutexattr_settype(&attr, PTHREAD_MUTEX_RECURSIVE);
		pthread_mutex_init( &localMutex, &attr );
	}
	int acquire() {
		return pthread_mutex_lock( &localMutex );
	}
	int acquire(TimeVal &t) {
		return pthread_mutex_timedlock( &localMutex, t.timespec() );
	}
	int release() {
		return pthread_mutex_unlock( &localMutex );
	}
	int tryAcquire() {
		return pthread_mutex_trylock( &localMutex );
	}
	~TW_RecursiveMutex() {
		pthread_mutex_destroy( &localMutex );
	}
};
/**
 * A simple sempahore class. I did this b/c I could not get ACE's sempahore stuff to work as expected.
 * When the Semaphore is above zero,
 */
class TW_Sema {
protected:
	int cnt;
//	bool deleteOnZero;
	pthread_mutex_t localMutex; // thread safety for FIFO
	pthread_cond_t gtZeroCond;
public:
	TW_Sema(int init_count) :
	cnt( init_count )
//	,deleteOnZero( false )
	{
		pthread_mutex_init( &localMutex, NULL );
		pthread_cond_init( &gtZeroCond, NULL );
	}
	/**
	 * Acuire the semaphore, waiting indefinitely. Waits until the semaphore is positive, then decrements the semaphore by
	 * one.
	 * @return
	 */
	int acquire() {
		int ret = 0;
		pthread_mutex_lock( &localMutex );
		while(cnt < 1) {
#ifdef _TW_SEMA_HEAVY_DEBUG
			printf ("TW_SEMA wait [%x]\n",this);
#endif
			ret = pthread_cond_wait( &gtZeroCond, &localMutex ); // wait for change in cnt
		}
		pthread_mutex_unlock( &localMutex );
		cnt--;
		return ret;
	}
	/**
	 * Acuire or wait until an absolute time. Waits until the semaphore is positive, then decrements the semaphore by
	 * one.
	 * @param abstime
	 * @return
	 */
	int acquire(const struct timespec *abstime) {
		int ret = 0;
		pthread_mutex_lock( &localMutex );
		if(cnt < 1) {
#ifdef _TW_SEMA_HEAVY_DEBUG
			printf ("TW_SEMA timedwait [%x]\n",this);
#endif
			ret = pthread_cond_timedwait( &gtZeroCond, &localMutex, abstime ); // wait for change in cnt
			if(ret == 0) {// if we waited successfully, and no timeout
				cnt--;   //   the decrement the count down one
#ifdef _TW_SEMA_HEAVY_DEBUG
				printf ("TW_SEMA decrementing [%x]\n",this);
#endif
			}
		}
//		if(deleteOnZero && cnt < 1)
//			delete this;
//		else
		pthread_mutex_unlock( &localMutex );
//		printf ("TW_SEMA acquire done\n");
		return ret;
	}
	/**
	 * An easier to use acquire waiting for a given number of microseconds.
	 * @param usec_wait
	 * @return
	 */
	int acquire(const int64_t usec_wait ) {
		timeval tv;
		timespec ts;
		gettimeofday(&tv, NULL);
		TWlib::add_usec_to_timeval(usec_wait, &tv);
		TWlib::timeval_to_timespec(&tv,&ts);
		return acquire( &ts );
	}
	/**
	 * Increments the counter, and alerts anyone waiting.
	 * @return
	 */
	int release() {
		int ret = 0;
		pthread_mutex_lock( &localMutex );
#ifdef _TW_SEMA_HEAVY_DEBUG
		printf ("TW_SEMA incrementing [%x]\n",this);
#endif
		cnt++;
		if(cnt > 0) // the 'if' should not be necessary
			ret = pthread_cond_signal( &gtZeroCond );
#ifdef _TW_SEMA_HEAVY_DEBUG
			printf ("TW_SEMA signaled [%x]\n",this);
#endif
		pthread_mutex_unlock( &localMutex );
		return ret;
	}

	/**
	 * Releases all blocking calls on the TW_Sema. It's *not safe* to use the TW_Sema after this
	 * as the value of the sema is uknown.
	 * @return
	 */
	int releaseAll() {
		int ret = 0;
		pthread_mutex_lock( &localMutex );
#ifdef _TW_SEMA_HEAVY_DEBUG
		printf ("TW_SEMA incrementing [%x]\n",this);
#endif
		cnt++;
		if(cnt > 0) // the 'if' should not be necessary
			ret = pthread_cond_broadcast( &gtZeroCond );
#ifdef _TW_SEMA_HEAVY_DEBUG
			printf ("TW_SEMA signaled [%x]\n",this);
#endif
		pthread_mutex_unlock( &localMutex );
		return ret;
	}

	/**
	 * returns the current count (value) on the semaphore
	 * @return
	 */
	int count() {
		int ret = 0;
		pthread_mutex_lock( &localMutex );
		ret = cnt;
		pthread_mutex_unlock( &localMutex );
		return ret;
	}

	/**
	 * flags the TW_Sema to destroy itself when the semaphore's internal count reaches zero.
	 * To be used like:
	 * sema->release();
	 * sema->flagDeleteAtZero(); ---> deletes immediately if at zero, or waits until zero, then deletes self
	 * ..some other thread:
	 * sema-acquire();
	 *
	 * .. use with care
	 */
/*	BAD IDEA - prevents stuff from being declared on stack easily
 * void flagDeleteAtZero() {
		pthread_mutex_lock( &localMutex );  // yes, order is important here...
		deleteOnZero = true;
		if(cnt < 1)
			delete this;
		else
			pthread_mutex_unlock( &localMutex );
	}
*/

	~TW_Sema() {
#ifdef _TW_SEMA_HEAVY_DEBUG
			printf ("TW_SEMA DESTRUCTOR [%x]\n",this);
#endif
//		if(!deleteOnZero) // if deleteOnZero is set, then this mutex is locked already
		pthread_mutex_lock( &localMutex );  // yes, order is important here...
		pthread_cond_broadcast( &gtZeroCond );
		pthread_cond_destroy( &gtZeroCond );
		pthread_mutex_unlock( &localMutex );
		pthread_mutex_destroy( &localMutex );

	}
};

} // end namespace

#endif /* TW_SEMA_H_ */
