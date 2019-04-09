
// tupperware container lib

// FIFO: a simple class to handle a fifo list of void pointers.
#include <pthread.h>
#include <ace/Malloc_Base.h>
#include <ace/Message_Block.h>
#include <ace/OS_Memory.h>

#include "logging.h"

#ifndef _TW_LLIST
#define _TW_LLIST

namespace TWlib {



/** T must support:
 * self assignment: operator= (T &a, T &b)
 * default constructor
 */
template <class T>
class tw_safeFIFO {
public:
	struct tw_FIFO_link {
		T d;
		tw_FIFO_link *next;
		tw_FIFO_link *prev;
		tw_FIFO_link();
		void init_link( T &the_d );
		void init_link();
	};

	class iter {
	public:
		iter() : look(NULL) { }
		bool getNext(T &fill);
		bool getPrev(T &fill);
		bool removeCurrent();
		bool getCurrent(T &fill);
		bool atEnd();
		friend class tw_safeFIFO;
	protected:
		tw_FIFO_link *look;
	};

	tw_safeFIFO( void );
	tw_safeFIFO( ACE_Allocator *_a );
#ifdef _TW_WINDOWS
	tw_safeFIFO( HANDLE theHeap );
#endif
	void add( T &d ); // add to tail
	void addToFront( T &d );
	T *addEmpty();
//	bool peek( T &fill ); // true if got valid value - look at next, dont remove
//	bool peekOrBlock( T &fill ); // true if got data - look at next, dont remove
	bool removeTail( T &fill ); // true if got data
//	bool removeOrBlock( T &fill ); // true if removed something
	void clearAll(); // remove all nodes (does not delete T)
	void unblock();  // unblock 1 blocking call
	void unblockAll(); // unblock all blocking calls
	void startIter( iter &i );
	tw_safeFIFO &operator=( const tw_safeFIFO &o );

//	void removeAtIter( iter &i );
	void releaseIter( iter &i );
	int remaining();
	~tw_safeFIFO();
private:
	pthread_mutex_t dataMutex; // thread safety for FIFO
	pthread_cond_t newdataCond;
	int _block_cnt;
	int remain;
	tw_FIFO_link *tail; // add from this end (tail)
	tw_FIFO_link *head; // remove from this end (head)
	ACE_Allocator *alloc;
#ifdef _TW_WINDOWS
	HANDLE hHeap;
#endif
};


/** T must support:
 * self assignment: operator= (T &a, T &b)
 * default constructor
 */
template <class T>
class tw_FIFO {
public:
	struct tw_FIFO_link {
		T d;
		tw_FIFO_link *next;
		tw_FIFO_link *prev;
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
	tw_FIFO( ACE_Allocator *_a );
#ifdef _TW_WINDOWS
	tw_FIFO( HANDLE theHeap );
#endif
	void add( T &d );
	void addToFront( T &d );
	T *addEmpty();
//	bool peek( T &fill ); // look at next, dont remove
	bool removeTail( T &fill );
	int remaining();
	void startIter( iter &i );
	tw_FIFO &operator=( const tw_FIFO &o );
	void clearAll(); // remove all nodes (does not delete T)
	~tw_FIFO();
private:
	tw_FIFO_link *tail; // add from this end
	tw_FIFO_link *head; // remove from this end
	int remain;
	ACE_Allocator *alloc;
#ifdef _TW_WINDOWS
	HANDLE hHeap;
#endif
//	pthread_mutex_t dataMutex; // thread safety for FIFO
};

}

using namespace TWlib;

template <class T>
bool tw_safeFIFO<T>::iter::getNext(T &fill) {
	if(look) {
		fill = look->d;
		look = look->next;
		return true;
	} else
		return false;
}

template <class T>
inline bool tw_safeFIFO<T>::iter::atEnd() {
	if(look)
		return false;
	else
		return true;
}

template <class T>
bool tw_FIFO<T>::iter::getNext(T &fill) {
	if(look) {
		fill = look->d;
		look = look->next;
		return true;
	} else
		return false;
}

template <class T>
bool tw_FIFO<T>::iter::getPrev(T &fill) {
	if(look) {
		fill = look->d;
		look = look->prev;
		return true;
	} else
		return false;
}


template <class T>
inline bool tw_FIFO<T>::iter::atEnd() {
	if(look)
		return false;
	else
		return true;
}



template <class T>
tw_FIFO<T>::tw_FIFO_link::tw_FIFO_link(void) {
//	d = (T *) NULL;
//	d = NULL;
	next = (tw_FIFO_link *) NULL;
	prev = (tw_FIFO_link *) NULL;
}

template <class T>
void tw_FIFO<T>::tw_FIFO_link::init_link( T &the_d ) {
	::new((void*)&d) T(); // must explicity call the constructor
	d = the_d;
	next = (tw_FIFO_link *) NULL;
	prev = (tw_FIFO_link *) NULL;
}

template <class T>
void tw_FIFO<T>::tw_FIFO_link::init_link() {
	::new((void*)&d) T(); // must explicity call the constructor
	next = (tw_FIFO_link *) NULL;
	prev = (tw_FIFO_link *) NULL;
}

#ifdef _TW_WINDOWS
template <class T>
tw_FIFO<T>::tw_FIFO( HANDLE theHeap ) {
	alloc = NULL;
	head = (tw_FIFO_link *) NULL;
	tail = (tw_FIFO_link *) NULL;
	remain = 0;
	hHeap = theHeap;
}
#endif

template <class T>
tw_FIFO<T>::tw_FIFO( void ) {
	alloc = NULL;
//	pthread_mutex_init( &dataMutex, NULL );
	head = (tw_FIFO_link *) NULL;
	tail = (tw_FIFO_link *) NULL;
	remain = 0;
#ifdef _TW_WINDOWS
	// todo: make hHeap the default process Heap
#endif
}

template <class T>
tw_FIFO<T>::tw_FIFO( ACE_Allocator *a ) {
	alloc = a;
//	pthread_mutex_init( &dataMutex, NULL );
	head = (tw_FIFO_link *) NULL;
	tail = (tw_FIFO_link *) NULL;
	remain = 0;
#ifdef _TW_WINDOWS
	// todo: make hHeap the default process Heap
#endif
}

template <class T>
inline void tw_FIFO<T>::startIter( iter &i ) {
	i.look = head;
}

template <class T>
inline void tw_FIFO<T>::startIterTail( iter &i ) {
	i.look = tail;
}



template <class T>
void tw_FIFO<T>::add( T &the_d ) {
	tw_FIFO_link *newlink;
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	if(alloc)
		newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
	else
		newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = tail;
//	pthread_mutex_lock(&dataMutex);
	if(tail)
		tail->next = newlink;
	tail = newlink;
	if(!head) {
		head = tail;
	}
	remain++;
//	pthread_mutex_unlock(&dataMutex);
}

template <class T>
T *tw_FIFO<T>::addEmpty() {
	tw_FIFO_link *newlink;
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	if(alloc)
		newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
	else
		newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link();
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = tail;
//	pthread_mutex_lock(&dataMutex);
	if(tail)
		tail->next = newlink;
	tail = newlink;
	if(!head) {
		head = tail;
	}
	remain++;
//	pthread_mutex_unlock(&dataMutex);
	return &(newlink->d);
}



template <class T>
bool tw_FIFO<T>::remove( T &fill ) {
	bool ret = false;
//	pthread_mutex_lock(&dataMutex);
	if(tail==head)
		tail = NULL;
	if(head) {
		ret = true;
		fill = head->d;
		tw_FIFO_link *oldout = head;
		head = head->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		if(alloc)
			alloc->free( oldout );
		else
			ACE_OS::free( oldout );
#endif
		remain--;
	}
	//	if(!head)
	//	head = tail;
//	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T>
bool tw_FIFO<T>::peek( T &fill ) {
	T ret = false;
	//	if(tail==head)
	//	tail = NULL;
//	pthread_mutex_lock(&dataMutex);
	if(head) {
		ret = true;
		fill = head->d;
	}
//	pthread_mutex_unlock(&dataMutex);
	return ret;
}



template <class T>
int tw_FIFO<T>::remaining(void) {
	int ret;
//	pthread_mutex_lock(&dataMutex);
	ret = remain;
//	pthread_mutex_unlock(&dataMutex);
	return remain;

}

// copies other tw_safeFIFO - does not copy Allocator
template <class T>
tw_FIFO<T> &tw_FIFO<T>::operator=( const tw_FIFO<T> &o ) {
	tw_FIFO_link *look;
	tw_FIFO_link *prev;
	tw_FIFO_link *newlink;

	this->clearAll(); // clear anything that might be there


	this->remain = 0;
	look = o.head;

	if(look) {
	if(alloc)
		newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
	else
		newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
	newlink->init_link(look->d);
	prev = newlink;
	this->head = newlink;
	look = look->next;
	this->remain++;
	}

	while(look) {
		if(alloc)
			newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
		else
			newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
		newlink->init_link(look->d);
		prev->next = newlink; // link to link behind us
		prev = newlink;       // move forward
		look = look->next;    // move the source forward
		this->remain++;
	}

	this->tail = prev;
	return *this;
}

template <class T>
void tw_FIFO<T>::clearAll() { // delete all remaining links (and hope someone took care of the data tail each of those)
	int ret;
	tw_FIFO_link *n = NULL;
//	pthread_mutex_lock(&dataMutex);
	while(head) {
		n = head->next;
		if(alloc)
			alloc->free(head);
		else
			ACE_OS::free(head);
		head = n;
	}

	head = (tw_FIFO_link *) NULL;
	tail = (tw_FIFO_link *) NULL;
	remain = 0;
//	pthread_mutex_unlock(&dataMutex);
}


template <class T>
tw_FIFO<T>::~tw_FIFO() { // delete all remaining links (and hope someone took care of the data tail each of those)
//	int ret;
	tw_FIFO_link *n = NULL;
//	pthread_mutex_lock(&dataMutex);
	while(head) {
		n = head->next;
		if(alloc)
			alloc->free(head);
		else
			ACE_OS::free(head);
		head = n;
	}
//	pthread_mutex_unlock(&dataMutex);
}
/////////////////////////////////////////////////////////////////////////
// thread safe FIFO

template <class T>
tw_safeFIFO<T>::tw_FIFO_link::tw_FIFO_link(void) {
//	d = (T *) NULL;
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

template <class T>
void tw_safeFIFO<T>::tw_FIFO_link::init_link( T &the_d ) {
	::new((void*)&d) T(); // must explicity call the constructor
	d = the_d;
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}

template <class T>
void tw_safeFIFO<T>::tw_FIFO_link::init_link( ) {
	::new((void*)&d) T(); // must explicity call the constructor
	next = (tw_FIFO_link *) NULL;
//	prev = (tw_FIFO_link *) NULL;
}


#ifdef _TW_WINDOWS
template <class T>
tw_safeFIFO<T>::tw_safeFIFO( HANDLE theHeap ) {
	alloc = NULL;
	head = (tw_FIFO_link *) NULL;
	tail = (tw_FIFO_link *) NULL;
	remain = 0;
	hHeap = theHeap;
}
#endif

template <class T>
tw_safeFIFO<T>::tw_safeFIFO( void ) {
	alloc = NULL;
	_block_cnt = 0;
	pthread_mutex_init( &dataMutex, NULL );
	head = (tw_FIFO_link *) NULL;
	tail = (tw_FIFO_link *) NULL;
	remain = 0;
#ifdef _TW_WINDOWS
	// todo: make hHeap the default process Heap
#endif
}

template <class T>
tw_safeFIFO<T>::tw_safeFIFO( ACE_Allocator *a ) {
	alloc = a;
	_block_cnt = 0;
//	dataMutex = PTHREAD_MUTEX_INITIALIZER;
//	newdataCond = PTHREAD_COND_INITIALIZER;
	pthread_mutex_init( &dataMutex, NULL );
	pthread_cond_init( &newdataCond, NULL );
	head = (tw_FIFO_link *) NULL;
	tail = (tw_FIFO_link *) NULL;
	remain = 0;
#ifdef _TW_WINDOWS
	// todo: make hHeap the default process Heap
#endif
}

template <class T>
inline void tw_safeFIFO<T>::startIter( iter &i ) {
	i.look = head;
	pthread_mutex_lock(&dataMutex);
}

template <class T>
inline void tw_safeFIFO<T>::releaseIter( iter &i ) {
	i.look = NULL;
	pthread_mutex_unlock(&dataMutex);
}

// copies other tw_safeFIFO - does not copy Allocator
template <class T>
tw_safeFIFO<T> &tw_safeFIFO<T>::operator=( const tw_safeFIFO<T> &o ) {
	tw_FIFO_link *look;
	tw_FIFO_link *prev;
	tw_FIFO_link *newlink;

	this->clearAll(); // clear anything that might be there

	pthread_mutex_lock(const_cast<pthread_mutex_t *>(&(o.dataMutex)));
	pthread_mutex_lock(&dataMutex);

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
	look = o.head;

	if(look) {
	if(alloc)
		newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
	else
		newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
	newlink->init_link(look->d);
	prev = newlink;
	this->head = newlink;
	look = look->next;
	this->remain++;
	}

	while(look) {
		if(alloc)
			newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
		else
			newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
		newlink->init_link(look->d);
		prev->next = newlink; // link to link behind us
		prev = newlink;       // move forward
		look = look->next;    // move the source forward
		this->remain++;
	}

	this->tail = prev;
	pthread_mutex_unlock(const_cast<pthread_mutex_t *>(&(o.dataMutex)));
	pthread_mutex_unlock(&dataMutex);
	return *this;
}

template <class T>
void tw_safeFIFO<T>::add( T &the_d ) {
	tw_FIFO_link *newlink;
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	if(alloc)
		newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
	else
		newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = tail;
	pthread_mutex_lock(&dataMutex);
	if(tail)
		tail->next = newlink;
	tail = newlink;
	if(!head) {
		head = tail;
	}
	remain++;
	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
}

template <class T>
T *tw_safeFIFO<T>::addEmpty() {
	tw_FIFO_link *newlink;
#ifdef _TW_WINDOWS
	newlink = (tw_FIFO_link *) HeapAlloc( hHeap, 0, sizeof( tw_FIFO_link ));
#else
	if(alloc)
		newlink = (tw_FIFO_link *) alloc->malloc( sizeof( tw_FIFO_link ));
	else
		newlink = (tw_FIFO_link *) ACE_OS::malloc( sizeof( tw_FIFO_link ));
#endif
	newlink->init_link();
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = tail;
	pthread_mutex_lock(&dataMutex);
	if(tail)
		tail->next = newlink;
	tail = newlink;
	if(!head) {
		head = tail;
	}
	remain++;
	pthread_mutex_unlock(&dataMutex);
	unblock(); // let one blocking call know...
	return &(newlink->d);
}


template <class T>
bool tw_safeFIFO<T>::remove( T &fill ) {
//	T ret = NULL; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	bool ret = false;
	pthread_mutex_lock(&dataMutex);
	if(tail==head)
		tail = NULL;
	if(head) {
		ret = true;
		fill = head->d;
		tw_FIFO_link *oldout = head;
		head = head->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		if(alloc)
			alloc->free( oldout );
		else
			ACE_OS::free( oldout );
#endif
		remain--;
	}
	//	if(!head)
	//	head = tail;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T>
void tw_safeFIFO<T>::unblock( void ) { // unblock any waiting blocking calls
	pthread_mutex_lock(&dataMutex);
	// NOTE: linux seems to want to block on pthread_cond_signal. according to POSIX this should never happen
	//       but the implementation seems to think otherwise...
	//       This is why _block_cnt exists - we only .._signal() when at least 1 or more calls are blocking
	if(_block_cnt > 0) {
	pthread_cond_signal(&newdataCond); // let one (and only one) blocking call know to wakeup
	_block_cnt--;
	}
	pthread_mutex_unlock(&dataMutex);
}

template <class T>
void tw_safeFIFO<T>::unblockAll( void ) { // unblock any waiting blocking calls
	pthread_mutex_lock(&dataMutex);
	if(_block_cnt > 0) {	// see unblock() about this var (above)
	pthread_cond_broadcast(&newdataCond); // let all blocking calls know to wakeup
	_block_cnt = 0;
	}
	pthread_mutex_unlock(&dataMutex);
}


template <class T>
bool tw_safeFIFO<T>::removeOrBlock( T &fill ) {
	bool ret = false; // TODO - this will not work if the argument is not a pointer - (fixed through operator =())
	pthread_mutex_lock(&dataMutex);
	if(tail==head) {
		tail = NULL;
	}
	if(!head) { // if no data... wait
		ZDB_DEBUGLT("pthread_cond_wait\n",NULL);
		_block_cnt++; // see unblock() for details
		pthread_cond_wait( &newdataCond, &dataMutex ); // wait until new data arrives
		ZDB_DEBUGLT("pthread head\n",NULL);
	}
	if(tail==head) { // this check must be done again
		tail = NULL;
	}
	if(head) {
		ret = true;
		fill = head->d;
		tw_FIFO_link *oldout = head;
		head = head->next;
#ifdef _TW_WINDOWS
		HeapFree( hHeap, 0, oldout );
#else
		if(alloc)
			alloc->free( oldout );
		else
			ACE_OS::free( oldout );
#endif
		remain--;
	}
	//	if(!head)
	//	head = tail;
	pthread_mutex_unlock(&dataMutex);
	return ret;
}


template <class T>
bool tw_safeFIFO<T>::peek( T &fill ) {
	bool ret = false;
	//	if(tail==head)
	//	tail = NULL;
	pthread_mutex_lock(&dataMutex);
	if(head) {
		ret = true;
		fill = head->d;
	}
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T>
bool tw_safeFIFO<T>::peekOrBlock( T &fill ) {
	T ret = false;
	//	if(tail==head)
	//	tail = NULL;
	pthread_mutex_lock(&dataMutex);
	if(!head) {
		_block_cnt++;
		pthread_cond_wait( &newdataCond, &dataMutex ); // wait until new data arrives
	}
	if(head) {
		ret = true;
		fill = head->d;
	}
	pthread_mutex_unlock(&dataMutex);
	return ret;
}

template <class T>
int tw_safeFIFO<T>::remaining(void) {
	int ret;
	pthread_mutex_lock(&dataMutex);
	ret = remain;
	pthread_mutex_unlock(&dataMutex);
	return remain;

}

template <class T>
void tw_safeFIFO<T>::clearAll() { // delete all remaining links (and hope someone took care of the data tail each of those)
	int ret;
	tw_FIFO_link *n = NULL;
	pthread_mutex_lock(&dataMutex);
	while(head) {
		n = head->next;
		if(alloc)
			alloc->free(head);
		else
			ACE_OS::free(head);
		head = n;
	}

	head = (tw_FIFO_link *) NULL;
	tail = (tw_FIFO_link *) NULL;
	remain = 0;
	pthread_mutex_unlock(&dataMutex);
	unblockAll();
}


template <class T>
tw_safeFIFO<T>::~tw_safeFIFO() { // delete all remaining links (and hope someone took care of the data tail each of those)
	int ret;
	tw_FIFO_link *n = NULL;
	unblockAll();
	pthread_mutex_lock(&dataMutex);
	while(head) {
		n = head->next;
		if(alloc)
			alloc->free(head);
		else
			ACE_OS::free(head);
		head = n;
	}
	pthread_mutex_unlock(&dataMutex);
}


#endif // _TW_LLIST
