/*
 * tw_list.h
 *
 *  Created on: Nov 29, 2011
 *      Author: ed
 * (c) 2011, WigWag LLC
 */

#ifndef TW_LIST_H_
#define TW_LIST_H_

#include <stddef.h>

/**
 * Layout:
 *                 HEAD  --NEXT-->  || || ||  --NEXT-->  TAIL --NEXT--> NULL
 * NULL <--PREV--        <--PREV--  || || ||  <--PREV--
 *
 */



namespace TWlib {



/**
 * LList is a doubly-linked list, with iterator.
 * Requirements for T:
 * - supports assignment operator
 * - supports default constructor
 * - supports copy constructor
 * - is generally friendly and gets along well with others.
 *
 */
template <typename T, typename ALLOC>
class LList {
	struct llist_link {
		T d;
		llist_link *next;
		llist_link *prev;
		llist_link();
		void init_link( T &the_d );
		void init_link();
		void destroy();
	};
public:
	class iter {
		public:
			iter() : look(NULL) { }
			bool getNext(T &fill);
			bool getCurrent( T &fill);
			bool setCurrent( T &fill);
			bool setCurrentVal( T fill);
			bool getPrev(T &fill);
			bool removeNext(T &fill);
			bool removePrev(T &fill);
			bool atEnd();
			friend class LList<T,ALLOC>;
		protected:
			llist_link *look;
			LList<T,ALLOC> *owner;
		};

		friend class iter;
		LList( void );
		LList( ALLOC *_a );
		LList( LList<T,ALLOC> &o );

		void addToTail( T &d );
		void addToHead( T &d );
		void transferFrom( LList<T,ALLOC> &other );
		T *addEmptyTail();
		bool peekHead( T &fill ); // look at next, dont remove
		bool peekTail( T &fill ); // look at next, dont remove
		bool removeHead( T &fill );
		bool removeTail( T &fill );
		int remaining();
		void startIterHead( iter &i );
		void startIterTail( iter &i );
		void releaseIter( iter &i );
		LList &operator=( const LList &o );
		void clearAll(); // remove all nodes (does not delete T)
		void disable();
		void enable();
		~LList();
	private:
		void removeLink(llist_link *l);
		bool enabled;
		llist_link *head;
		llist_link *tail;
		int remain;
		ALLOC *alloc;
	#ifdef _TW_WINDOWS
		HANDLE hHeap;
	#endif
	//	pthread_mutex_t dataMutex; // thread safety for FIFO
};

} // end namespace

using namespace TWlib;

template <class T,class ALLOC>
bool LList<T,ALLOC>::iter::getNext(T &fill) {
	if(look) {
		fill = look->d;
		look = look->next;
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
bool LList<T,ALLOC>::iter::getPrev(T &fill) {
	if(look) {
		fill = look->d;
		look = look->prev;
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
bool LList<T,ALLOC>::iter::getCurrent(T &fill) {
	if(look) {
		fill = look->d;
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
bool LList<T,ALLOC>::iter::setCurrent(T &fill) {
	if(look) {
		look->d = fill;
		return true;
	} else
		return false;
}

///
/// set Current by passing by Value. Could be expensive on objects.
///
template <class T,class ALLOC>
bool LList<T,ALLOC>::iter::setCurrentVal(T fill) {
	if(look) {
		look->d = fill;
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
bool LList<T,ALLOC>::iter::removeNext(T &fill) {
	if(look) {
		fill = look->d;
		llist_link *oldlook = look;
		look = look->next;
		owner->removeLink(oldlook);
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
bool LList<T,ALLOC>::iter::removePrev(T &fill) {
	if(look) {
		fill = look->d;
		llist_link *oldlook = look;
		look = look->prev;
		owner->removeLink(oldlook);
		return true;
	} else
		return false;
}

template <class T,class ALLOC>
inline bool LList<T,ALLOC>::iter::atEnd() {
	if(look)
		return false;
	else
		return true;
}





template <class T,class ALLOC>
LList<T,ALLOC>::llist_link::llist_link() {
	next = (llist_link *) NULL;
	prev = (llist_link *) NULL;
}

template <class T,class ALLOC>
void LList<T,ALLOC>::llist_link::init_link() {
	::new((void*)&d) T(); // must explicity call the constructor
	next = (llist_link *) NULL;
	prev = (llist_link *) NULL;
}

template <class T,class ALLOC>
void LList<T,ALLOC>::llist_link::init_link(T &the_d) {
	::new((void*)&d) T(the_d); // must explicity call the constructor
	next = (llist_link *) NULL;
	prev = (llist_link *) NULL;
}

template <class T,class ALLOC>
void LList<T,ALLOC>::llist_link::destroy() {
	d.~T(); // call destructor of data
}





template <class T,class ALLOC>
LList<T,ALLOC>::LList( void ) : enabled( true ) {
	alloc = NULL;
//	pthread_mutex_init( &dataMutex, NULL );
	head = (llist_link *) NULL;
	tail = (llist_link *) NULL;
	remain = 0;
}

template <class T,class ALLOC>
LList<T,ALLOC>::LList( LList<T,ALLOC> &o ) : enabled( true ) {
	alloc = NULL;
//	pthread_mutex_init( &dataMutex, NULL );
	head = (llist_link *) NULL;
	tail = (llist_link *) NULL;
	remain = 0;

	*this = o; // use assignment operator to copy data
}


template <class T,class ALLOC>
LList<T,ALLOC>::LList( ALLOC *a ) : enabled( true ) {
	alloc = a;
//	pthread_mutex_init( &dataMutex, NULL );
	head = (llist_link *) NULL;
	tail = (llist_link *) NULL;
	remain = 0;
}

template <class T,class ALLOC>
inline void LList<T,ALLOC>::startIterHead( iter &i ) {
	i.look = head;
	i.owner = this;
}

template <class T,class ALLOC>
inline void LList<T,ALLOC>::startIterTail( iter &i ) {
	i.look = tail;
	i.owner = this;
}

template <class T,class ALLOC>
inline void LList<T,ALLOC>::releaseIter( iter &i ) {
	i.look = NULL;
	i.owner = NULL;
}


template <class T,class ALLOC>
void LList<T,ALLOC>::addToTail( T &the_d ) {
	if(enabled) {
	llist_link *newlink;
	newlink = (llist_link *) ALLOC::malloc( sizeof( llist_link ));
	newlink->init_link(the_d);
//	pthread_mutex_lock(&dataMutex);
	if(tail) {
		tail->next = newlink;
		newlink->prev = tail;
	}
	tail = newlink;
	if(!head) {
		head = tail;
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
void LList<T,ALLOC>::addToHead( T &the_d ) {
	if(enabled) {
	llist_link *newlink;
	newlink = (llist_link *) ALLOC::malloc( sizeof( llist_link ));
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
//	pthread_mutex_lock(&dataMutex);
//	if(in)
//		in->next = newlink;
//	in = newlink;
//	newlink->d = the_d;
	newlink->next = head;
	if(head)
		head->prev = newlink;
	head = newlink;
	if(!tail)
		tail = head;
	remain++;
//	pthread_mutex_unlock(&dataMutex);
//	unblock(); // let one blocking call know...
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif
}


/** pulls the items queued out of 'other' and placs them in this LList
 *
 * @param other
 */

template <class T,class ALLOC>
void LList<T,ALLOC>::transferFrom( LList<T,ALLOC> &other ) {
	if(enabled) {
	if(other.head) { // take care of first link - in case 'this' is empty
		if(tail) {
			tail->next = other.head;
			tail->next->prev = tail;
			tail = tail->next;
		} else {
			tail = other.head;
		}
		remain++;
	}

	if(!head) // if 'this' was empty - setup the out pointer
		head = tail;

	while(tail->next) { // count the number of nodes we just tacked on...
//		other.out = other.out->next;
		tail = tail->next; // in should be pointing at the same thing as other.out;
		remain++;
	}

#ifdef _TW_FIFO_DEBUG_ON
	if(in->next != NULL) // verify - in->next should be NULL
		TW_ERROR("LList --- error in transferFrom\n",NULL);
#endif

	other.head = NULL; // that list is now empty
	other.tail = NULL;
	other.remain = 0;
	}
#ifdef _TW_FIFO_DEBUG_ON
	else
		TW_DEBUG_LT("tw_safeFIFO: not enabled\n");
#endif
}

template <class T,class ALLOC>
void LList<T,ALLOC>::enable() {
	enabled = true;
}

template <class T,class ALLOC>
void LList<T,ALLOC>::disable() {
	enabled = false;
}



template <class T,class ALLOC>
T *LList<T,ALLOC>::addEmptyTail() {
	if(enabled) {
	llist_link *newlink;
	newlink = (llist_link *) ALLOC::malloc( sizeof( llist_link ));
	newlink->init_link();
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
//	pthread_mutex_lock(&dataMutex);
	if(tail) {
		tail->next = newlink;
		newlink->prev = tail;
	}
	tail = newlink;
	if(!head) {
		head = tail;
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
bool LList<T,ALLOC>::removeHead( T &fill ) {
	bool ret = false;
//	pthread_mutex_lock(&dataMutex);
	if(head==tail) {
		tail = NULL;
	}
	if(head) {
		ret = true;
		fill = head->d;
		llist_link *oldout = head;
		head = head->next;
		if(head)
			head->prev = NULL;
		oldout->destroy();
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
bool LList<T,ALLOC>::removeTail( T &fill ) {
	bool ret = false;
//	pthread_mutex_lock(&dataMutex);
	if(head==tail) {
		head = NULL;
	}
	if(tail) {
		ret = true;
		fill = tail->d;
		llist_link *oldout = tail;
		tail = tail->prev;
		if(tail)
			tail->next = NULL;
		oldout->destroy();
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
void LList<T,ALLOC>::removeLink( LList<T,ALLOC>::llist_link *l ) {
	if(l) {
		llist_link *oldout = l;
		if(oldout) {
			if(oldout->next)
				oldout->next->prev = oldout->prev;
			if(oldout->prev)
				oldout->prev->next = oldout->next;
		}
		if(head == l) {
			head = head->next;
		}
		if(tail == l)
			tail = tail->prev;
		remain--;
	}
}


template <class T,class ALLOC>
bool LList<T,ALLOC>::peekHead( T &fill ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
//	pthread_mutex_lock(&dataMutex);
	if(head) {
		ret = true;
		fill = head->d;
	}
//	pthread_mutex_unlock(&dataMutex);
	return ret;
}


template <class T,class ALLOC>
bool LList<T,ALLOC>::peekTail( T &fill ) {
	bool ret = false;
	//	if(in==out)
	//	in = NULL;
//	pthread_mutex_lock(&dataMutex);
	if(tail) {
		ret = true;
		fill = tail->d;
	}
//	pthread_mutex_unlock(&dataMutex);
	return ret;
}


template <class T,class ALLOC>
int LList<T,ALLOC>::remaining(void) {
	int ret;
//	pthread_mutex_lock(&dataMutex);
	ret = remain;
//	pthread_mutex_unlock(&dataMutex);
	return remain;

}

// copies other tw_safeFIFO - does not copy Allocator
template <class T,class ALLOC>
LList<T,ALLOC> &LList<T,ALLOC>::operator=( const LList<T,ALLOC> &o ) {
	llist_link *look;
	llist_link *prev;
	llist_link *newlink;

	this->enabled = o.enabled;
	this->clearAll(); // clear anything that might be there


	this->remain = 0;
	look = o.head;

	if(look) {
	newlink = (llist_link *) ALLOC::malloc( sizeof( llist_link ));
	newlink->init_link(look->d);
	prev = newlink;
	this->head = newlink;
	look = look->next;
	this->remain++;
	}

	while(look) {
		newlink = (llist_link *) ALLOC::malloc( sizeof( llist_link ));
		newlink->init_link(look->d);
		prev->next = newlink; // link to link behind us
		newlink->prev = prev; // backwards link
		prev = newlink;       // move forward
		look = look->next;    // move the source forward
		this->remain++;
	}

	this->tail = prev;
	return *this;
}

template <class T,class ALLOC>
void LList<T,ALLOC>::clearAll() { // delete all remaining links (and hope someone took care of the data in each of those)
	int ret;
	llist_link *n = NULL;
//	pthread_mutex_lock(&dataMutex);
	while(head) {
		n = head->next;
		head->destroy();
		ALLOC::free(head);
		head= n;
	}

	head = (llist_link *) NULL;
	tail = (llist_link *) NULL;
	remain = 0;
//	pthread_mutex_unlock(&dataMutex);
}


template <class T,class ALLOC>
LList<T,ALLOC>::~LList() { // delete all remaining links (and hope someone took care of the data in each of those)
//	int ret;
	llist_link *n = NULL;
//	pthread_mutex_lock(&dataMutex);
	clearAll();
//	pthread_mutex_unlock(&dataMutex);
}





#endif /* TW_LIST_H_ */
