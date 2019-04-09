
// tupperware container lib
#ifndef _TW_CIRCULAR
#define _TW_CIRCULAR

// FIFO: a simple class to handle a fifo list of void pointers.
#include <pthread.h>
#include <errno.h>

//#include <TW/tw_log.h>

// #ifdef 'g++' whatever that is...
#include <new>
// #endif

#include <TW/tw_utils.h> // TimeVal
#include <TW/tw_sema2.h>
#include <TW/tw_alloc.h>
//#include "logging.h"

#if __cplusplus >= 201103L
#define TWLIB_HAS_MOVE_SEMANTICS 1
#endif
#ifdef TWLIB_HAS_MOVE_SEMANTICS
#include <utility>
#endif



//#define DEBUG_TW_CIRCULAR_H


#ifdef DEBUG_TW_CIRCULAR_H
#include <sys/syscall.h>

static long gettid__twcircular() {
	return syscall(SYS_gettid);
}

#pragma message "!!!!!!!!!!!! tw_circular is Debug Heavy!!"
// confused? here: https://gcc.gnu.org/onlinedocs/cpp/Variadic-Macros.html
#define TW_CIRCULAR_DBG_OUT(s,...) fprintf(stderr, "**DEBUG (tid:%d)** " s "\n", gettid__twcircular(), ##__VA_ARGS__ )
#define IF_CIRCULAR_DBG_OUT( x ) { x }
#else
#define TW_CIRCULAR_DBG_OUT(s,...) {}
#define IF_CIRCULAR_DBG_OUT( x ) {}
#endif


namespace TWlib {





/**
 * A circular FIFO. This is a thread-safe, non and/or blocking queue using pthread conditions.
 * T must support:
 * self assignment: operator= (T &a, T &b) or move()
 * default constructor
 */
template <class T, class ALLOC>
class tw_safeCircular {
public:

//	class iter {
//	public:
//		iter() : n(0) { }
//		bool getNext(T &fill);
//		bool atEnd();
//		friend class tw_safeCircular;
//	protected:
//		int n;
//	};

	tw_safeCircular( int size, bool initobj = false );
	tw_safeCircular() = delete;
	tw_safeCircular( tw_safeCircular<T,ALLOC> &o ) = delete;
#ifdef _TW_WINDOWS
	tw_safeCircular( HANDLE theHeap );
#endif
	void add( T &d );
	bool add( T &the_d, const int64_t usec_wait  );
	bool addIfRoom( T &the_d );
#ifdef TWLIB_HAS_MOVE_SEMANTICS
	void addMv( T &d );
	bool addMv( T &the_d, const int64_t usec_wait  );
	bool addMvIfRoom( T &the_d );
#endif

	// array like functions
	bool get(int n, T &d );
	bool set(int n, T &d );
	bool setMv(int n, T &d );


	// unimplemented-->
//	void transferFrom( tw_safeCircular<T,ALLOC> &other ); // transfer record from other to 'this' - can block
//	bool transferFromNoBlock( tw_safeCircular<T,ALLOC> &other ); // transfer record from other to 'this' -
//	                                               // wont block - false if would have blocked
	// <--unimplemented

	void cloneFrom( tw_safeCircular<T,ALLOC> &other );
#ifdef TWLIB_HAS_MOVE_SEMANTICS
	void transferFrom( tw_safeCircular<T,ALLOC> &other );
#endif


	bool remove( T &fill ); // true if got data
#ifdef TWLIB_HAS_MOVE_SEMANTICS
	bool removeMv( T &fill );
#endif
	bool removeOrBlock( T &fill ); // true if removed something
//	bool removeOrBlock( T &fill, TimeVal &t );
	bool removeOrBlock( T &fill, const int64_t usec_wait );
	bool removeMvOrBlock( T &fill );
	bool removeMvOrBlock( T &fill, const int64_t usec_wait );
	void clear(); // remove all nodes (does not delete T)
//	void unblock();  // unblock 1 blocking call
	void unblockAll(); // unblock all blocking calls
	void unblockAllRemovers();
	void disable();
	void enable();

	class iter final {
		friend class tw_safeCircular<T,ALLOC>;
	public:
		bool atEnd();
		bool data(T &);
		bool next();
		void release();
	protected:
		iter(tw_safeCircular &container);
		tw_safeCircular &owner;
		bool valid;
		int n; // count
		int p; // pointer into array
	};


	tw_safeCircular<T,ALLOC>::iter getIter();

	int remaining();
	~tw_safeCircular();
protected:
	bool isObjects;
	bool _reverse;
	TW_SemaTwoWay sema;
	bool enabled; // if enabled the FIFO can take new values
//	pthread_mutex_t newDataMutex; // thread safety for FIFO
//	pthread_cond_t newdataCond;
//	int _block_cnt;
	// should only be called with a sema lock...
	bool full() {
		return (!sema.countNoBlock());
	}
	// should only be called with a sema lock...
	int nextNextIn() {
		int n;
//		if(full()) nextOut = nextNextOut();
		if(_reverse) {
			n = nextIn - 1;
			if(n < 0) n = _size - 1;
		} else {
			n = nextIn + 1;
			if(n >= _size) n = 0;
		}
		return n;
	}
	// should only be called with a sema lock...
	int nextNextOut() {
		int n;
		if(_reverse) {
			n = nextOut - 1;
			if(n < 0) n = _size - 1;
		} else {
			n = nextOut + 1;
			if(n >= _size) n = 0;
		}
		return n;
	}
	// should only be called with a sema lock...
	int remain() {  // nextIn is always ahead of nextOut. circular
		return _size - sema.countNoBlock();
	}
	int nextIn;   // position to place next in value
	int nextOut;  // position to pull next out value
	int _size;     // size of the Circular buffer - only set once.
	T *data;
//	ALLOC *alloc;
#ifdef _TW_WINDOWS
	HANDLE hHeap;
#endif

public:
	// flips the circular around, so that the first element is now the last
	void reverse() {
		if(remain() < 1)
			return;

		_reverse = !(_reverse);
		int t = nextOut;
		nextOut = nextIn;
		nextIn = t;
	}

};


#ifdef TWLIB_HAS_MOVE_SEMANTICS
#endif

}

using namespace TWlib;

/////////////////////////////////////////////////////////////////////////
// thread safe FIFO


#ifdef _TW_WINDOWS
template <class T,class ALLOC>
tw_safeCircular<T,ALLOC>::tw_safeCircular( HANDLE theHeap ) : enabled( true ) {
	alloc = NULL;
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
	hHeap = theHeap;
}
#endif

template <class T,class ALLOC>
tw_safeCircular<T,ALLOC>::tw_safeCircular( int size, bool initobj ) : isObjects(initobj), _reverse(false), sema(size), enabled( true ),
	nextIn(-1), nextOut(0), _size(size), data(NULL) {
//	alloc = NULL;
//	pthread_mutex_init( &newDataMutex, NULL );
//	pthread_cond_init( &newdataCond, NULL );
	data = (T *) ALLOC::malloc( size * sizeof(T) );
	if(isObjects) {
		for(int n=0;n<size;n++) {
			T *p = data + n;
			p = new (p) T(); // placement new, if objects require an init.
		}
	}
}



/**
 * copies all values from the other circular buffer. This requires T to have a copy cstor
 */
template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::cloneFrom( tw_safeCircular &other ) {
	sema.lockSemaOnly();
	other.sema.lockSemaOnly();

	if(data) {
		if(isObjects) { // cleanup objects if needed
			for(int n=0;n<_size;n++) {
				data[n].~T();
			}
		}
		ALLOC::free(data);
	}

	sema.cloneFrom(other.sema); // copy count, etc.
	nextIn = other.nextIn;
	nextOut = other.nextOut;
	_size = other._size;
	isObjects = other.isObjects;
	_reverse = other._reverse;
	enabled = other.enabled; // if enabled the FIFO can take new values

	data = (T *) ALLOC::malloc( _size * sizeof(T) );

	if(isObjects) {
		for(int n=0;n<_size;n++) {
			T *p = data + n;
			p = new (p) T(other.data[n]); // placement new, if objects require an init.
		}
	}

	other.sema.releaseSemaLock();
	sema.releaseSemaLock();
}


#ifdef TWLIB_HAS_MOVE_SEMANTICS
/**
 * transfer all values from the other circular buffer. This requires T to have a move cstor
 */
template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::transferFrom( tw_safeCircular &other ) {
	sema.lockSemaOnly();
	other.sema.lockSemaOnly();

	if(data) {
		if(isObjects) { // cleanup objects if needed
			for(int n=0;n<_size;n++) {
				data[n].~T();
			}
		}
		ALLOC::free(data);
	}

	sema.cloneFrom(other.sema); // copy count, etc.
	nextIn = other.nextIn;
	nextOut = other.nextOut;
	_size = other._size;
	isObjects = other.isObjects;
	_reverse = other._reverse;
	enabled = other.enabled; // if enabled the FIFO can take new values

	data = (T *) ALLOC::malloc( _size * sizeof(T) );

	if(isObjects) {
		for(int n=0;n<_size;n++) {
			T *p = data + n;
			p = new (p) T(std::move(other.data[n])); // placement new, if objects require an init.
		}
	}

	// cleanup 'other'. Note, the
	other.sema.resetNoLock();
	other.nextIn = -1;
	other.nextOut = 0;

	other.sema.releaseSemaLock();
	sema.releaseSemaLock();
}

#endif

/**
 * enables the circular buffer, allowing the adding of new items.
 */
template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::enable() {
	sema.lockSemaOnly();
	enabled = true;
	sema.releaseSemaLock();
}

/**
 * disables the FIFO, preventing the adding of new items.
 * Items already in the FIFO can be pulled out.
 */
template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::disable() {
	sema.lockSemaOnly();
	enabled = false;
	sema.releaseSemaLock();
}


/**
 * Will block if queue is full
 */
template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::add( T &the_d ) {
	sema.acquireAndKeepLock();
	TW_CIRCULAR_DBG_OUT("post acquireAndKeepLock - add()");
	nextIn = nextNextIn();
	data[nextIn] = the_d;
	TW_CIRCULAR_DBG_OUT("remain post-add() data[%d]: %d",nextIn, remain());
	sema.releaseSemaLock();
//	unblock(); // let one blocking call know...
}

#ifdef TWLIB_HAS_MOVE_SEMANTICS
template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::addMv( T &the_d ) {
	sema.acquireAndKeepLock();
	TW_CIRCULAR_DBG_OUT("post acquireAndKeepLock - add(move)");
	nextIn = nextNextIn();
	data[nextIn] = std::move(the_d);
	TW_CIRCULAR_DBG_OUT("remain post-add(): %d",remain());
	sema.releaseSemaLock();
//	unblock(); // let one blocking call know...
}
#endif

template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::addIfRoom( T &the_d ) {
	bool ret = false;
	TW_CIRCULAR_DBG_OUT("acquireAndKeepLock - add()");
	if(sema.acquireAndKeepLockNoBlock()) {
		nextIn = nextNextIn();
		data[nextIn] = the_d;
		TW_CIRCULAR_DBG_OUT("remain post-add(): %d",remain());
		ret = true;
	} else {
		TW_CIRCULAR_DBG_OUT("not adding. no room: %d",remain());
	}
	sema.releaseSemaLock();
	return ret;
}

#ifdef TWLIB_HAS_MOVE_SEMANTICS
template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::addMvIfRoom( T &the_d ) {
	bool ret = false;
	TW_CIRCULAR_DBG_OUT("acquireAndKeepLock - add(move)");
	if(sema.acquireAndKeepLockNoBlock()) {
		nextIn = nextNextIn();
		data[nextIn] = std::move(the_d);
		TW_CIRCULAR_DBG_OUT("remain post-add(): %d",remain());
		ret = true;
	} else {
		TW_CIRCULAR_DBG_OUT("not adding. no room: %d",remain());
	}
	sema.releaseSemaLock();
	return ret;
}
#endif


// will block is queue is full!!
template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::add( T &the_d, const int64_t usec_wait  ) {
	bool ret = true;
	TW_CIRCULAR_DBG_OUT("acquireAndKeepLock - add()");
	int r = sema.acquireAndKeepLock(usec_wait);
	if(!r) {
		nextIn = nextNextIn();
		data[nextIn] = the_d;
		TW_CIRCULAR_DBG_OUT("remain post-add(): %d",remain());
	} else {
		TW_CIRCULAR_DBG_OUT("timeout / error on circular buffer: remain = %d",remain());
		ret = false;
	}
	sema.releaseSemaLock();
	return ret;
}

#ifdef TWLIB_HAS_MOVE_SEMANTICS
template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::addMv( T &the_d, const int64_t usec_wait  ) {
	bool ret = true;
	TW_CIRCULAR_DBG_OUT("acquireAndKeepLock - add(move)");
	int r = sema.acquireAndKeepLock(usec_wait);
	if(!r) {
		nextIn = nextNextIn();
		data[nextIn] = std::move(the_d);
		TW_CIRCULAR_DBG_OUT("remain post-add(): %d",remain());
	} else {
		TW_CIRCULAR_DBG_OUT("timeout / error on circular buffer: remain = %d",remain());
		ret = false;
	}
	sema.releaseSemaLock();
	return ret;
}
#endif

template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::get(int n, T &d ) {
	bool ret = false;
	sema.lockSemaOnly();
	if((n >= 0) && (n < remain())) {
		int c = 0;
		int p = nextOut - 1;
		if(_reverse) {
			p = nextIn - 2;
			if(p < 0) p = _size -1;
		}
		p++;
		while(c != n) {
			if(p >= _size) p = 0;
			p++; c++;
		}
		d = data[p];
		ret = true;
	}
	sema.releaseSemaLock();
	return ret;
}

template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::set(int n, T &d ) {
	bool ret = false;
	sema.lockSemaOnly();
	if((n >= 0) && (n < remain())) {
		int c = 0;
		int p = nextOut - 1;
		if(_reverse) {
			p = nextIn - 2;
			if(p < 0) p = _size -1;
		}
		p++;
		while(c != n) {
			if(p >= _size) p = 0;
			p++; c++;
		}
		data[p] = d;
		ret = true;
	}
	sema.releaseSemaLock();
	return ret;
}

#ifdef TWLIB_HAS_MOVE_SEMANTICS
template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::setMv(int n, T &d ) {
	bool ret = false;
	sema.lockSemaOnly();
	if((n >= 0) && (n < remain())) {
		int c = 0;
		int p = nextOut - 1;
		if(_reverse) {
			p = nextIn -2;
			if(p < 0) p = _size -1;
		}
		p++;
		while(c != n) {
			if(p >= _size) p = 0;
			p++; c++;
		}
		data[p] = std::move(d);
		ret = true;
	}
	sema.releaseSemaLock();
	return ret;
}
#endif



template <class T,class ALLOC>
typename tw_safeCircular<T,ALLOC>::iter tw_safeCircular<T,ALLOC>::getIter() {
	return tw_safeCircular<T,ALLOC>::iter(*this);
}

template <class T,class ALLOC>
tw_safeCircular<T,ALLOC>::iter::iter(tw_safeCircular<T,ALLOC> &c) :
	owner(c), valid(false), n(0), p(0)
{
	c.sema.lockSemaOnly();
	p = c.nextOut;
	if(c.remain() > 0)
		valid = true;
}

template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::iter::next() {
	n++;
	if(n < owner.remain()) {
		p++;
		if(p >= owner._size) p = 0;
		return true;
	} else {
		valid = false;
		return false;
	}
}

template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::iter::data(T &d) {
	if(valid && n < owner.remain()) {
		d = owner.data[p];
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::iter::atEnd() {
	if(valid && n < owner.remain()) {
		return false;
	} else
		return true;
}

template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::iter::release() {
	owner.sema.releaseSemaLock();
	valid = false;
}


template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::remove( T &fill ) {
	bool ret = true;
	sema.lockSemaOnly();
	int r = remain();
	if(r > 0) {
		sema.releaseWithoutLock();
		fill = data[nextOut];
		nextOut = nextNextOut();
		if(r == 1) { // if we are now empty...
			nextIn = -1; nextOut = 0; _reverse = false;
		}
	} else {
		ret = false;
	}
	sema.releaseSemaLock();
	return ret;
}


#ifdef TWLIB_HAS_MOVE_SEMANTICS
template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::removeMv( T &fill ) {
	bool ret = true;
	sema.lockSemaOnly();
	int r = remain();
	if(r > 0) {
		sema.releaseWithoutLock();
		fill = std::move(data[nextOut]);
		nextOut = nextNextOut();
		if(r == 1) { // if we are now empty...
			nextIn = -1; nextOut = 0; _reverse = false;
		}
	} else {
		ret = false;
	}
	sema.releaseSemaLock();
	return ret;
}
#endif

template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::removeOrBlock( T &fill ) {
	bool ret = true;
	sema.lockSemaOnly();
	TW_CIRCULAR_DBG_OUT("removeOrBlock.. remain = %d", remain());
	int rm = remain();
	if(rm > 0) {
		sema.releaseWithoutLock();
		TW_CIRCULAR_DBG_OUT("   ...removeOrBlock(2).. data[%d] remain = %d", nextOut, remain());
		fill = data[nextOut];
		nextOut = nextNextOut();
		if(rm == 1) { // if we are now empty...
			nextIn = -1; nextOut = 0; _reverse = false;
		}
		sema.releaseSemaLock();
	} else {
		TW_CIRCULAR_DBG_OUT("  ...removeOrBlock(3) waitForAcquirers. remain = %d", remain());
		int r = sema.waitForAcquirersKeepLock(false); // unlocks while waiting for acquire
		if(r == 0) {
			TW_CIRCULAR_DBG_OUT("  ...waitForAcquirers complete. data[%d] remain = %d", nextOut, remain());
			fill = data[nextOut];
			if(remain() == 0) { // if we are now empty...
				nextIn = -1; nextOut = 0; _reverse = false;
			}
			nextOut = nextNextOut();
			sema.releaseWithoutLock();
		} else {
			ret = false;
			TW_CIRCULAR_DBG_OUT("  ...waitForAcquirers error (%d). remain = %d", r, remain());
		}
		sema.releaseSemaLock();
	}
	return ret;
}

template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::removeOrBlock( T &fill, const int64_t usec_wait ) {
	bool ret = true;
	sema.lockSemaOnly();
	TW_CIRCULAR_DBG_OUT("removeOrBlock.. remain = %d", remain());
	int rm = remain();
	if(rm > 0) {
		sema.releaseWithoutLock();
		TW_CIRCULAR_DBG_OUT("   ...removeOrBlock(2).. data[%d] remain = %d", nextOut, remain());
		fill = data[nextOut];
		nextOut = nextNextOut();
		if(rm == 1) { // if we are now empty...
			nextIn = -1; nextOut = 0; _reverse = false;
		}
		sema.releaseSemaLock();
	} else {
		TW_CIRCULAR_DBG_OUT("  ...removeOrBlock(%d) waitForAcquirers", remain());
		int r = sema.waitForAcquirersKeepLock(usec_wait, false); // unlocks while waiting for acquire
		if(r == 0) {
			TW_CIRCULAR_DBG_OUT("  ...waitForAcquirers complete. remain = %d", remain());
			fill = data[nextOut];
			if(remain() == 0) { // if we are now empty...
				nextIn = -1; nextOut = 0; _reverse = false;
			}
			TW_CIRCULAR_DBG_OUT("  ...removeOrBlock(3). data[%d]",nextOut);
			nextOut = nextNextOut();
			sema.releaseWithoutLock();
		} else {
			ret = false;
			TW_CIRCULAR_DBG_OUT("  ...waitForAcquirers timeout or error (%d). remain = %d", r, remain());
		}
		sema.releaseSemaLock();
	}
	return ret;
}


#ifdef TWLIB_HAS_MOVE_SEMANTICS
template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::removeMvOrBlock( T &fill ) {
	bool ret = true;
	sema.lockSemaOnly();
	TW_CIRCULAR_DBG_OUT("removeOrBlock.. remain = %d", remain());
	int rm = remain();
	if(rm > 0) {
		sema.releaseWithoutLock();
		TW_CIRCULAR_DBG_OUT("   ...removeOrBlock(2).. data[%d] remain = %d", nextOut, remain());
		fill = std::move(data[nextOut]);
		nextOut = nextNextOut();
		if(rm == 1) { // if we are now empty...
			nextIn = -1; nextOut = 0; _reverse = false;
		}
		sema.releaseSemaLock();
	} else {
		TW_CIRCULAR_DBG_OUT("  ...removeOrBlock(3) waitForAcquirers. remain = %d", remain());
		int r = sema.waitForAcquirersKeepLock(false); // unlocks while waiting for acquire
		if(r == 0) {
			TW_CIRCULAR_DBG_OUT("  ...waitForAcquirers complete. data[%d] remain = %d", nextOut, remain());
			fill = std::move(data[nextOut]);
			if(remain() == 0) { // if we are now empty...
				nextIn = -1; nextOut = 0; _reverse = false;
			}
			nextOut = nextNextOut();
			sema.releaseWithoutLock();
		} else {
			ret = false;
			TW_CIRCULAR_DBG_OUT("  ...waitForAcquirers error (%d). remain = %d", r, remain());
		}
		sema.releaseSemaLock();
	}
	return ret;
}
template <class T,class ALLOC>
bool tw_safeCircular<T,ALLOC>::removeMvOrBlock( T &fill, const int64_t usec_wait ) {
	bool ret = true;
	sema.lockSemaOnly();
	TW_CIRCULAR_DBG_OUT("removeOrBlock.. remain = %d", remain());
	int rm = remain();
	if(rm > 0) {
		sema.releaseWithoutLock();
		TW_CIRCULAR_DBG_OUT("   ...removeOrBlock(2).. data[%d] remain = %d", nextOut, remain());
		fill = std::move(data[nextOut]);
		nextOut = nextNextOut();
		if(rm == 1) { // if we are now empty...
			nextIn = -1; nextOut = 0; _reverse = false;
		}
		sema.releaseSemaLock();
	} else {
		TW_CIRCULAR_DBG_OUT("  ...removeOrBlock(3) waitForAcquirers. remain = %d", remain());
		int r = sema.waitForAcquirersKeepLock(usec_wait, false); // unlocks while waiting for acquire
		if(r == 0) {
			TW_CIRCULAR_DBG_OUT("  ...waitForAcquirers complete. data[%d] remain = %d", nextOut, remain());
			fill = std::move(data[nextOut]);
			if(remain() == 0) { // if we are now empty...
				nextIn = -1; nextOut = 0; _reverse = false;
			}
			nextOut = nextNextOut();
			sema.releaseWithoutLock();
		} else {
			ret = false;
			TW_CIRCULAR_DBG_OUT("  ...waitForAcquirers error (%d). remain = %d", r, remain());
		}
		sema.releaseSemaLock();
	}
	return ret;
}

#endif




template <class T,class ALLOC>
int tw_safeCircular<T,ALLOC>::remaining(void) {
	int ret;
	sema.lockSemaOnly();
	ret = remain();
	sema.releaseSemaLock();
	return ret;
}

template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::clear() { // delete all remaining links (and hope someone took care of the data in each of those)
	sema.lockSemaOnly();
	if(isObjects) { // cleanup objects if needed. and rebuild empty ones
		for(int n=0;n<_size;n++) {
			data[n].~T();
		}
		for(int n=0;n<_size;n++) {
			T *p = data + n;
			p = new (p) T(); // placement new, if objects require an init.
		}
	}
	nextIn = -1; nextOut = 0;  // reset back to beginning (just like cstor)
	_reverse = false; enabled=true;
	sema.resetNoLock();
	sema.releaseSemaLock();
}

/**
 * Unblocks everything. NOTE: The queue is not safe to use after calling this, and should be discarded.
 * Use unblockAllRemovers() to safely unblock all removal calls.
 */
template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::unblockAll() {
	sema.releaseAll();
}

/**
 * Unblocks all calls involving element removal. Those calls will fail gracefully with a false.
 */
template <class T,class ALLOC>
void tw_safeCircular<T,ALLOC>::unblockAllRemovers() {
	sema.releaseAllAcquireLocks();
}


template <class T,class ALLOC>
tw_safeCircular<T,ALLOC>::~tw_safeCircular() { // delete all remaining links (and hope someone took care of the data in each of those)
	unblockAll();
	sema.lockSemaOnly();
	if(isObjects) { // cleanup objects if needed
		for(int n=0;n<_size;n++) {
			data[n].~T();
		}
	}
	sema.releaseSemaLock();
}



#endif // _TW_FIFO
