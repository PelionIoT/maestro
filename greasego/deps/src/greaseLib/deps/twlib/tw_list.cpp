// tupperware container lib

// simple FIFO data struct...


// fifo.cpp

// simple tw_list class
#ifdef _TW_WINDOWS
#define WIN32_LEAN_AND_MEAN
#include "stdafx.h"
#include <windows.h>
#else
#include <stdlib.h>
#endif

#ifdef _TW_DEBUG
#include <stdio.h>
#endif

#include <TW/tw_list.h>

#ifndef NULL
#define NULL 0
#endif

using namespace TWlib;

/*tw_list::tw_list_link::tw_list_link(void) {
	d = (void *) NULL;
	next = (tw_list_link *) NULL;
	prev = (tw_list_link *) NULL;
}*/

void tw_list::tw_list_link::init_link( void *the_d ) {
	d = the_d;
	next = (tw_list_link *) NULL;
	prev = (tw_list_link *) NULL;
}


#ifdef _TW_WINDOWS
tw_DList::tw_DList( HANDLE theHeap ) {
	first = (tw_list_link *) NULL;
	iter = (tw_list_link *) NULL;
	_size = 0;
	hHeap = theHeap;
}
#endif

tw_list::tw_list( void ) {
	first = (tw_list_link *) NULL;
	_size = 0;
#ifdef _TW_WINDOWS
	// todo: make hHeap the default process Heap
#endif
}


void tw_list::add( void *the_d ) {
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

void *tw_list::get(int index) {
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

void *tw_list::remove(int index) {
	tw_list_link *walk = first;
	int c = 0;
	void *ret = NULL;

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


bool tw_list::insert(int index, void *d) {
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


void tw_list::iterStartBegin(tw_list_iter &iter ) {
	iter.i = first;
}

void tw_list::iterStartEnd(tw_list_iter &iter) {
	tw_list_link *walk = first;
	while(walk->next)
		walk = walk->next;
	iter.i = walk;
}

void *tw_list::iterGetCurrent(tw_list_iter &iter) {
	if(iter.i)
		return iter.i->d;
	else
		return NULL;
}

void *tw_list::iterGetNext(tw_list_iter &iter) {
	void *ret = NULL;
	if(iter.i) {
		iter.i=iter.i->next;
		if(iter.i)
			ret = iter.i->d;
	}
	return ret;
}

void *tw_list::iterGetPrev(tw_list_iter &iter) {
	void *ret = NULL;
	if(iter.i) {
		iter.i=iter.i->prev;
		if(iter.i)
			ret = iter.i->d;
	}
	return ret;
}

void *tw_list::iterRemoveCurrent(tw_list_iter &iter) {
	void *ret = NULL;
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


void tw_list::remove_link( tw_list_link *l ) {
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
void tw_list::printnodes(void) {
	tw_list_link *walk = first;
	printf("tw_list %p", this );
	while(walk) {
		printf(" node[ %p ->d: %p]", walk, walk->d );
		walk = walk->next;
	}
	printf(" end_tw_list.\n");
}
#endif


tw_list::~tw_list() {
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

