
// tupperware container lib

// FIFO: a simple class to handle a fifo list of void pointers.
#include <pthread.h>
#include <errno.h>
//#include <ace/Malloc_Base.h>
//#include <ace/Message_Block.h>
//#include <ace/OS_Memory.h>

#include <TW/tw_log.h>

// #ifdef 'g++' whatever that is...
#include <new>
// #endif

#include <TW/tw_utils.h> // TimeVal
#include <TW/tw_sema.h>
#include <TW/tw_alloc.h>
//#include "logging.h"

#ifdef TWLIB_HAS_MOVE_SEMANTICS
#include <utility>
#endif

#ifndef _TW_FIFO
#define _TW_FIFO


namespace TWlib {

#ifdef TWLIB_HAS_MOVE_SEMANTICS

/**
 * Unlimited sized FIFO. This is a thread-safe, blocking queue using pthread conditions.
 * T must support:
 * self assignment: operator= (T &&a, T &&b)  rvalue assignment required (regular assignment *not* required)
 * default constructor
 */
template <class T, class ALLOC >
class tw_safeFIFOmv {
public:
	struct tw_FIFO_link {
		T d;
		tw_FIFO_link *next;
//		tw_FIFO_link *prev;
		tw_FIFO_link();
		void init_link( T &the_d );
		void init_link();
	};

	class iter {
	public:
		iter() : look(NULL) { };
		T &el();
		void next();
		bool atEnd();
		friend class tw_safeFIFOmv;
	protected:
		tw_FIFO_link *look;
	};

	tw_safeFIFOmv( void );
	tw_safeFIFOmv( ALLOC *_a );
	tw_safeFIFOmv( tw_safeFIFOmv<T,ALLOC> &o );
#ifdef _TW_WINDOWSs
	tw_safeFIFOmv( HANDLE theHeap );
#endif
	void add( T &d );
#if __cplusplus >= 201103L
	void add( T &&d );
#endif

	void addToHead( T &d );
	void transferFrom( tw_safeFIFOmv<T,ALLOC> &other ); // transfer record from other to 'this' - can block
	bool transferFromNoBlock( tw_safeFIFOmv<T,ALLOC> &other ); // transfer record from other to 'this' -
	                                               // wont block - false if would have blocked
	T *addEmpty();
	bool peek( T &fill ); // true if got valid value - look at next, dont remove
	bool peekOrBlock( T &fill ); // true if got data - look at next, dont remove
	bool peekOrBlock( T &fill, TimeVal &t );
	bool remove( T &fill ); // true if got data
	bool remove_mv( T &fill );
	bool removeOrBlock( T &fill ); // true if removed something
	bool removeOrBlock( T &fill, TimeVal &t );
	void clearAll(); // remove all nodes (does not delete T)
	void unblock();  // unblock 1 blocking call
	void unblockAll(); // unblock all blocking calls
	void disable();
	void enable();
	void startIter( iter &i );
	tw_safeFIFOmv &operator=( const tw_safeFIFOmv &o );

//	void removeAtIter( iter &i );
	void releaseIter( iter &i );
	int remaining();
	~tw_safeFIFOmv();
protected:
	bool enabled; // if enabled the FIFO can take new values
	pthread_mutex_t dataMutex; // thread safety for FIFO
	pthread_cond_t newdataCond;
	int _block_cnt;
	int remain;
	tw_FIFO_link *in; // add from this end (tail)
	tw_FIFO_link *out; // remove from this end (head)
	ALLOC *alloc;
#ifdef _TW_WINDOWS
	HANDLE hHeap;
#endif
};

#endif
/**
 * Unlimited sized FIFO. This is a thread-safe, blocking queue using pthread conditions.
 * T must support:
 * self assignment: operator= (T &a, T &b)
 * default constructor
 */
template <class T, class ALLOC >
class tw_safeFIFO {
public:
	struct tw_FIFO_link {
		T d;
		tw_FIFO_link *next;
//		tw_FIFO_link *prev;
		tw_FIFO_link();
		void init_link( T &the_d );
		void init_link();
	};

	class iter {
	public:
		iter() : look(NULL) { }
		bool getNext(T &fill);
		bool atEnd();
		friend class tw_safeFIFO;
	protected:
		tw_FIFO_link *look;
	};

	tw_safeFIFO( void );
	tw_safeFIFO( ALLOC *_a );
	tw_safeFIFO( tw_safeFIFO<T,ALLOC> &o );
#ifdef _TW_WINDOWSs
	tw_safeFIFO( HANDLE theHeap );
#endif
	void add( T &d );
#if __cplusplus >= 201103L
	void add( T &&d );
#endif

	void addToHead( T &d );
	void transferFrom( tw_safeFIFO<T,ALLOC> &other ); // transfer record from other to 'this' - can block
	bool transferFromNoBlock( tw_safeFIFO<T,ALLOC> &other ); // transfer record from other to 'this' -
	                                               // wont block - false if would have blocked
	T *addEmpty();
	bool peek( T &fill ); // true if got valid value - look at next, dont remove
	bool peekOrBlock( T &fill ); // true if got data - look at next, dont remove
	bool peekOrBlock( T &fill, TimeVal &t );
	bool remove( T &fill ); // true if got data
#ifdef TWLIB_HAS_MOVE_SEMANTICS
	bool remove_mv( T &fill );
#endif
	bool removeOrBlock( T &fill ); // true if removed something
	bool removeOrBlock( T &fill, TimeVal &t );
	void clearAll(); // remove all nodes (does not delete T)
	void unblock();  // unblock 1 blocking call
	void unblockAll(); // unblock all blocking calls
	void disable();
	void enable();
	void startIter( iter &i );
	tw_safeFIFO &operator=( const tw_safeFIFO &o );

//	void removeAtIter( iter &i );
	void releaseIter( iter &i );
	int remaining();
	~tw_safeFIFO();
protected:
	bool enabled; // if enabled the FIFO can take new values
	pthread_mutex_t dataMutex; // thread safety for FIFO
	pthread_cond_t newdataCond;
	int _block_cnt;
	int remain;
	tw_FIFO_link *in; // add from this end (tail)
	tw_FIFO_link *out; // remove from this end (head)
	ALLOC *alloc;
#ifdef _TW_WINDOWS
	HANDLE hHeap;
#endif
};

/**
 * This is like the TW_safeFIFO class, expect there is a bounded limit on the number of units it can hold.
 */
template <class T,class ALLOC>
class tw_bndSafeFIFO {
protected:
	tw_safeFIFO<T,ALLOC> _fifo; // we use the above FIFO class do the most fifo work here...
public:

	typedef typename tw_safeFIFO<T,ALLOC>::iter iter;

	tw_bndSafeFIFO( void );
	tw_bndSafeFIFO( int maxsize );
	tw_bndSafeFIFO( int maxsize, ALLOC *_a );
	tw_bndSafeFIFO( int masize, tw_safeFIFO<T,ALLOC> &o );
	tw_bndSafeFIFO( tw_bndSafeFIFO<T,ALLOC> &o );
//#ifdef _TW_WINDOWSs
//	tw_safeFIFO( HANDLE theHeap );
//#endif
	void add( T &d );
	void addToHead( T &d );
	void transferFrom( tw_safeFIFO<T,ALLOC> &other ); // transfer record from other to 'this' - can block
	bool transferFromNoBlock( tw_safeFIFO<T,ALLOC> &other ); // transfer record from other to 'this' -
	                                               // wont block - false if would have blocked
	T *addEmpty();
	bool peek( T &fill ); // true if got valid value - look at next, dont remove
	bool peekOrBlock( T &fill ); // true if got data - look at next, dont remove
	bool peekOrBlock( T &fill, TimeVal &t );
	bool remove( T &fill ); // true if got data
	bool removeOrBlock( T &fill ); // true if removed something
	bool removeOrBlock( T &fill, TimeVal &t );
	void clearAll(); // remove all nodes (does not delete T)
	void unblockRemoveCalls();  // unblock 1 blocking call
	void unblockAll(); // unblock all blocking calls
	void disable();
	void enable();

	void startIter( iter &i );
	tw_bndSafeFIFO<T,ALLOC> &operator=( const tw_bndSafeFIFO<T,ALLOC> &o );

//	void removeAtIter( iter &i );
	void releaseIter( iter &i );
	int remaining();
	~tw_bndSafeFIFO();
protected:
	TW_Sema *_sizeSema; // use this semaphore to not over fill the FIFO
	int _max;
};

/** T must support:
 * self assignment: operator= (T &a, T &b)
 * default constructor
 */
template <class T,class ALLOC>
class tw_FIFO {
public:
	struct tw_FIFO_link {
		T d;
		tw_FIFO_link *next;
//		tw_FIFO_link *prev;
		tw_FIFO_link();
		void init_link( T &the_d );
		void init_link();
	};

	class iter {
	public:
		iter() : look(NULL) { }
		bool getNext(T &fill);
		bool atEnd();
		friend class tw_FIFO;
	protected:
		tw_FIFO_link *look;
	};

	tw_FIFO( void );
	tw_FIFO( ALLOC *_a );
	tw_FIFO( tw_FIFO<T,ALLOC> &o );
#ifdef _TW_WINDOWS
	tw_FIFO( HANDLE theHeap );
#endif
	void add( T &d );
	void addToHead( T &d );
	void transferFrom( tw_FIFO<T,ALLOC> &other );
	T *addEmpty();
	bool peek( T &fill ); // look at next, dont remove
	bool remove( T &fill );
	int remaining();
	void startIter( iter &i );
	void releaseIter( iter &i );
	tw_FIFO &operator=( const tw_FIFO &o );
	void clearAll(); // remove all nodes (does not delete T)
	void disable();
	void enable();
	~tw_FIFO();
private:
	bool enabled;
	tw_FIFO_link *in; // add from this end
	tw_FIFO_link *out; // remove from this end
	int remain;
	ALLOC *alloc;
#ifdef _TW_WINDOWS
	HANDLE hHeap;
#endif
//	pthread_mutex_t dataMutex; // thread safety for FIFO
};

}

using namespace TWlib;

template <class T,class ALLOC>
bool tw_safeFIFO<T,ALLOC>::iter::getNext(T &fill) {
	if(look) {
		fill = look->d;
		look = look->next;
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
inline bool tw_safeFIFO<T,ALLOC>::iter::atEnd() {
	if(look)
		return false;
	else
		return true;
}

template <class T,class ALLOC>
bool tw_FIFO<T,ALLOC>::iter::getNext(T &fill) {
	if(look) {
		fill = look->d;
		look = look->next;
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
inline bool tw_FIFO<T,ALLOC>::iter::atEnd() {
	if(look)
		return false;
	else
		return true;
}

#ifdef TWLIB_HAS_MOVE_SEMANTICS

template <class T,class ALLOC>
T &tw_safeFIFOmv<T,ALLOC>::iter::el() {
	if(look) {
		return look->d;
	} else {
		TW_ERROR("tw_safeFIFOmv --- went past end of list\n",NULL);
#ifdef __EXCEPTIONS
		throw false;
#endif
	}
}

template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::iter::next() {
	if(look)
		look = look->next;
}


template <class T,class ALLOC>
inline bool tw_safeFIFOmv<T,ALLOC>::iter::atEnd() {
	if(look)
		return false;
	else
		return true;
}

#endif


template <class T,class ALLOC>
tw_FIFO<T,ALLOC>::tw_FIFO_link::tw_FIFO_link(void) {
//	d = (T *) NULL;
//	d = NULL;
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

template <class T,class ALLOC>
void tw_FIFO<T,ALLOC>::tw_FIFO_link::init_link( T &the_d ) {
	::new((void*)&d) T(); // must explicity call the constructor
	d = the_d;
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

template <class T,class ALLOC>
void tw_FIFO<T,ALLOC>::tw_FIFO_link::init_link() {
	::new((void*)&d) T(); // must explicity call the constructor
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

#ifdef _TW_WINDOWS
template <class T,class ALLOC>
tw_FIFO<T,ALLOC>::tw_FIFO( HANDLE theHeap ) : enabled ( true ) {
	alloc = NULL;
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
	hHeap = theHeap;
}
#endif

template <class T,class ALLOC>
tw_FIFO<T,ALLOC>::tw_FIFO( void ) : enabled( true ) {
	alloc = NULL;
//	pthread_mutex_init( &dataMutex, NULL );
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
}

template <class T,class ALLOC>
tw_FIFO<T,ALLOC>::tw_FIFO( tw_FIFO<T,ALLOC> &o ) : enabled( true ) {
	alloc = NULL;
//	pthread_mutex_init( &dataMutex, NULL );
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;

	this->transferFrom( o );
}


template <class T,class ALLOC>
tw_FIFO<T,ALLOC>::tw_FIFO( ALLOC *a ) : enabled( true ) {
	alloc = a;
//	pthread_mutex_init( &dataMutex, NULL );
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
}

template <class T,class ALLOC>
inline void tw_FIFO<T,ALLOC>::startIter( iter &i ) {
	i.look = out;
}

template <class T,class ALLOC>
inline void tw_FIFO<T,ALLOC>::releaseIter( iter &i ) {
	i.look = NULL;
}


template <class T,class ALLOC>
void tw_FIFO<T,ALLOC>::add( T &the_d ) {
	if(enabled) {
	tw_FIFO_link *newlink;
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
//	pthread_mutex_lock(&dataMutex);
	if(in)
		in->next = newlink;
	in = newlink;
	if(!out) {
		out = in;
	}
	remain++;
//	pthread_mutex_unlock(&dataMutex);
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif
}

template <class T,class ALLOC>
void tw_FIFO<T,ALLOC>::addToHead( T &the_d ) {
	if(enabled) {
	tw_FIFO_link *newlink;
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
//	pthread_mutex_lock(&dataMutex);
//	if(in)
//		in->next = newlink;
//	in = newlink;
	newlink->d = the_d;
	if(!out) {
		out = newlink;
	} else {
		newlink->next = out;
		out = newlink;
	}
	if(!in)
		in = out;
	remain++;
//	pthread_mutex_unlock(&dataMutex);
//	unblock(); // let one blocking call know...
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif
}


/** pulls the items queued out of 'other' and placs them in this tw_FIFO
 *
 * @param other
 */
template <class T,class ALLOC>
void tw_FIFO<T,ALLOC>::transferFrom( tw_FIFO<T,ALLOC> &other ) {
	if(enabled) {
	if(other.out) { // take care of first link - in case 'this' is empty
		if(in) {
			in->next = other.out;
			in = in->next;
		} else
			in = other.out;
		remain++;
	}

	if(!out) // if 'this' was empty - setup the out pointer
		out = in;

	while(in->next) { // count the number of nodes we just tacked on...
//		other.out = other.out->next;
		in = in->next; // in should be pointing at the same thing as other.out;
		remain++;
	}

#ifdef _TW_FIFO_DEBUG_ON
	if(in->next != NULL) // verify - in->next should be NULL
		TW_ERROR("tw_FIFO --- error in transferFrom\n",NULL);
#endif

	other.out = NULL; // that list is now empty
	other.in = NULL;
	other.remain = 0;
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif
}

template <class T,class ALLOC>
void tw_FIFO<T,ALLOC>::enable() {
	enabled = true;
}

template <class T,class ALLOC>
void tw_FIFO<T,ALLOC>::disable() {
	enabled = false;
}



template <class T,class ALLOC>
T *tw_FIFO<T,ALLOC>::addEmpty() {
	if(enabled) {
	tw_FIFO_link *newlink;
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
	newlink->init_link();
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
//	pthread_mutex_lock(&dataMutex);
	if(in)
		in->next = newlink;
	in = newlink;
	if(!out) {
		out = in;
	}
	remain++;
//	pthread_mutex_unlock(&dataMutex);
	return &(newlink->d);
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif
	return NULL;
}



template <class T,class ALLOC>
bool tw_FIFO<T,ALLOC>::remove( T &fill ) {
	bool ret = false;
//	pthread_mutex_lock(&dataMutex);
	if(in==out)
		in = NULL;
	if(out) {
		ret = true;
		fill = out->d;
		tw_FIFO_link *oldout = out;
		out = out->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		ALLOC::free( oldout );
#endif
		remain--;
	}
	//	if(!out)
	//	out = in;
//	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_FIFO<T,ALLOC>::peek( T &fill ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
//	pthread_mutex_lock(&dataMutex);
	if(out) {
		ret = true;
		fill = out->d;
	}
//	pthread_mutex_unlock(&dataMutex);
	return ret;
}



template <class T,class ALLOC>
int tw_FIFO<T,ALLOC>::remaining(void) {
	int ret;
//	pthread_mutex_lock(&dataMutex);
	ret = remain;
//	pthread_mutex_unlock(&dataMutex);
	return remain;

}

// copies other tw_safeFIFO - does not copy Allocator
template <class T,class ALLOC>
tw_FIFO<T,ALLOC> &tw_FIFO<T,ALLOC>::operator=( const tw_FIFO<T,ALLOC> &o ) {
	tw_FIFO_link *look;
	tw_FIFO_link *prev;
	tw_FIFO_link *newlink;

	this->enabled = o.enabled;
	this->clearAll(); // clear anything that might be there


	this->remain = 0;
	look = o.out;

	if(look) {
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
	newlink->init_link(look->d);
	prev = newlink;
	this->out = newlink;
	look = look->next;
	this->remain++;
	}

	while(look) {
		newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
		newlink->init_link(look->d);
		prev->next = newlink; // link to link behind us
		prev = newlink;       // move forward
		look = look->next;    // move the source forward
		this->remain++;
	}

	this->in = prev;
	return *this;
}

template <class T,class ALLOC>
void tw_FIFO<T,ALLOC>::clearAll() { // delete all remaining links (and hope someone took care of the data in each of those)
	int ret;
	tw_FIFO_link *n = NULL;
//	pthread_mutex_lock(&dataMutex);
	while(out) {
		n = out->next;
		ALLOC::free(out);
		out = n;
	}

	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
//	pthread_mutex_unlock(&dataMutex);
}


template <class T,class ALLOC>
tw_FIFO<T,ALLOC>::~tw_FIFO() { // delete all remaining links (and hope someone took care of the data in each of those)
//	int ret;
	tw_FIFO_link *n = NULL;
//	pthread_mutex_lock(&dataMutex);
	while(out) {
		n = out->next;
		ALLOC::free(out);
		out = n;
	}
//	pthread_mutex_unlock(&dataMutex);
}
/////////////////////////////////////////////////////////////////////////
// thread safe FIFO

template <class T,class ALLOC>
tw_safeFIFO<T,ALLOC>::tw_FIFO_link::tw_FIFO_link(void) {
//	d = (T *) NULL;
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::tw_FIFO_link::init_link( T &the_d ) {
	::new((void*)&d) T(); // must explicity call the constructor
	d = the_d;
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::tw_FIFO_link::init_link( ) {
	::new((void*)&d) T(); // must explicity call the constructor
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}


#ifdef _TW_WINDOWS
template <class T,class ALLOC>
tw_safeFIFO<T,ALLOC>::tw_safeFIFO( HANDLE theHeap ) : enabled( true ) {
	alloc = NULL;
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
	hHeap = theHeap;
}
#endif

template <class T,class ALLOC>
tw_safeFIFO<T,ALLOC>::tw_safeFIFO( void ) : enabled( true ) {
	alloc = NULL;
	_block_cnt = 0;
	pthread_mutex_init( &dataMutex, NULL );
	pthread_cond_init( &newdataCond, NULL );
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
}


template<class T, class ALLOC>
tw_safeFIFO<T,ALLOC>::tw_safeFIFO(tw_safeFIFO<T,ALLOC> &o) : enabled( true ) {
	alloc = NULL;
	_block_cnt = 0;
	pthread_mutex_init(&dataMutex, NULL);
	pthread_cond_init(&newdataCond, NULL);
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;

	*this = o;
}

template <class T,class ALLOC>
tw_safeFIFO<T,ALLOC>::tw_safeFIFO( ALLOC *a ) : enabled( true ) {
	alloc = a;
	_block_cnt = 0;
//	dataMutex = PTHREAD_MUTEX_INITIALIZER;
//	newdataCond = PTHREAD_COND_INITIALIZER;
	pthread_mutex_init( &dataMutex, NULL );
	pthread_cond_init( &newdataCond, NULL );
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
}

template <class T,class ALLOC>
inline void tw_safeFIFO<T,ALLOC>::startIter( iter &i ) {
	i.look = out;
	pthread_mutex_lock(&dataMutex);
}

template <class T,class ALLOC>
inline void tw_safeFIFO<T,ALLOC>::releaseIter( iter &i ) {
	i.look = NULL;
	pthread_mutex_unlock(&dataMutex);
}

// copies other tw_safeFIFO - does not copy Allocator
template <class T,class ALLOC>
tw_safeFIFO<T,ALLOC> &tw_safeFIFO<T,ALLOC>::operator=( const tw_safeFIFO<T,ALLOC> &o ) {
	tw_FIFO_link *look;
	tw_FIFO_link *prev;
	tw_FIFO_link *newlink;

	this->clearAll(); // clear anything that might be there

	pthread_mutex_lock(const_cast<pthread_mutex_t *>(&(o.dataMutex)));
	pthread_mutex_lock(&dataMutex);
	this->enabled = o.enabled;
	this->remain = 0;
	/*
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	if(alloc)
		newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
	else
		newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);
*/
	look = o.out;

	if(look) {
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
	newlink->init_link(look->d);
	prev = newlink;
	this->out = newlink;
	look = look->next;
	this->remain++;
	}

	while(look) {
		newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
		newlink->init_link(look->d);
		prev->next = newlink; // link to link behind us
		prev = newlink;       // move forward
		look = look->next;    // move the source forward
		this->remain++;
	}

	this->in = prev;
	pthread_mutex_unlock(const_cast<pthread_mutex_t *>(&(o.dataMutex)));
	pthread_mutex_unlock(&dataMutex);
	return *this;
}

/**
 * enables the FIFO, allowing the adding of new items.
 * Items already in the FIFO can be pulled out irregardless.
 */
template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::enable() {
	pthread_mutex_lock(&dataMutex);
	enabled = true;
	pthread_mutex_unlock(&dataMutex);
}

/**
 * disables the FIFO, preventing the adding of new items.
 * Items already in the FIFO can be pulled out.
 */
template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::disable() {
	pthread_mutex_lock(&dataMutex);
	enabled = false;
	pthread_mutex_unlock(&dataMutex);
}

template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::add( T &the_d ) {
	tw_FIFO_link *newlink;
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	pthread_mutex_lock(&dataMutex);
	if(enabled) {
	if(in)
		in->next = newlink;
	in = newlink;
	if(!out) {
		out = in;
	}
	remain++;
#ifdef _TW_FIFO_DEBUG_ON
	TW_DEBUG_LT("tw_safeFifo:add - blk_cnt: %d\n",_block_cnt);
#endif
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif

	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
}
#if __cplusplus >= 201103L
template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::add( T &&the_d ) {
	tw_FIFO_link *newlink;
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	pthread_mutex_lock(&dataMutex);
	if(enabled) {
	if(in)
		in->next = newlink;
	in = newlink;
	if(!out) {
		out = in;
	}
	remain++;
#ifdef _TW_FIFO_DEBUG_ON
	TW_DEBUG_LT("tw_safeFifo:add - blk_cnt: %d\n",_block_cnt);
#endif
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif

	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
}
#endif

template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::addToHead( T &the_d ) {
	tw_FIFO_link *newlink;
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	pthread_mutex_lock(&dataMutex);
	if(enabled) {
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);

//	if(in)
//		in->next = newlink;
//	in = newlink;
	newlink->d = the_d;
	if(!out) {
		out = newlink;
	} else {
		newlink->next = out;
		out = newlink;
	}
	if(!in)
		in = out;
	remain++;
#ifdef _TW_FIFO_DEBUG_ON
	TW_DEBUG_LT("tw_safeFifo:add - blk_cnt: %d\n",_block_cnt);
#endif

	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif

	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
}

template <class T,class ALLOC>
T *tw_safeFIFO<T,ALLOC>::addEmpty() {
	tw_FIFO_link *newlink;
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	pthread_mutex_lock(&dataMutex);
	if(enabled) {
		newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
		newlink->init_link();
	if(in)
		in->next = newlink;
	in = newlink;
	if(!out) {
		out = in;
	}
	remain++;
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif
	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
	return &(newlink->d);
}


template <class T,class ALLOC>
bool tw_safeFIFO<T,ALLOC>::remove( T &fill ) {
//	T ret = NULL; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	bool ret = false;
	pthread_mutex_lock(&dataMutex);
	if(in==out)
		in = NULL;
	if(out) {
		ret = true;
		fill = out->d;
		tw_FIFO_link *oldout = out;
		out = out->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		ALLOC::free( oldout );
#endif
		remain--;
	}
	//	if(!out)
	//	out = in;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

#ifdef TWLIB_HAS_MOVE_SEMANTICS
template <class T,class ALLOC>
bool tw_safeFIFO<T,ALLOC>::remove_mv( T &fill ) {
//	T ret = NULL; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	bool ret = false;
	pthread_mutex_lock(&dataMutex);
	if(in==out)
		in = NULL;
	if(out) {
		ret = true;
		fill = std::move(out->d);
		tw_FIFO_link *oldout = out;
		out = out->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		ALLOC::free( oldout );
#endif
		remain--;
	}
	//	if(!out)
	//	out = in;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}
#endif

template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::unblock( void ) { // unblock any waiting blocking calls
	pthread_mutex_lock(&dataMutex);
	// NOTE: linux seems to want to block on pthread_cond_signal. according to POSIX this should never happen
	//       but the implementation seems to think otherwise...
	//       This is why _block_cnt exists - we only .._signal() when at least 1 or more calls are blocking
	if(_block_cnt > 0) {
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_signal\n",NULL);
#endif
	pthread_cond_signal(&newdataCond); // let one (and only one) blocking call know to wakeup
	_block_cnt--;
	}
	pthread_mutex_unlock(&dataMutex);
}

template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::unblockAll( void ) { // unblock any waiting blocking calls
	pthread_mutex_lock(&dataMutex);
	if(_block_cnt > 0) {	// see unblock() about this var (above)
	pthread_cond_broadcast(&newdataCond); // let all blocking calls know to wakeup
	_block_cnt = 0;
	}
	pthread_mutex_unlock(&dataMutex);
}


template <class T,class ALLOC>
bool tw_safeFIFO<T,ALLOC>::removeOrBlock( T &fill ) {
	bool ret = false; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	pthread_mutex_lock(&dataMutex);
	if(in==out) {
		in = NULL;
	}
	if(!out) { // if no data... wait
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_wait\n",NULL);
#endif
		_block_cnt++; // see unblock() for details
		pthread_cond_wait( &newdataCond, &dataMutex ); // wait until new data arrives
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread out\n",NULL);
#endif
	}
	if(in==out) { // this check must be done again
		in = NULL;
	}
	if(out) {
		ret = true;
		fill = out->d;
		tw_FIFO_link *oldout = out;
		out = out->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		ALLOC::free( oldout );
#endif
		remain--;
	}
	//	if(!out)
	//	out = in;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_safeFIFO<T,ALLOC>::removeOrBlock( T &fill, TimeVal &t ) {
	bool ret = false; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	pthread_mutex_lock(&dataMutex);
	if(in==out) {
		in = NULL;
	}
	if(!out) { // if no data... wait
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_wait\n",NULL);
#endif
		_block_cnt++; // see unblock() for details
		int err = pthread_cond_timedwait( &newdataCond, &dataMutex, t.timespec() ); // wait until new data arrives
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread out\n",NULL);
#endif
		if(err==ETIMEDOUT) {
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_timedwait - ETIMEDOUT\n",NULL);
#endif
			pthread_mutex_unlock(&dataMutex);
			return false;
		}
	}
	if(in==out) { // this check must be done again
		in = NULL;
	}
	if(out) {
		ret = true;
		fill = out->d;
		tw_FIFO_link *oldout = out;
		out = out->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		ALLOC::free( oldout );
#endif
		remain--;
	}
	//	if(!out)
	//	out = in;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_safeFIFO<T,ALLOC>::peek( T &fill ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
	pthread_mutex_lock(&dataMutex);
	if(out) {
		ret = true;
		fill = out->d;
	}
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_safeFIFO<T,ALLOC>::peekOrBlock( T &fill ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
	pthread_mutex_lock(&dataMutex);
	if(!out) {
		_block_cnt++;
		pthread_cond_wait( &newdataCond, &dataMutex ); // wait until new data arrives
	}
	if(out) {
		ret = true;
		fill = out->d;
	}
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_safeFIFO<T,ALLOC>::peekOrBlock( T &fill, TimeVal &t ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
	pthread_mutex_lock(&dataMutex);
	if(!out) {
		_block_cnt++;
		int err = pthread_cond_timedwait( &newdataCond, &dataMutex, t.timespec() ); // wait until new data arrives
		if(err == ETIMEDOUT) { // if timeout, don't look at value.
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_timedwait - peek ETIMEDOUT\n",NULL);
#endif
			pthread_mutex_unlock(&dataMutex);
			return false;
		}
	}
	if(out) {
		ret = true;
		fill = out->d;
	}
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
int tw_safeFIFO<T,ALLOC>::remaining(void) {
	int ret;
	pthread_mutex_lock(&dataMutex);
	ret = remain;
	pthread_mutex_unlock(&dataMutex);
	return remain;

}

template <class T,class ALLOC>
void tw_safeFIFO<T,ALLOC>::clearAll() { // delete all remaining links (and hope someone took care of the data in each of those)
	int ret;
	tw_FIFO_link *n = NULL;
	pthread_mutex_lock(&dataMutex);
	while(out) {
		n = out->next;
		ALLOC::free(out);
		out = n;
	}

	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
	pthread_mutex_unlock(&dataMutex);
	unblockAll();
}


template <class T,class ALLOC>
tw_safeFIFO<T,ALLOC>::~tw_safeFIFO() { // delete all remaining links (and hope someone took care of the data in each of those)
	tw_FIFO_link *n = NULL;
	unblockAll();
	pthread_mutex_lock(&dataMutex);
	while(out) {
		n = out->next;
		ALLOC::free(out);
		out = n;
	}
	pthread_cond_destroy(&newdataCond);   // NEW
	pthread_mutex_unlock(&dataMutex);
	pthread_mutex_destroy(&dataMutex);    // NEW
	// Should call pthread_cond_destroy
}

/////////////////////////////////////////////////////////////////////////
// thread safe FIFO mv (rvalue assignments)
#ifdef TWLIB_HAS_MOVE_SEMANTICS

template <class T,class ALLOC>
tw_safeFIFOmv<T,ALLOC>::tw_FIFO_link::tw_FIFO_link(void) {
//	d = (T *) NULL;
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::tw_FIFO_link::init_link( T &the_d ) {
	::new((void*)&d) T(); // must explicity call the constructor
	d = std::move(the_d);
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::tw_FIFO_link::init_link( ) {
	::new((void*)&d) T(); // must explicity call the constructor
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}


#ifdef _TW_WINDOWS
template <class T,class ALLOC>
tw_safeFIFOmv<T,ALLOC>::tw_safeFIFOmv( HANDLE theHeap ) : enabled( true ) {
	alloc = NULL;
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
	hHeap = theHeap;
}
#endif

template <class T,class ALLOC>
tw_safeFIFOmv<T,ALLOC>::tw_safeFIFOmv( void ) : enabled( true ) {
	alloc = NULL;
	_block_cnt = 0;
	pthread_mutex_init( &dataMutex, NULL );
	pthread_cond_init( &newdataCond, NULL );
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
}


template<class T, class ALLOC>
tw_safeFIFOmv<T,ALLOC>::tw_safeFIFOmv(tw_safeFIFOmv<T,ALLOC> &o) : enabled( true ) {
	alloc = NULL;
	_block_cnt = 0;
	pthread_mutex_init(&dataMutex, NULL);
	pthread_cond_init(&newdataCond, NULL);
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;

	*this = o;
}

template <class T,class ALLOC>
tw_safeFIFOmv<T,ALLOC>::tw_safeFIFOmv( ALLOC *a ) : enabled( true ) {
	alloc = a;
	_block_cnt = 0;
//	dataMutex = PTHREAD_MUTEX_INITIALIZER;
//	newdataCond = PTHREAD_COND_INITIALIZER;
	pthread_mutex_init( &dataMutex, NULL );
	pthread_cond_init( &newdataCond, NULL );
	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
}

template <class T,class ALLOC>
inline void tw_safeFIFOmv<T,ALLOC>::startIter( iter &i ) {
	i.look = out;
	pthread_mutex_lock(&dataMutex);
}

template <class T,class ALLOC>
inline void tw_safeFIFOmv<T,ALLOC>::releaseIter( iter &i ) {
	i.look = NULL;
	pthread_mutex_unlock(&dataMutex);
}

// copies other tw_safeFIFOmv - does not copy Allocator
template <class T,class ALLOC>
tw_safeFIFOmv<T,ALLOC> &tw_safeFIFOmv<T,ALLOC>::operator=( const tw_safeFIFOmv<T,ALLOC> &o ) {
	tw_FIFO_link *look;
	tw_FIFO_link *prev;
	tw_FIFO_link *newlink;

	this->clearAll(); // clear anything that might be there

	pthread_mutex_lock(const_cast<pthread_mutex_t *>(&(o.dataMutex)));
	pthread_mutex_lock(&dataMutex);
	this->enabled = o.enabled;
	this->remain = 0;
	/*
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	if(alloc)
		newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
	else
		newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);
*/
	look = o.out;

	if(look) {
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
	newlink->init_link(look->d);
	prev = newlink;
	this->out = newlink;
	look = look->next;
	this->remain++;
	}

	while(look) {
		newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
		newlink->init_link(look->d);
		prev->next = newlink; // link to link behind us
		prev = newlink;       // move forward
		look = look->next;    // move the source forward
		this->remain++;
	}

	this->in = prev;
	pthread_mutex_unlock(const_cast<pthread_mutex_t *>(&(o.dataMutex)));
	pthread_mutex_unlock(&dataMutex);
	return *this;
}

/**
 * enables the FIFO, allowing the adding of new items.
 * Items already in the FIFO can be pulled out irregardless.
 */
template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::enable() {
	pthread_mutex_lock(&dataMutex);
	enabled = true;
	pthread_mutex_unlock(&dataMutex);
}

/**
 * disables the FIFO, preventing the adding of new items.
 * Items already in the FIFO can be pulled out.
 */
template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::disable() {
	pthread_mutex_lock(&dataMutex);
	enabled = false;
	pthread_mutex_unlock(&dataMutex);
}

template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::add( T &the_d ) {
	tw_FIFO_link *newlink;
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	pthread_mutex_lock(&dataMutex);
	if(enabled) {
	if(in)
		in->next = newlink;
	in = newlink;
	if(!out) {
		out = in;
	}
	remain++;
#ifdef _TW_FIFO_DEBUG_ON
	TW_DEBUG_LT("tw_safeFIFOmv:add - blk_cnt: %d\n",_block_cnt);
#endif
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFOmv: not enabled\n");
#endif

	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
}
#if __cplusplus >= 201103L
template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::add( T &&the_d ) {
	tw_FIFO_link *newlink;
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	pthread_mutex_lock(&dataMutex);
	if(enabled) {
	if(in)
		in->next = newlink;
	in = newlink;
	if(!out) {
		out = in;
	}
	remain++;
#ifdef _TW_FIFO_DEBUG_ON
	TW_DEBUG_LT("tw_safeFIFOmv:add - blk_cnt: %d\n",_block_cnt);
#endif
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFOmv: not enabled\n");
#endif

	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
}
#endif

template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::addToHead( T &the_d ) {
	tw_FIFO_link *newlink;
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	pthread_mutex_lock(&dataMutex);
	if(enabled) {
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);

//	if(in)
//		in->next = newlink;
//	in = newlink;
	newlink->d = the_d;
	if(!out) {
		out = newlink;
	} else {
		newlink->next = out;
		out = newlink;
	}
	if(!in)
		in = out;
	remain++;
#ifdef _TW_FIFO_DEBUG_ON
	TW_DEBUG_LT("tw_safeFIFOmv:add - blk_cnt: %d\n",_block_cnt);
#endif

	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFOmv: not enabled\n");
#endif

	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
}

template <class T,class ALLOC>
T *tw_safeFIFOmv<T,ALLOC>::addEmpty() {
	tw_FIFO_link *newlink;
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	pthread_mutex_lock(&dataMutex);
	if(enabled) {
		newlink = (tw_FIFO_link *) ALLOC::malloc( sizeof( tw_FIFO_link ));
		newlink->init_link();
	if(in)
		in->next = newlink;
	in = newlink;
	if(!out) {
		out = in;
	}
	remain++;
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFOmv: not enabled\n");
#endif
	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
	return &(newlink->d);
}


template <class T,class ALLOC>
bool tw_safeFIFOmv<T,ALLOC>::remove( T &fill ) {
//	T ret = NULL; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	bool ret = false;
	pthread_mutex_lock(&dataMutex);
	if(in==out)
		in = NULL;
	if(out) {
		ret = true;
		fill = std::move(out->d);
		tw_FIFO_link *oldout = out;
		out = out->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		ALLOC::free( oldout );
#endif
		remain--;
	}
	//	if(!out)
	//	out = in;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::unblock( void ) { // unblock any waiting blocking calls
	pthread_mutex_lock(&dataMutex);
	// NOTE: linux seems to want to block on pthread_cond_signal. according to POSIX this should never happen
	//       but the implementation seems to think otherwise...
	//       This is why _block_cnt exists - we only .._signal() when at least 1 or more calls are blocking
	if(_block_cnt > 0) {
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_signal\n",NULL);
#endif
	pthread_cond_signal(&newdataCond); // let one (and only one) blocking call know to wakeup
	_block_cnt--;
	}
	pthread_mutex_unlock(&dataMutex);
}

template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::unblockAll( void ) { // unblock any waiting blocking calls
	pthread_mutex_lock(&dataMutex);
	if(_block_cnt > 0) {	// see unblock() about this var (above)
	pthread_cond_broadcast(&newdataCond); // let all blocking calls know to wakeup
	_block_cnt = 0;
	}
	pthread_mutex_unlock(&dataMutex);
}


template <class T,class ALLOC>
bool tw_safeFIFOmv<T,ALLOC>::removeOrBlock( T &fill ) {
	bool ret = false; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	pthread_mutex_lock(&dataMutex);
	if(in==out) {
		in = NULL;
	}
	if(!out) { // if no data... wait
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_wait\n",NULL);
#endif
		_block_cnt++; // see unblock() for details
		pthread_cond_wait( &newdataCond, &dataMutex ); // wait until new data arrives
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread out\n",NULL);
#endif
	}
	if(in==out) { // this check must be done again
		in = NULL;
	}
	if(out) {
		ret = true;
		fill = std::move(out->d);
		tw_FIFO_link *oldout = out;
		out = out->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		ALLOC::free( oldout );
#endif
		remain--;
	}
	//	if(!out)
	//	out = in;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_safeFIFOmv<T,ALLOC>::removeOrBlock( T &fill, TimeVal &t ) {
	bool ret = false; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	pthread_mutex_lock(&dataMutex);
	if(in==out) {
		in = NULL;
	}
	if(!out) { // if no data... wait
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_wait\n",NULL);
#endif
		_block_cnt++; // see unblock() for details
		int err = pthread_cond_timedwait( &newdataCond, &dataMutex, t.timespec() ); // wait until new data arrives
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread out\n",NULL);
#endif
		if(err==ETIMEDOUT) {
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_timedwait - ETIMEDOUT\n",NULL);
#endif
			pthread_mutex_unlock(&dataMutex);
			return false;
		}
	}
	if(in==out) { // this check must be done again
		in = NULL;
	}
	if(out) {
		ret = true;
		fill = out->d;
		tw_FIFO_link *oldout = out;
		out = out->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		ALLOC::free( oldout );
#endif
		remain--;
	}
	//	if(!out)
	//	out = in;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_safeFIFOmv<T,ALLOC>::peek( T &fill ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
	pthread_mutex_lock(&dataMutex);
	if(out) {
		ret = true;
		fill = out->d;
	}
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_safeFIFOmv<T,ALLOC>::peekOrBlock( T &fill ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
	pthread_mutex_lock(&dataMutex);
	if(!out) {
		_block_cnt++;
		pthread_cond_wait( &newdataCond, &dataMutex ); // wait until new data arrives
	}
	if(out) {
		ret = true;
		fill = out->d;
	}
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
bool tw_safeFIFOmv<T,ALLOC>::peekOrBlock( T &fill, TimeVal &t ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
	pthread_mutex_lock(&dataMutex);
	if(!out) {
		_block_cnt++;
		int err = pthread_cond_timedwait( &newdataCond, &dataMutex, t.timespec() ); // wait until new data arrives
		if(err == ETIMEDOUT) { // if timeout, don't look at value.
#ifdef _TW_FIFO_DEBUG_ON
		TW_DEBUG_LT("pthread_cond_timedwait - peek ETIMEDOUT\n",NULL);
#endif
			pthread_mutex_unlock(&dataMutex);
			return false;
		}
	}
	if(out) {
		ret = true;
		fill = out->d;
	}
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T,class ALLOC>
int tw_safeFIFOmv<T,ALLOC>::remaining(void) {
	int ret;
	pthread_mutex_lock(&dataMutex);
	ret = remain;
	pthread_mutex_unlock(&dataMutex);
	return remain;

}

template <class T,class ALLOC>
void tw_safeFIFOmv<T,ALLOC>::clearAll() { // delete all remaining links (and hope someone took care of the data in each of those)
	int ret;
	tw_FIFO_link *n = NULL;
	pthread_mutex_lock(&dataMutex);
	while(out) {
		n = out->next;
		ALLOC::free(out);
		out = n;
	}

	out = (tw_FIFO_link *) NULL;
	in = (tw_FIFO_link *) NULL;
	remain = 0;
	pthread_mutex_unlock(&dataMutex);
	unblockAll();
}


template <class T,class ALLOC>
tw_safeFIFOmv<T,ALLOC>::~tw_safeFIFOmv() { // delete all remaining links (and hope someone took care of the data in each of those)
	tw_FIFO_link *n = NULL;
	unblockAll();
	pthread_mutex_lock(&dataMutex);
	while(out) {
		n = out->next;
		ALLOC::free(out);
		out = n;
	}
	pthread_cond_destroy(&newdataCond);   // NEW
	pthread_mutex_unlock(&dataMutex);
	pthread_mutex_destroy(&dataMutex);    // NEW
	// Should call pthread_cond_destroy
}

#endif // TWLIB_HAS_MOVE_SEMANTICS

/////////////////////////////////////////////////////
//// TW_bndSafeFIFO
/////////////////////////////////////////////////////



template <class T,class ALLOC>
tw_bndSafeFIFO<T,ALLOC>::tw_bndSafeFIFO( void ) : _fifo(), _sizeSema( NULL ), _max( 0 ) {
}

template <class T,class ALLOC>
tw_bndSafeFIFO<T,ALLOC>::tw_bndSafeFIFO( int maxsize ) : _fifo(),  _sizeSema( NULL ), _max( maxsize ) {
	_sizeSema = new TW_Sema( maxsize );
}

template<class T, class ALLOC>
tw_bndSafeFIFO<T,ALLOC>::tw_bndSafeFIFO(tw_bndSafeFIFO<T,ALLOC> &o) : _fifo( o._fifo ), _sizeSema( NULL ), _max( o._max ) {
	_sizeSema = new TW_Sema( o._sizeSema->count() );
}

template<class T, class ALLOC>
tw_bndSafeFIFO<T,ALLOC>::tw_bndSafeFIFO(int maxsize, tw_safeFIFO<T,ALLOC> &o) : _fifo( o ), _sizeSema( NULL ), _max( maxsize ) {
#ifdef _TW_FIFO_DEBUG_ON
	if(maxsize < o.remaining() )
		TW_DEBUG_LT("Unsafe constructor call - tw_bndSafeFIFO\n",NULL);
#endif
	_sizeSema = new TW_Sema( maxsize - o.remaining() );
}

template <class T,class ALLOC>
tw_bndSafeFIFO<T,ALLOC>::tw_bndSafeFIFO( int maxsize, ALLOC *a ) : _fifo( a ),  _sizeSema( NULL ), _max( maxsize ) {
	_sizeSema = new TW_Sema( maxsize );
}

template <class T,class ALLOC>
inline void tw_bndSafeFIFO<T,ALLOC>::startIter( iter &i ) {
	_fifo.startIter(i);
}

template <class T,class ALLOC>
inline void tw_bndSafeFIFO<T,ALLOC>::releaseIter( iter &i ) {
	_fifo.releaseIter(i);
}

// copies other tw_bndSafeFIFO - does not copy Allocator
template <class T,class ALLOC>
tw_bndSafeFIFO<T,ALLOC> &tw_bndSafeFIFO<T,ALLOC>::operator=( const tw_bndSafeFIFO<T,ALLOC> &o ) {
	_fifo = o._fifo; // call operator=() for tw_safeFIFO

	if(_sizeSema) {
		_sizeSema->releaseAll();
		delete _sizeSema;
	}
	_sizeSema = new TW_Sema( o._sizeSema->count() );
	return *this;
}

template <class T,class ALLOC>
inline void tw_bndSafeFIFO<T,ALLOC>::disable() {
	_fifo.disable();
}

template <class T,class ALLOC>
inline void tw_bndSafeFIFO<T,ALLOC>::enable() {
	_fifo.enable();
}

template <class T,class ALLOC>
void tw_bndSafeFIFO<T,ALLOC>::add( T &the_d ) {
	_sizeSema->acquire(); // may block (if we get the acquire, then we should be for at least one add
	                      //            and only 1 thread - this thread - gets this acquire)
	_fifo.add(the_d);
}

template <class T,class ALLOC>
void tw_bndSafeFIFO<T,ALLOC>::addToHead( T &the_d ) {
	_sizeSema->acquire(); // may block (if we get the acquire, then we should be for at least one add
	                      //            and only 1 thread - this thread - gets this acquire)
	_fifo.addToHead(the_d);
}


template <class T,class ALLOC>
T *tw_bndSafeFIFO<T,ALLOC>::addEmpty() {
	_sizeSema->acquire(); // may block
	return _fifo.addEmpty();
}

template <class T,class ALLOC>
bool tw_bndSafeFIFO<T,ALLOC>::remove( T &fill ) {
	bool ret = _fifo.remove(fill);
	if(ret) // only decrease if something was actually removed
		_sizeSema->release();
	return ret;
}

template <class T,class ALLOC>
void tw_bndSafeFIFO<T,ALLOC>::unblockRemoveCalls( void ) { // unblock any waiting blocking calls
	_fifo.unblock();
}

template <class T,class ALLOC>
void tw_bndSafeFIFO<T,ALLOC>::unblockAll( void ) { // unblock any waiting blocking calls
	_sizeSema->releaseAll();
	_fifo.unblockAll();
}

template <class T,class ALLOC>
bool tw_bndSafeFIFO<T,ALLOC>::removeOrBlock( T &fill ) {
	bool ret = _fifo.removeOrBlock(fill);
	if(ret) // only decrease if something was actually removed
		_sizeSema->release();
	return ret;
}


template <class T,class ALLOC>
bool tw_bndSafeFIFO<T,ALLOC>::peek( T &fill ) {
	return _fifo.peek();
}

template <class T,class ALLOC>
bool tw_bndSafeFIFO<T,ALLOC>::peekOrBlock( T &fill ) {
	return _fifo.peekOrBlock();
}

template <class T,class ALLOC>
int tw_bndSafeFIFO<T,ALLOC>::remaining(void) {
	return _fifo.remaining();
}

template <class T,class ALLOC>
void tw_bndSafeFIFO<T,ALLOC>::clearAll() { // delete all remaining links (and hope someone took care of the data in each of those)
	_fifo.clearAll();

	if(_sizeSema) {
		_sizeSema->releaseAll();
		delete _sizeSema;
	}
	_sizeSema = new TW_Sema( _max );
}

template <class T,class ALLOC>
tw_bndSafeFIFO<T,ALLOC>::~tw_bndSafeFIFO() { // delete all remaining links (and hope someone took care of the data in each of those)
	if(_sizeSema) {
		_sizeSema->releaseAll();
		delete _sizeSema;
	}
	// internal FIFO will call destructor for it on teardown...
}




#endif // _TW_FIFO
