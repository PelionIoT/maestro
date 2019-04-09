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

#ifndef TW_SEMA2_H_
#define TW_SEMA2_H_

#include <unistd.h>
#include <pthread.h>
#include <time.h>
#include <sys/time.h>

#include <TW/tw_utils.h>

//#define _TW_SEMA2_HEAVY_DEBUG

#ifdef _TW_SEMA2_HEAVY_DEBUG
#include <stdio.h>
#endif

namespace TWlib {


#ifdef _TW_SEMA2_HEAVY_DEBUG
#define SEMA2_MUTEX_LOCK(m)  { printf ("*** SEMA2 **** line %d ** Mutex Lock %p\n",__LINE__,this); \
	pthread_mutex_lock(m); }
#define SEMA2_MUTEX_UNLOCK(m) { printf ("*** SEMA2 **** line %d ** Mutex Unlock %p\n",__LINE__,this); \
	pthread_mutex_unlock(m); }
#else
#define SEMA2_MUTEX_LOCK(m) pthread_mutex_lock(m)
#define SEMA2_MUTEX_UNLOCK(m) pthread_mutex_unlock(m)
#endif

/**
 * A two-way semaphore class. Will signal when counter is above zero, or when it decrements.
 * Good for queue type work.
 */
class TW_SemaTwoWay {
protected:
	int cnt;
	int size;

// Yes, two cond variables can share the same mutex:
//	http://stackoverflow.com/questions/4062126/can-2-pthread-condition-variables-share-the-same-mutex
	pthread_mutex_t localMutex; // thread safety for FIFO
	pthread_cond_t gtZeroCond;    // signaled when the count is larger than zero
	pthread_cond_t decrementCond; // signaled when the count goes down
public:
	TW_SemaTwoWay() = delete;
	TW_SemaTwoWay(int init_count) :
	cnt( init_count ), size( init_count )
	{
		pthread_mutex_init( &localMutex, NULL );
		pthread_cond_init( &gtZeroCond, NULL );
		pthread_cond_init( &decrementCond, NULL );
	}


	void reset() {
		SEMA2_MUTEX_LOCK( &localMutex );
		cnt = size;
		SEMA2_MUTEX_UNLOCK( &localMutex );
	}

	void resetNoLock() {
		cnt = size;
	}

	/** should be used only if you understand this class well */
	void cloneFrom(TW_SemaTwoWay &other) {
		cnt = other.cnt;
		size = other.size;
	}

	/**
	 * Acuire the semaphore, waiting indefinitely. Waits until the semaphore is positive, then decrements the semaphore by
	 * one.
	 * @return
	 */
	int acquire() {
		int ret = 0;
		SEMA2_MUTEX_LOCK( &localMutex );
		while(cnt < 1) {
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 wait (acquire) [%p]\n",this);
#endif
			ret = pthread_cond_wait( &gtZeroCond, &localMutex ); // wait for change in cnt
			if(ret) {
#ifdef _TW_SEMA2_HEAVY_DEBUG
				printf ("TW_SEMA2 acquire error (errno %d) (cnt=%d) [%p]\n",ret, cnt, this);
#endif
				break;
			}
		}
		if(!ret) {
			cnt--;
			pthread_cond_signal( &decrementCond );
		}
		SEMA2_MUTEX_UNLOCK( &localMutex );
		return ret;
	}

	int acquire(const struct timespec *abstime) {
		int ret = 0;
		SEMA2_MUTEX_LOCK( &localMutex );
		while(cnt < 1) {
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 wait (acquire) [%p]\n",this);
#endif
			ret = pthread_cond_timedwait( &gtZeroCond, &localMutex, abstime ); // wait for change in cnt
			if(ret) {
#ifdef _TW_SEMA2_HEAVY_DEBUG
				printf ("TW_SEMA2 acquire error (errno %d) (cnt=%d) [%p]\n",ret, cnt, this);
#endif
				break;
			}
		}
		if(!ret) {
			cnt--;
			pthread_cond_signal( &decrementCond );
		}
		SEMA2_MUTEX_UNLOCK( &localMutex );
		return ret;
	}

	int acquireAndKeepLock() {
		int ret = 0;
		SEMA2_MUTEX_LOCK( &localMutex );
		while(1) {
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 acquire wait (cnt=%d) [%p]\n",cnt, this);
#endif
			if(cnt >= 1) break;
			ret = pthread_cond_wait( &gtZeroCond, &localMutex ); // wait for change in cnt
			if(ret) {
#ifdef _TW_SEMA2_HEAVY_DEBUG
				printf ("TW_SEMA2 acquire error (errno %d) (cnt=%d) [%p]\n",ret, cnt, this);
#endif
				break;
			}
		}
		if(ret == 0) {
			cnt--;
			pthread_cond_signal( &decrementCond );
		}
		return ret;
	}

	bool acquireAndKeepLockNoBlock() {
		bool ret = false;
		SEMA2_MUTEX_LOCK( &localMutex );
		if(cnt >= 1) {
			cnt--;
			pthread_cond_signal( &decrementCond );
			ret = true;
		}
		return ret;
	}

	int acquireAndKeepLock(const struct timespec *abstime) {
		int ret = 0;
		SEMA2_MUTEX_LOCK( &localMutex );
		while(1) {
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 acquire wait (cnt=%d) [%p]\n",cnt, this);
#endif
			if(cnt >= 1) break;
			ret = pthread_cond_timedwait( &gtZeroCond, &localMutex, abstime ); // wait for change in cnt
			if(ret) {
#ifdef _TW_SEMA2_HEAVY_DEBUG
				printf ("TW_SEMA acquire error or timeout (errno %d) (cnt=%d) [%p]\n",ret, cnt, this);
#endif
				break;
			}
		}
		if(!ret) {
			cnt--;
			pthread_cond_signal( &decrementCond );
		}
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

	int acquireAndKeepLock(const int64_t usec_wait ) {
		timeval tv;
		timespec ts;
		gettimeofday(&tv, NULL);
		TWlib::add_usec_to_timeval(usec_wait, &tv);
		TWlib::timeval_to_timespec(&tv,&ts);
		return acquireAndKeepLock( &ts );
	}

	void lockSemaOnly() {
		SEMA2_MUTEX_LOCK( &localMutex );
	}


	void releaseSemaLock() {
		SEMA2_MUTEX_UNLOCK( &localMutex );
	}

	int waitForAcquirers(bool lock = true) {
		int ret = 0;
		if(lock)
			SEMA2_MUTEX_LOCK( &localMutex );
//		while(1) {
		    //  if(cnt < size) break;
			if(cnt >= size) {
				ret = pthread_cond_wait( &decrementCond, &localMutex );
				if(cnt >= size) // new
					ret = -1;   // new
			}
//		}
		SEMA2_MUTEX_UNLOCK( &localMutex );
		return ret;
	}

	int waitForAcquirers(const struct timespec *abstime, bool lock = true) {
		int ret = 0;
		if(lock)
			SEMA2_MUTEX_LOCK( &localMutex );
//		while(1) {
//			if(cnt < size) break;
			if(cnt >= size) {
				ret = pthread_cond_timedwait( &decrementCond, &localMutex, abstime );
				if(cnt >= size) // new
					ret = -1;   // new
			}
//		}
		SEMA2_MUTEX_UNLOCK( &localMutex );
		return ret;
	}

	int waitForAcquirers(const int64_t usec_wait, bool lock = true ) {
		timeval tv;
		timespec ts;
		gettimeofday(&tv, NULL);
		TWlib::add_usec_to_timeval(usec_wait, &tv);
		TWlib::timeval_to_timespec(&tv,&ts);
		return waitForAcquirers( &ts, lock );
	}


	int waitForAcquirersKeepLock(bool lock = true) {
		int ret = 0;
		if(lock)
			SEMA2_MUTEX_LOCK( &localMutex );
//		while(1) {
//			if(cnt < size) break;
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 waitForAcquirers decrement (cnt=%d) [%p]\n",cnt, this);
			if(cnt > size) printf("EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEK!!!\n");
#endif
			if(cnt>=size) {
				ret = pthread_cond_wait( &decrementCond, &localMutex );
				if(cnt >= size) // new
					ret = -1;   // new
#ifdef _TW_SEMA2_HEAVY_DEBUG
				else
					printf ("TW_SEMA2 got decrement (cnt=%d)\n",cnt);
#endif
			}
//		}
		return ret;
	}


	int waitForAcquirersKeepLock(const struct timespec *abstime, bool lock = true) {
		int ret = 0;
		if(lock)
			SEMA2_MUTEX_LOCK( &localMutex );
//		while(1) {
//			if(cnt < size) break;
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 waitForAcquirers decrement (cnt=%d) [%p]\n",cnt, this);
			if(cnt > size) printf("EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEK!!!\n");
#endif
			if(cnt>=size) {
				ret = pthread_cond_wait( &decrementCond, &localMutex );
				if(cnt >= size) // new
					ret = -1;   // new
			}
//			ret = pthread_cond_timedwait( &decrementCond, &localMutex, abstime );
#ifdef _TW_SEMA2_HEAVY_DEBUG
			if(ret == 0) printf ("TW_SEMA2 got decrement (cnt=%d)\n",cnt);
			else printf("TW_SEMA2 got error or timeout or other wakeup (cnt=%d)\n",cnt);
#endif
//		}
		return ret;
	}


	int waitForAcquirersKeepLock(const int64_t usec_wait, bool lock = true ) {
		timeval tv;
		timespec ts;
		gettimeofday(&tv, NULL);
		TWlib::add_usec_to_timeval(usec_wait, &tv);
		TWlib::timeval_to_timespec(&tv,&ts);
		return waitForAcquirersKeepLock( &ts, lock );
	}

	int waitForDecrementKeepLock(const struct timespec *abstime, bool lock = true) {
		if(lock) SEMA2_MUTEX_LOCK( &localMutex );
		return pthread_cond_timedwait( &decrementCond, &localMutex, abstime );
	}

	int waitForDecrementKeepLock( const int64_t usec_wait, bool lock = true ) {
		timeval tv;
		timespec ts;
		gettimeofday(&tv, NULL);
		TWlib::add_usec_to_timeval(usec_wait, &tv);
		TWlib::timeval_to_timespec(&tv,&ts);
		if(lock)
			SEMA2_MUTEX_LOCK( &localMutex );
		return pthread_cond_timedwait( &decrementCond, &localMutex, &ts );
	}


//	int waitForDecrementKeepLock(bool lock = true) {
//		int ret = 0;
//		if(lock) {
//			pthread_mutex_lock( &localMutex );
//		}
//		ret = pthread_cond_wait( &decrementCond, &localMutex );
//		return ret;
//	}
//
//	int waitForDecrementKeepLock(const struct timespec *abstime, bool lock = true) {
//		if(lock) pthread_mutex_lock( &localMutex );
//		return pthread_cond_timedwait( &decrementCond, &localMutex, abstime );
//	}
//
//	int waitForDecrementKeepLock( const int64_t usec_wait, bool lock = true ) {
//		timeval tv;
//		timespec ts;
//		gettimeofday(&tv, NULL);
//		TWlib::add_usec_to_timeval(usec_wait, &tv);
//		TWlib::timeval_to_timespec(&tv,&ts);
//		if(lock)
//			pthread_mutex_lock( &localMutex );
//		return pthread_cond_timedwait( &decrementCond, &localMutex, &ts );
//	}

	/**
	 * Acuire or wait until an absolute time. Waits until the semaphore is positive, then decrements the semaphore by
	 * one.
	 * @param abstime
	 * @return
	 */
//	int acquire(const struct timespec *abstime) {
//		int ret = 0;
//		pthread_mutex_lock( &localMutex );
//		if(cnt < 1) {
//#ifdef _TW_SEMA2_HEAVY_DEBUG
//			printf ("TW_SEMA2 timedwait [%p]\n",this);
//#endif
//			ret = pthread_cond_timedwait( &gtZeroCond, &localMutex, abstime ); // wait for change in cnt
//			if(ret == 0) {// if we waited successfully, and no timeout
//				cnt--;   //   the decrement the count down one
//#ifdef _TW_SEMA2_HEAVY_DEBUG
//				printf ("TW_SEMA2 decrementing [%p]\n",this);
//#endif
//			}
//		}
////		if(deleteOnZero && cnt < 1)
////			delete this;
////		else
//		pthread_mutex_unlock( &localMutex );
////		printf ("TW_SEMA2 acquire done\n");
//		return ret;
//	}
//
//	int acquireAndKeepLock(const struct timespec *abstime) {
//		int ret = 0;
//		pthread_mutex_lock( &localMutex );
//		if(cnt < 1) {
//#ifdef _TW_SEMA2_HEAVY_DEBUG
//			printf ("TW_SEMA2 timedwait [%p]\n",this);
//#endif
//			ret = pthread_cond_timedwait( &gtZeroCond, &localMutex, abstime ); // wait for change in cnt
//			if(ret == 0) {// if we waited successfully, and no timeout
//				cnt--;   //   the decrement the count down one
//#ifdef _TW_SEMA2_HEAVY_DEBUG
//				printf ("TW_SEMA2 decrementing [%p]\n",this);
//#endif
//			}
//		}
//		return ret;
//	}





	/**
	 * Increments the counter, and alerts anyone waiting.
	 * @return
	 */
	int release() {
		int ret = 0;
		SEMA2_MUTEX_LOCK( &localMutex );
#ifdef _TW_SEMA2_HEAVY_DEBUG
		printf ("TW_SEMA2 (release) incrementing [%p]\n",this);
#endif
		cnt++;
		if(cnt > 0) { // the 'if' should not be necessary
			ret = pthread_cond_signal( &gtZeroCond );
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 signaled [%p]\n",this);
#endif
		}
		SEMA2_MUTEX_UNLOCK( &localMutex );
		return ret;
	}


	int releaseWithoutLock() {
		int ret = 0;
		cnt++;
#ifdef _TW_SEMA2_HEAVY_DEBUG
		printf ("TW_SEMA2 (release) incrementing (%d) [%p]\n",cnt, this);
#endif
		if(cnt > 0) { // the 'if' should not be necessary
			ret = pthread_cond_signal( &gtZeroCond );
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 signaled [%p]\n",this);
#endif
		}
		return ret;
	}
	/**
	 * Increments the counter, and alerts anyone waiting.
	 * @return
	 */
	int releaseAndKeepLock() {
		int ret = 0;
		SEMA2_MUTEX_LOCK( &localMutex );
#ifdef _TW_SEMA2_HEAVY_DEBUG
		printf ("TW_SEMA2 (release) incrementing [%p]\n",this);
#endif
		cnt++;
		if(cnt > 0) // the 'if' should not be necessary
			ret = pthread_cond_signal( &gtZeroCond );
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 signaled [%p]\n",this);
#endif
		return ret;
	}


	/**
	 * Releases all blocking calls on the TW_Sema. It's *not safe* to use the TW_Sema after this
	 * as the value of the sema is uknown.
	 * @return
	 */
	int releaseAll() {
		int ret = 0;
		SEMA2_MUTEX_LOCK( &localMutex );
#ifdef _TW_SEMA2_HEAVY_DEBUG
		printf ("TW_SEMA2 incrementing [%p]\n",this);
#endif
		cnt++;
//		if(cnt > 0) // the 'if' should not be necessary
		pthread_cond_broadcast( &decrementCond );
		ret = pthread_cond_broadcast( &gtZeroCond );
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 signaled [%p]\n",this);
#endif
		SEMA2_MUTEX_UNLOCK( &localMutex );
		return ret;
	}

	void releaseAllAcquireLocks() {
		SEMA2_MUTEX_LOCK( &localMutex );
		pthread_cond_broadcast( &decrementCond );
		SEMA2_MUTEX_UNLOCK( &localMutex );
	}

	/**
	 * returns the current count (value) on the semaphore
	 * @return
	 */
	int count() {
		int ret = 0;
		SEMA2_MUTEX_LOCK( &localMutex );
		ret = cnt;
		SEMA2_MUTEX_UNLOCK( &localMutex );
		return ret;
	}

	int countNoBlock() {
		return cnt;
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

	~TW_SemaTwoWay() {
#ifdef _TW_SEMA2_HEAVY_DEBUG
			printf ("TW_SEMA2 DESTRUCTOR [%p]\n",this);
#endif
//		if(!deleteOnZero) // if deleteOnZero is set, then this mutex is locked already
		SEMA2_MUTEX_LOCK( &localMutex );  // yes, order is important here...
		pthread_cond_broadcast( &gtZeroCond );
		pthread_cond_broadcast( &decrementCond );
		pthread_cond_destroy( &gtZeroCond );
		pthread_cond_destroy( &decrementCond );
		SEMA2_MUTEX_UNLOCK( &localMutex );
		pthread_mutex_destroy( &localMutex );

	}
};

} // end namespace

#endif /* TW_SEMA2_H_ */
