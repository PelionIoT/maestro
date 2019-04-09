

#ifndef TW_AUTOPOINTER
#define TW_AUTOPOINTER

#include <pthread.h>
//#include <ace/Malloc_Base.h>
//#include <ace/Message_Block.h>
#include "tw_alloc.h"
//#include "tw_fifo.h"

// reference counts
// callbacks when deleted.

// the callbacks are called right before the object is deleted.
// the callback must not need the data the autoPointer is pointing to after to the callback returns.


/* Behavior: An autoPointer points at an object, but keeps a reference count. When the autoPointer is created, its reference is 1.
// a release() causes a refCount--
// a getP() causes a refCount++
// users should call release when done. The owner of the data autoPointer is pointing to should release() only when its done with it,
// and it knows the other callers who need it have had a chance to do a getP() - lest the refCount go down to zero.
 if the autoPointer is destroyed, the data is deleted immediately irregardless of refCount.
 Before the data is deleted, each callback is called, allowing callers to be informed if the data goes away.
 B/c of this, the deleting should happen in a non speed-critical area*/
namespace TWlib {

// class you are pointing to
template <class T>
class autoPointer {
public:
	typedef void(*autoPointerCallback)(void *obj);

//	static autoPointer create( void *obj, ACE_Allocator *a = NULL );
	autoPointer( T *d,  Allocator<Alloc_Std> *a = NULL );
	void reset( T *d,  Allocator<Alloc_Std> *a=NULL );
	bool attached();
	autoPointer( ); // so we can create null ones, and on the stack
	autoPointer<T> &dupP(void); // duplicate pointer - get data pointing to (and increment the reference)
	T* operator->() { return D; } // return the data without incrementing a reference.
	operator T* () { return D; }  // return the data without incrementing a reference.
	T* dat() { return D; } // return the data without incrementing a reference - readability function
	void release(void); // decrement references
	void registerCallback( autoPointerCallback cb );
	~autoPointer();

protected:
	void destroy();
	T *D;
	int refCount;
	pthread_mutex_t dataMutex; // thread safety for autoPointer
	Allocator<Alloc_Std> *alloc;
	class callbackEntry {
	public:
		autoPointerCallback _cb;
		callbackEntry *_next;
		callbackEntry() : _next(NULL), _cb(NULL) {}
	};
	void removeAllCallbacks();
	void addCallback(autoPointerCallback cb);
	callbackEntry *firstCallback;
//	tw_FIFO<autoPointerCallback, Allocator<Alloc_Std> > *callbackList;
};

}

/*
 * autoPointer.cpp
 *
 *  Created on: Dec 31, 2009
 *      Author: ed
 */

using namespace TWlib;
// upon create a the autoPointer is at reference 1.
template <class T>
autoPointer<T>::autoPointer(T *d,  Allocator<Alloc_Std> *a) : firstCallback(NULL), refCount( 0 ), alloc( a ), D(NULL) {
	pthread_mutex_init( &dataMutex, NULL );
	reset(d, a);
}

template <class T>
autoPointer<T>::autoPointer() : firstCallback( NULL ), refCount( 0 ), alloc( NULL ), D(NULL) {
	pthread_mutex_init( &dataMutex, NULL );
}

template <class T>
void autoPointer<T>::removeAllCallbacks() {
	callbackEntry *nex = firstCallback;
	callbackEntry *here = NULL;
	while(nex) {
		here = nex;
		nex = nex->_next;
		delete here;
	}
}

template <class T>
void autoPointer<T>::addCallback(autoPointerCallback cb) {
	callbackEntry *nex = firstCallback;
	if(nex) {
		while(nex->_next) {
			nex = nex->_next;
		}
		nex->_next = new callbackEntry();
		nex->_next->_cb = cb;
	} else {
		firstCallback = new callbackEntry();
		firstCallback->_cb = cb;
	}
}


template <class T>
void autoPointer<T>::reset(T *d,  Allocator<Alloc_Std> *a) {
	D = d;
//	pthread_mutex_init( &dataMutex, NULL );
	refCount = 1;
	alloc = a;
//	if(D != NULL)
//		callbackList = new tw_FIFO<autoPointerCallback, Allocator<Alloc_Std> >();
//	else
//		callbackList = NULL;
	removeAllCallbacks();
	firstCallback = NULL;
}

// is the autoPointer actually pointing to something?
template <class T>
bool autoPointer<T>::attached() {
	bool ret = false;
	pthread_mutex_lock(&dataMutex);
	if(D != NULL)
		ret = true;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T>
autoPointer<T> &autoPointer<T>::dupP(void) {
//	T *ret = NULL;
	pthread_mutex_lock(&dataMutex);
//	ret = D;
	refCount++;
	pthread_mutex_unlock(&dataMutex);
	return *this;
}

template <class T>
void autoPointer<T>::release(void) {
	pthread_mutex_lock(&dataMutex);
	refCount--;
	if(refCount < 1)
		destroy();
	pthread_mutex_unlock(&dataMutex);
}

template <class T>
void autoPointer<T>::registerCallback( autoPointerCallback cb ) {
	pthread_mutex_lock(&dataMutex);
//	if(callbackList)
//		callbackList->add(cb);
	addCallback(cb);
	pthread_mutex_unlock(&dataMutex);
}

template <class T>
void autoPointer<T>::destroy() { // ** MUST WRAP with mutex
	callbackEntry *walk = firstCallback;
	callbackEntry *last = NULL;
	while (walk) {
		walk->_cb(D); // call each callback
		last = walk;
		walk = walk->_next;
		delete last;
	}
//	delete callbackList;
	delete D; // call destructor of Data being pointed to
	D = NULL; // ensure its NULL
	// run through call backs
	// delete using ACE_Allocator
}

template <class T>
autoPointer<T>::~autoPointer() {
	pthread_mutex_lock(&dataMutex);
	if (D != NULL) {
		destroy();
	}
	pthread_mutex_unlock(&dataMutex);
}

#endif


