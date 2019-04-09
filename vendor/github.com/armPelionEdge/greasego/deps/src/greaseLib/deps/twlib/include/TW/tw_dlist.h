// tupperware container lib

// FIFO: a simple class to handle a fifo list of void pointers.

#ifndef _TW_DLIST
#define _TW_DLIST

namespace TWlib {

template <class T>
class tw_DList {
public:

	struct tw_list_link {
		T d;
		tw_list_link *next;
		tw_list_link *prev;
		void init_link( T the_d );
	};

	class tw_list_iter {
	friend class tw_DList;
	public:
		tw_list_iter() { i = NULL; }
	protected:
		tw_list_link *i;
	};

	tw_DList( void );
	~tw_DList();
#ifdef _TW_WINDOWS
	tw_DList( HANDLE theHeap );
#endif
	void add( void *d );
	T get(int index); // zero based
	T remove(int index); // zero based
	bool insert(int index, T d); // insert at this index
	void iterStartBegin(tw_list_iter &iter);
	T iterGetNext(tw_list_iter &iter);
	T iterGetCurrent(tw_list_iter &iter);
	T iterStartEnd(tw_list_iter &iter);
	T iterGetPrev(tw_list_iter &iter);
	T iterRemoveCurrent( tw_list_iter &iter); // remove the link you are at (and goes to next)

#ifdef _TW_DEBUG
	void printnodes(void);
#endif

	T getLast(void);

	int size() { return _size; }
private:
	void remove_link( tw_list_link *l );
	tw_list_link *first; // add from this end
//	tw_list_link *iter; // remove from this end
	int _size;
#ifdef _TW_WINDOWS
	HANDLE hHeap;
#endif
};

}



using namespace TWlib;

/*tw_list::tw_list_link::tw_list_link(void) {
	d = (void *) NULL;
	next = (tw_list_link *) NULL;
	prev = (tw_list_link *) NULL;
}*/

template <class T>
void tw_DList::tw_list_link::init_link( T the_d ) {
	d = the_d;
	next = (tw_list_link *) NULL;
	prev = (tw_list_link *) NULL;
}


#ifdef _TW_WINDOWS
template <class T>
tw_DList::tw_DList( HANDLE theHeap ) {
	first = (tw_list_link *) NULL;
	iter = (tw_list_link *) NULL;
	_size = 0;
	hHeap = theHeap;
}
#endif

template <class T>
tw_DList::tw_DList( void ) {
	first = (tw_list_link *) NULL;
	_size = 0;
#ifdef _TW_WINDOWS
	// todo: make hHeap the default process Heap
#endif
}

template <class T>
void tw_DList::add( T the_d ) {
#ifdef _TW_WINDOWS
	tw_list_link *newlink = (tw_list_link *) HeapAlloc( hHeap, 0, sizeof( tw_list_link ));
#else
	tw_list_link *newlink = (tw_list_link *) malloc( sizeof( tw_list_link ));
#endif
	newlink->init_link(the_d);
//	newlink->prev=NULL;
//	newlink->d = the_d;
//	newlink->prev = in;
	tw_list_link *walk = first;

	if(!first) {
		first = newlink;
	} else {
		while(walk->next)
			walk = walk->next;
		walk->next = newlink;
		walk->next->prev = walk;
	}
	_size++;

}

template <class T>
T tw_DList::get(int index) {
	tw_list_link *walk = first;
	int c = 0;

	while(walk->next && c < index) {
		c++;
		walk = walk->next;
	}

	if(c==index)
		return walk->d;
	else
		return NULL;
}

template <class T>
T tw_DList::remove(int index) {
	tw_list_link *walk = first;
	int c = 0;
	T ret = NULL;

	while(walk->next && c < index) {
		c++;
		walk = walk->next;
	}

	if(c==index) {
		remove_link(walk);
		ret = walk->d;
		//		delete walk;
		free(walk);
		_size--;
	}
    return ret;
}


template <class T>
bool tw_DList::insert(int index, T d) {
	tw_list_link *walk = first;
	int c = 0;
	tw_list_link *newlink = NULL;

	while(walk->next && c < index) {
		c++;
		walk = walk->next;
	}

	if(c==index) {
#ifdef _TW_WINDOWS
		tw_list_link *newlink = (tw_list_link *) HeapAlloc( hHeap, 0, sizeof( tw_list_link ));
#else
		tw_list_link *newlink = (tw_list_link *) malloc( sizeof( tw_list_link ));
#endif
		newlink->init_link(d);
		newlink->prev = walk->prev;
		newlink->next = walk;
		if(newlink->prev)
			newlink->prev->next = newlink;
		walk->prev = newlink;
		_size++;
		return true;
	}
	return false;
}

template <class T>
void tw_DList::iterStartBegin(tw_list_iter &iter ) {
	iter.i = first;
}

template <class T>
void tw_DList::iterStartEnd(tw_list_iter &iter) {
	tw_list_link *walk = first;
	while(walk->next)
		walk = walk->next;
	iter.i = walk;
}

template <class T>
T tw_DList::iterGetCurrent(tw_list_iter &iter) {
	if(iter.i)
		return iter.i->d;
	else
		return NULL;
}

template <class T>
T tw_DList::iterGetNext(tw_list_iter &iter) {
	T ret = NULL;
	if(iter.i) {
		iter.i=iter.i->next;
		if(iter.i)
			ret = iter.i->d;
	}
	return ret;
}

template <class T>
T tw_DList::iterGetPrev(tw_list_iter &iter) {
	T ret = NULL;
	if(iter.i) {
		iter.i=iter.i->prev;
		if(iter.i)
			ret = iter.i->d;
	}
	return ret;
}

template <class T>
T tw_DList::iterRemoveCurrent(tw_list_iter &iter) {
	T ret = NULL;
	tw_list_link *l = iter.i;
	if(iter.i) {
		ret = iter.i->d;
		iter.i = iter.i->next;
		remove_link( l );
		free( l ); // was delete l
		_size--;
	}
	return ret;
}


template <class T>
void tw_DList::remove_link( tw_list_link *l ) {
	if(l) {
		if(l == first)
			first = first->next;
		if(l->prev) {
			l->prev->next = l->next;
		}
		if(l->next) {
			l->next->prev = l->prev;
		}
	}
}


#ifdef _TW_DEBUG
template <class T>
void tw_DList::printnodes(void) {
	tw_list_link *walk = first;
	printf("tw_list %p", this );
	while(walk) {
		printf(" node[ %p ->d: %p]", walk, walk->d );
		walk = walk->next;
	}
	printf(" end_tw_list.\n");
}
#endif


template <class T>
tw_DList::~tw_DList() {
	tw_list_link *walk = first;
	tw_list_link *l;
	if(walk) {
		while(walk->next) {
			l = walk;
			walk = walk->next;
			delete l;
		}
		delete walk;
	}
}


#endif // TW_DLIST
