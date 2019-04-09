// WigWag LLC
// (c) 2010
// tw_bufblk.h
// Author: ed
// Sep 9, 2010

/*
 * tw_bufblk.h
 *
 *  Created on: Sep 9, 2010
 *      Author: ed
 */

#ifndef TW_BUFBLK_H_
#define TW_BUFBLK_H_

#include <string>
#include <iostream>

#include <TW/tw_log.h>
#include <TW/tw_utils.h>

#include <malloc.h>
#include <string.h>
#include <pthread.h>
#include <limits.h>
#include <stdio.h>

#include <string>
#include <sstream>
#include <iostream>

using namespace std;

namespace TWlib {

template <class ALLOC>
class BufBlk;

/**
 * The basic abstraction of a chunk of memory.
 * -Contigous
 * -Is not aware of other memory
 * -Has a reference count for itself
 */
template <class ALLOC>
class MemBlk {
public:
	MemBlk( int size );
	MemBlk( int size, ALLOC *a );
	MemBlk( void *data, int size, bool manage = true );
	int getRefCount();
	void checkout();            // tells tw_memblk that another tw_bufblk is referencing it, _ref_count++
	void release();             // decreases the ref count
	int size();                 // report block size
	bool resize(int size);      // attempts to resize the block
	void fillWith(char v);      // fills the memblk with a byte (good for init)
	void *base();               // report base of block
	MemBlk *deepCopy();         // makes an entirely new copy of the memory
	MemBlk *deepCopy(ALLOC *a);
	std::string &hexDump(std::string &in);
	friend class BufBlk<ALLOC>;
protected:
	int _concatHexDump(std::ostringstream &outs, char *rdptr, char *wrtptr, int max);
	int _ref_count;
	pthread_mutex_t _refMutex; // thread safety for buffer, ref count
	void *_base;                // the memory
	int _size;                  // size of memory region
	bool _manage;
	bool _proper;
};

template <class ALLOC>
class BufBlkIter;

/**
 * The abstraction of a buffer.
 * - Has one or more MemBlks
 * - Maintains ref count by way of MemBlk
 */
template <class ALLOC>
class BufBlk {
protected:
	BufBlk( MemBlk<ALLOC> *m );
	void assignFromBlk( const BufBlk<ALLOC> &o );
	MemBlk<ALLOC> *_blk;
	char *_rd_ptr;
	char *_wr_ptr;
	char *_mark;   // a pointer use to move the _rd_ptr back to the beginning w/ rewind(N) - if not set, it is the same as base()
	BufBlk<ALLOC> *_nextblk;
	uint32_t _flags;
#ifdef _TW_BUFBLK_DEBUG_STACK_
	StackDump *_stackOnCreate; // the call where the BufBlk was originally created
	StackDump *_stackOnRelease; // the call where the BufBlk is fully released
#endif
public:
	friend class twbufblkIter;
	BufBlk();
	BufBlk( BufBlk &o );
	BufBlk( void *data, int size, bool manage = true );
	BufBlk( int size );
	BufBlk( int size, ALLOC *a );
	BufBlk( int size, BufBlk<ALLOC> *next, ALLOC *a );

	~BufBlk();

	void release();
	BufBlk *nexblk();
	void setNexblk( BufBlk<ALLOC> * );
	void addToEnd( BufBlk<ALLOC> * );

	BufBlk *eatBytes( int rd_offset, int size );
	bool getContigBlock( BufBlk<ALLOC> *&outblk, int rd_offset, int size );
	bool isContigous(char *pos, int len);
	bool isContigousReadble(char *pos, int len);

	int copyFrom( const void *src, int size );
	void fillWith( char v );
	void reset();
	void setMark();
	char *mark();
	bool resize(int size);
	void rewind(int offset);
	int length();
	int total_length();
	int capacity();
	int total_capacity();
	int freespace();
	bool isValid();
	void *base();
	char *rd_ptr();
	char *wr_ptr();
	void rd_ptr(char *p);
	void wr_ptr(char *p);
	void inc_rd_ptr( int x );
	void inc_wr_ptr( int x );
	void dec_rd_ptr( int x );
	void dec_wr_ptr( int x );

	BufBlk *deepCopy();
	BufBlk *deepCopy(ALLOC *a);

	void set_flags( uint32_t flags );
	uint32_t get_flags();

	BufBlk& operator=(const BufBlk<ALLOC> &o);
	BufBlk *duplicate();
	int getRefCount();
	MemBlk<ALLOC> *memblk() { return _blk; } // returns a pointer to underlying MemBlk
	std::string &hexDump(std::string &outs, int max = 0);
};

template <class ALLOC>
class BufBlkIter {
public:
	BufBlkIter( BufBlk<ALLOC> &b );
	bool getNextChunk( char *&start, int &size, int max = -1 );
	bool previewNextChunk( char *&start, int &size, int max = -1 );
	bool copyNextChunks( char *to, int space, int &copied);
	void rewind();
	~BufBlkIter();
protected:
	BufBlk<ALLOC> *_cur; // current bufblk we are on
	BufBlk<ALLOC> *_start;
	int _rd_offset;  // current offset from that blk's rd_ptr
};

} // end namespace





///////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////

// For extra debuggin, uncomment:
//#define _DEBUG_TW_BUFBLK 1


using namespace TWlib;
using namespace std;

template <class ALLOC>
MemBlk<ALLOC>::MemBlk(int size) :
	_ref_count( 1 ),
	_base( 0 ),
	_size( 0 ),
	_manage( true ),
	_proper( true )
{
	_base = ALLOC::malloc( size );
	if(_base) {
		_size = size;
	} else
		_proper = false;
	pthread_mutex_init( &_refMutex, NULL );
}

template <class ALLOC>
MemBlk<ALLOC>::MemBlk(int size, ALLOC *a) :
	_ref_count( 1 ),
	_base( 0 ),
	_size( 0 ),
	_manage( true ),
	_proper( true )
{
	_base = a->i_malloc( size );
	if(_base) {
		_size = size;
	} else
		_proper = false;
	pthread_mutex_init( &_refMutex, NULL );
}

template <class ALLOC>
MemBlk<ALLOC>::MemBlk(void *data, int size, bool manage ) :
	_ref_count( 1 ),
	_base( data ),
	_size( size ),
	_manage( manage )
{ // def: manage = true
	pthread_mutex_init( &_refMutex, NULL );
}

/**
 * @return returns the reference count for this block. When reference count reaches zero means the block will be free'd, and
 * this object deleted.
 */
template <class ALLOC>
int MemBlk<ALLOC>::getRefCount() {
	int ret = 0;
	pthread_mutex_lock(&_refMutex);
	ret = _ref_count;
	pthread_mutex_unlock(&_refMutex);
	return ret;
}

/**
 * tells tw_memblk that another tw_bufblk is referencing it, _ref_count++
 */
template <class ALLOC>
void MemBlk<ALLOC>::checkout() {
	pthread_mutex_lock(&_refMutex);
	_ref_count++;
	pthread_mutex_unlock(&_refMutex);
}

template <class ALLOC>
void MemBlk<ALLOC>::release() {             // decreases the ref count
	bool remove = false;
	pthread_mutex_lock(&_refMutex);
	_ref_count--;
	if(_ref_count < 1)
		remove = true;
	pthread_mutex_unlock(&_refMutex);
//	printf("release %x - ref: %d\n", _base, _ref_count);
	if(remove) {
//		printf("remove: %x\n", _base);
		if(_manage)
			ALLOC::free( _base ); // free the memory block
		delete this;         // good bye...
	}
	pthread_mutex_destroy(&_refMutex);
}

template <class ALLOC>
MemBlk<ALLOC> *MemBlk<ALLOC>::deepCopy() {
	MemBlk<ALLOC> *ret = new MemBlk<ALLOC>(this->_size);
	ALLOC::memcpy(ret->_base,this->_base,this->_size);
	return ret;
}

template <class ALLOC>
MemBlk<ALLOC> *MemBlk<ALLOC>::deepCopy(ALLOC *a) {
	MemBlk<ALLOC> *ret = new MemBlk<ALLOC>(this->_size,a);
	ALLOC::memcpy(ret->_base,this->_base,this->_size);
	return ret;
}

template <class ALLOC>
bool MemBlk<ALLOC>::resize(int size) {                 // report block size
	void *newb = NULL;
	newb = ALLOC::realloc(this->_base,size);
	if(newb) {
		this->_base = newb;
		this->_size = size;
		return true;
	} else
		return false;
}


template <class ALLOC>
int MemBlk<ALLOC>::size() {                 // report block size
	return _size;
}

template <class ALLOC>
void *MemBlk<ALLOC>::base() {               // report base of block
	return _base;
}

template <class ALLOC>
void MemBlk<ALLOC>::fillWith(char v) {
	ALLOC::memset(this->_base,(int) v, this->_size);
}

template <class ALLOC>
int MemBlk<ALLOC>::_concatHexDump(std::ostringstream &outs, char *rdptr, char *wrtptr, int max) {
	int left = max;
	char buf[8];
	if(!max)
		left = INT_MAX;
	char *walk = (char *) _base;
	int c = 0;
	snprintf(buf,8,"(%d",(int) _size);
	outs << buf;
	outs << ")[";
//	outs.setf(std::ios_base::hex,std::ios_base::basefield);
	while(c < _size) {
		if(left < 6) // typical hex digit FF (2 chars) plus  ']' plus possible RD*
			break;
		if(rdptr == walk) {
			if(rdptr == wrtptr)
				outs << "R*W";
			else
				outs << "RD*";
		} else if(wrtptr == walk)
			outs << "WR*";
	//	outs << (int) *walk; // send as in so its hex
		snprintf(buf,8,"%02X",(unsigned char) *walk);
		outs << buf;
		c++;
		walk++;
		left --;
		if(c < _size) {
			outs << ",";
			left--;
		}
	}
	if(walk == wrtptr) // if the block is fully written to, then the wr_ptr is point to byte after the end of the block.
		outs << "WR*";
	outs << "]";
	return max - left;
}

template <class ALLOC>
std::string &MemBlk<ALLOC>::hexDump(std::string &out) {
	std::ostringstream outs;
	out.clear();
	outs.str(out);
	_concatHexDump(outs,0,0,0);
	out = outs.str();
	return out;
}


/** Default constructor. Object unusuable unless assigned to another.
 */
template <class ALLOC>
BufBlk<ALLOC>::BufBlk(  ) :
_blk( NULL ),
_wr_ptr( NULL ),
_rd_ptr( NULL ),
_mark( NULL ),
_nextblk( NULL ),
_flags( 0 )
{
#ifdef _TW_BUFBLK_DEBUG_STACK_
	_stackOnCreate = StackDump::getStackTrace();
	_stackOnRelease = NULL;
#endif
}

/** internal constructor... dups the tw_memblk.
 */
template <class ALLOC>
BufBlk<ALLOC>::BufBlk( MemBlk<ALLOC> *m ) :
_blk( m ),
_wr_ptr( NULL ),
_rd_ptr( NULL ),
_mark( NULL ),
_nextblk( NULL ),
_flags( 0 )
{
	_blk->checkout();
	_wr_ptr = (char *) _blk->_base; // set pointers at base
	_rd_ptr = (char *) _blk->_base;
#ifdef _TW_BUFBLK_DEBUG_STACK_
	_stackOnCreate = StackDump::getStackTrace();
	_stackOnRelease = NULL;
#endif
}


/** Copy constructor.
 * Reference count on the underlying tw_memblk is incremented 1.
 *
 * @param o
 * @return
 */
template <class ALLOC>
BufBlk<ALLOC>::BufBlk( BufBlk<ALLOC> &o ) :
_blk( o._blk ),
_wr_ptr( o._wr_ptr ),
_rd_ptr( o._rd_ptr ),
_mark( o._mark ),
_nextblk( NULL ),
_flags( 0 )
{
#ifdef _TW_BUFBLK_DEBUG_STACK_
	_stackOnCreate = NULL;
	_stackOnRelease = NULL;
#endif
	assignFromBlk( o );
}

template <class ALLOC>
inline void BufBlk<ALLOC>::set_flags( uint32_t flags ) { _flags = flags; }

template <class ALLOC>
inline uint32_t BufBlk<ALLOC>::get_flags() { return _flags; }


/** internal only - this is used when the tw_bufblk is duplicate()'d..
 * assumes all internal values are empty.
 * copies all data from [o]. copies all data, duplicates entire chain
 * @param o tw_bufblk to get data from.
 */
template <class ALLOC>
void BufBlk<ALLOC>::assignFromBlk( const BufBlk<ALLOC> &o ) {
	if(o._blk)
		o._blk->checkout(); // increase ref count on memblk
	_blk = o._blk;
	_nextblk = NULL;
	_rd_ptr = o._rd_ptr;
	_wr_ptr = o._wr_ptr;
	_mark = o._mark;
	BufBlk *walko = o._nextblk;
	BufBlk *walk = this;
	while(walko) { // copy the chain... (non-recursive more efficient)
#ifdef _TW_BUFBLK_DEBUG_STACK_
		if(!walko->_blk) {
			string s;
			TW_DEBUG_LT("BufBlk asked to dup NULL MemBLK, trace: %s\n", StackDump::stackDump(s).c_str());
		}
#endif
		walk->_nextblk = new BufBlk( walko->_blk );
		walk->_nextblk->_rd_ptr = walko->_rd_ptr;
		walk->_nextblk->_wr_ptr = walko->_wr_ptr;
		walk = walk->_nextblk;
		walko = walko->_nextblk;
 	}
#ifdef _TW_BUFBLK_DEBUG_STACK_
	if(_stackOnCreate)
		delete _stackOnCreate;
	_stackOnCreate = StackDump::getStackTrace();
#endif
}

/**
 * Provides a complete copy of the BufBlk chain, to include all chained BufBlk's
 * @return
 */
template <class ALLOC>
BufBlk<ALLOC> *BufBlk<ALLOC>::deepCopy() {
	BufBlk *ret = new BufBlk();
	BufBlk *walknew = ret;
	BufBlk *walk = this;
	while(walk) {
		if(walk->_blk) {
			walknew->_blk = walk->_blk->deepCopy();
			walknew->_flags = walk->_flags;
			walknew->_rd_ptr = (char *) walknew->_blk->base() + (walk->_rd_ptr - (char *) walk->_blk->base());
			walknew->_wr_ptr = (char *) walknew->_blk->base() + (walk->_wr_ptr - (char *) walk->_blk->base());
			walknew->_mark =   (char *) walknew->_blk->base() + (walk->_mark -   (char *) walk->_blk->base());
		}
		walk = walk->_nextblk;
		if(walk) {
			walknew->_nextblk = new BufBlk();
			walknew = walknew->_nextblk;
		}
	}
	return ret;
}

template <class ALLOC>
BufBlk<ALLOC> *BufBlk<ALLOC>::deepCopy(ALLOC *a) {
	BufBlk *ret = new BufBlk();
	BufBlk *walknew = ret;
	BufBlk *walk = this;
	while(walk) {
		if(walk->_blk) {
			walknew->_blk = walk->_blk->deepCopy(a);
			walknew->_flags = walk->_flags;
			walknew->_rd_ptr = (char *) walknew->_blk->base() + (walk->_rd_ptr - (char *) walk->_blk->base());
			walknew->_wr_ptr = (char *) walknew->_blk->base() + (walk->_wr_ptr - (char *) walk->_blk->base());
			walknew->_mark =   (char *) walknew->_blk->base() + (walk->_mark -   (char *) walk->_blk->base());
		}
		walk = walk->_nextblk;
		if(walk) {
			walknew->_nextblk = new BufBlk();
			walknew = walknew->_nextblk;
		}
	}
	return ret;
}

/**
 * Assignment operator.
 * Reference count on the underlying tw_memblk is incremented 1.
 * @param o
 * @return
 */
template <class ALLOC>
BufBlk<ALLOC>& BufBlk<ALLOC>::operator=(const BufBlk<ALLOC> &o) {
	if(_nextblk)
		_nextblk->release(); // release next block also
	if(_blk)
		_blk->release(); // release underlying memblk

	assignFromBlk( o );
}


template <class ALLOC>
int BufBlk<ALLOC>::getRefCount() {
	if(_blk)
		return _blk->getRefCount();
	else
		return 0;
}

template <class ALLOC>
BufBlk<ALLOC> *BufBlk<ALLOC>::duplicate() {
	return new BufBlk( *this );
}

/** This constuctor attaches a bufblk to an existing block of memory. If manage is true,
 * it will also manage this buffer, and free it when no references exist.
 * Reference count start at one upon creation.
 *
 * @param data the base address of the block to manage
 * @param size the size of this block
 * @param manage if true, the bufblk will free() the block upon final release(). otherwise it will leave it upon final release()
 */
template <class ALLOC>
BufBlk<ALLOC>::BufBlk( void *data, int size, bool manage ) :
_blk( NULL ),
_rd_ptr( NULL ),
_wr_ptr( NULL ),
_mark( NULL ),
_nextblk( NULL )
{  // def: manage = true
	_blk = new MemBlk<ALLOC>( data, size, manage );
	_rd_ptr = (char *) _blk->_base;
	_wr_ptr = (char *) _blk->_base;
	_mark = (char *) _blk->_base;
#ifdef _TW_BUFBLK_DEBUG_STACK_
	_stackOnCreate = StackDump::getStackTrace();
	_stackOnRelease = NULL;
#endif

}

/**
 * Vanilla constructor, create an underlying tw_memblk for the amount size. Reference count starts at 1.
 * When release() is called, the ref count decrements, and this object deletes itself.
 * On ref count 0, the underlying memory block deletes itself.
 * @param size in bytes of memory required
 */
template <class ALLOC>
BufBlk<ALLOC>::BufBlk( int size ) :
_blk( NULL ),
_wr_ptr( NULL ),
_rd_ptr( NULL ),
_mark( NULL ),
_nextblk( NULL )
{  // def: manage = true
	_blk = new MemBlk<ALLOC>( size );
	_rd_ptr = (char *) _blk->_base;
	_wr_ptr = (char *) _blk->_base;
	_mark = (char *) _blk->_base;
#ifdef _TW_BUFBLK_DEBUG_STACK_
	_stackOnCreate = StackDump::getStackTrace();
	_stackOnRelease = NULL;
#endif

}


/**
 * Vanilla constructor with instantiated ALLOC, create an underlying tw_memblk for the amount size. Reference count starts at 1.
 * When release() is called, the ref count decrements, and this object deletes itself.
 * On ref count 0, the underlying memory block deletes itself.
 * @param size in bytes of memory required
 */
template <class ALLOC>
BufBlk<ALLOC>::BufBlk( int size, ALLOC *a ) :
_blk( NULL ),
_wr_ptr( NULL ),
_rd_ptr( NULL ),
_mark( NULL ),
_nextblk( NULL )
{  // def: manage = true
	_blk = new MemBlk<ALLOC>( size, a );
	_rd_ptr = (char *) _blk->_base;
	_wr_ptr = (char *) _blk->_base;
	_mark = (char *) _blk->_base;
#ifdef _TW_BUFBLK_DEBUG_STACK_
	_stackOnCreate = StackDump::getStackTrace();
	_stackOnRelease = NULL;
#endif

}

/**
 * Vanilla constructor with instantiated ALLOC, create an underlying tw_memblk for the amount size. Reference count starts at 1.
 * Also puts next, as the following BufBlk to this one (_nextblk).
 * When release() is called, the ref count decrements, and this object deletes itself.
 * On ref count 0, the underlying memory block deletes itself.
 * @param size in bytes of memory required
 */
template <class ALLOC>
BufBlk<ALLOC>::BufBlk( int size, BufBlk<ALLOC> *next, ALLOC *a ) :
_blk( NULL ),
_rd_ptr( NULL ),
_wr_ptr( NULL ),
_mark( NULL ),
_nextblk( next )
{  // def: manage = true
	_blk = new MemBlk<ALLOC>( size, a );
	_rd_ptr = (char *) _blk->_base;
	_wr_ptr = (char *) _blk->_base;
	_mark = (char *) _blk->_base;
#ifdef _TW_BUFBLK_DEBUG_STACK_
	_stackOnCreate = StackDump::getStackTrace();
	_stackOnRelease = NULL;
#endif
}


/**
 * destructor also release next block, and underlying block.
 * Do not call ->release() and the destructor;
 */
template <class ALLOC>
BufBlk<ALLOC>::~BufBlk() {
	if (_nextblk)
		_nextblk->release(); // release next block also
	if (_blk)
		_blk->release(); // release underlying memblk
#ifdef _TW_BUFBLK_DEBUG_STACK_
	if(_stackOnCreate)
		delete _stackOnCreate;
#endif
}

/**
 *  releases the underlying memory block - meaning reference count decreases.
 *  This function deletes this object - the object is no longer valid after calling it.
 *  calling release() also releases any chained blocks.
 *  Warning: you can't call this if you are on the stack.
 */
template <class ALLOC>
void BufBlk<ALLOC>::release() {
	if(_nextblk)
		_nextblk->release(); // release next block also
	if(_blk)
		_blk->release(); // release underlying memblk
#ifdef _TW_BUFBLK_DEBUG_STACK_
	string s;
	TW_DEBUG("BufBlk->release() @ %s\n   was created @ %s\n",StackDump::stackDump(s).c_str(), _stackOnCreate->stackDump(s).c_str());
#endif
	_nextblk = NULL;
	_blk = NULL;
	delete this;     // goodbye
}
/**
 * Returns the next chained tw_bufblk.
 * @return a pointer to the next chained tw_bufblk
 */
template <class ALLOC>
BufBlk<ALLOC> *BufBlk<ALLOC>::nexblk() {
	return _nextblk;
}

/**
 * Assign the block to follow this block. This next block will be chained to this one, and will release() when this one does.
 * DOES NOT increase reference count.
 * @param b BufBlk pointer
 */
template <class ALLOC>
void BufBlk<ALLOC>::setNexblk( BufBlk *b ) {
	_nextblk = b;
}

/**
 * Adds the 'b' BufBlk to end of chain of blocks.
 * DOES NOT increase reference count.
 * @param b BufBlk pointer
 */
template <class ALLOC>
void BufBlk<ALLOC>::addToEnd( BufBlk *b ) {
	BufBlk *look = this;
	while(look->_nextblk) {
		look=look->_nextblk;
	}
	look->_nextblk = b;
}

/** Copy data into the buffer. This moves the wr_ptr 'size' forward (or less, if not enough space)
 *
 * @param src base pointer of data
 * @param size in bytes to copy
 * @return the amount of data copied in bytes
 */
template <class ALLOC>
int BufBlk<ALLOC>::copyFrom( const void *src, int size ) {
	int s = size;
	if(BufBlk<ALLOC>::freespace() < size)
		s = BufBlk<ALLOC>::freespace();
	ALLOC::memcpy(_wr_ptr, src, s);
	_wr_ptr = _wr_ptr + s;
	return s;
}

/**
 * Makes rd_ptr = base, and wr_ptr = base.
 */
template <class ALLOC>
void BufBlk<ALLOC>::reset() {
	_rd_ptr = (char *) _blk->base();
	_wr_ptr = (char *) _blk->base();
	_mark = _rd_ptr;
}

template <class ALLOC>
bool BufBlk<ALLOC>::isValid() {
	if(_blk) return true;
	else return false;
}

/**
 * Attempts to resize the BufBlk's contiguous buffer, true if success
 * @param offset bytes from the base
 */
template <class ALLOC>
bool BufBlk<ALLOC>::resize(int size) {
	bool ret = false;
	if(_blk) {
		char *oldb = (char *) _blk->_base;
		if(_blk->resize(size)) {
			_rd_ptr = ((char *) _blk->_base) + (_rd_ptr - oldb);
			_wr_ptr = ((char *) _blk->_base) + (_wr_ptr - oldb);
			_mark =   ((char *) _blk->_base) +   (_mark - oldb);
			ret = true;
		}
	}
	return ret;
}

/**
 * sets rd_ptr to mark() + offset [bytes]
 * mark() is by default set to base()
 * @param offset bytes from the base
 */
template <class ALLOC>
void BufBlk<ALLOC>::rewind(int offset) {
//	_rd_ptr = ((char *) _blk->base()) + _mark + offset;
	_rd_ptr = _mark + offset;
}

/**
 * Set's Begin mark to the current rd_ptr. Calling rewind(0) will reset the rd_ptr back to this mark.
 */
template <class ALLOC>
void BufBlk<ALLOC>::setMark() {
	_mark = _rd_ptr;
}

/**
 * Returns the _mark pointer, which should always be >= the base()
 * @return
 */
template <class ALLOC>
char *BufBlk<ALLOC>::mark() {
#ifdef _TW_BUFBLK_DEBUG_STACK_
	if(!_blk) {
		string s;
		_stackOnCreate->stringify(s);
		TW_DEBUG_L("BufBlk->mark() *** _blk == NULL !!, call @: %s\n",s.c_str());
	}
#endif
	return _mark;
}


/**
 * wr_ptr - rd_ptr
 * @return
 */
template <class ALLOC>
int BufBlk<ALLOC>::length() {
	return (int) ((unsigned long) _wr_ptr - (unsigned long) _rd_ptr);
}

template <class ALLOC>
int BufBlk<ALLOC>::total_length() {
	BufBlk *look = this;
	int ret = 0;
	while(look) {
		ret+=look->length();
		look=look->_nextblk;
	}
	return ret;
}

/**
 * @return the total amount of space the under lying memory block has.
 */
template <class ALLOC>
int BufBlk<ALLOC>::capacity() {
	return _blk->size();
}

template <class ALLOC>
int BufBlk<ALLOC>::total_capacity() {
	BufBlk *look = this;
	int ret = 0;
	while(look) {
		ret+=look->capacity();
		look=look->_nextblk;
	}
	return ret;
}

/**
 * Fills the entire underlying memory block with value 'v']
 * (Irregardless of rd_ptr and wr_ptr)
 * @param v
 */
template <class ALLOC>
void BufBlk<ALLOC>::fillWith(char v) {
	_blk->fillWith(v);
}

/**
 * dumps the entire chain of memory blocks, as a list of hexadecimal number (per byte)
 * @param in string object the dump will be placed in
 * @param max number of chars to put in string (0 is no max, dump everything in)
 */
template <class ALLOC>
std::string &BufBlk<ALLOC>::hexDump(std::string &out, int max) {
	ostringstream outs;
	out.clear();
	outs.str(out);
	outs.setf(std::ios_base::hex,std::ios_base::basefield);
	BufBlk *look = this;
	int left = max - 3; // for "END"
	while(look) {
		left = left - 4;
		if(max && left <= 0) break;
		if(max)
			left = left - look->_blk->_concatHexDump(outs, look->_rd_ptr, look->_wr_ptr, left);
		else
			look->_blk->_concatHexDump(outs, look->_rd_ptr, look->_wr_ptr, 0);
		outs << "->";
		look=look->_nextblk;
	}
	outs << "END";
	out = outs.str();
	return out;
}

/**
 * @return the amount of space in bytes, the write pointer can still use
 */
template <class ALLOC>
int BufBlk<ALLOC>::freespace() {
	return (int) _blk->size() - ((unsigned long) _wr_ptr - (unsigned long) _blk->base());
}

/**
 * @return the base of the underlying memory block
 */
template <class ALLOC>
void *BufBlk<ALLOC>::base() {
#ifdef _TW_BUFBLK_DEBUG_STACK_
	if(!_blk) {
		string s;
		_stackOnCreate->stringify(s);
		TW_DEBUG_L("BufBlk->base() *** _blk == NULL !!, call @: %s\n",s.c_str());
	}
#endif
	return _blk->base();
}

template <class ALLOC>
char *BufBlk<ALLOC>::rd_ptr() {
	return _rd_ptr;
}

template <class ALLOC>
char *BufBlk<ALLOC>::wr_ptr() {
	return _wr_ptr;
}

template <class ALLOC>
void BufBlk<ALLOC>::inc_rd_ptr( int x ) {
	_rd_ptr += x;
}

template <class ALLOC>
void BufBlk<ALLOC>::inc_wr_ptr( int x ) {
	_wr_ptr += x;
}

template <class ALLOC>
void BufBlk<ALLOC>::dec_rd_ptr( int x ) {
	_rd_ptr -= x;
}

template <class ALLOC>
void BufBlk<ALLOC>::dec_wr_ptr( int x ) {
	_wr_ptr -= x;
}

/**
 * @param p - rd_ptr is set to p
 */
template <class ALLOC>
void BufBlk<ALLOC>::rd_ptr(char *p) {
	_rd_ptr = p;
}
/**
 * @param p - wr_ptr is set to p
 */
template <class ALLOC>
void BufBlk<ALLOC>::wr_ptr(char *p) {
	_wr_ptr = p;
}

/**
 * returns true if the area pointed to by 'pos' and of length 'len'
 * fits contiguously into the BufBlk. Note this does not consider
 * the rd_ptr and wr_ptr.
 * @param pos
 * @param len
 * @return true if the area is contigously inside the BufBlk
 */
template <class ALLOC>
bool BufBlk<ALLOC>::isContigous(char *pos, int len) {
	if(pos >= _blk->base() && ((pos+len) <= (_blk->base() + _blk->size())))
		return true;
	else
		return false;
}

/**
 * returns true if the area pointed to by 'pos' and of length 'len'
 * fits contiguously into the readble portion of the BufBlk. The readable area
 * is of length() bytes in size.
 * @param pos
 * @param len
 * @return true is contigously readable
 */
template <class ALLOC>
bool BufBlk<ALLOC>::isContigousReadble(char *pos, int len) {
	if(pos >= _rd_ptr && ((pos+len) <= _wr_ptr))
		return true;
	else
		return false;
}

/**
 * returns to the caller a contiguous memory block
 * @param rd_offset offset where you would like the contiguous block to begin, from where the rd_ptr of the current block is(the rd_ptr will not be moved)
 * The offset needs to be in range of the entire block chain, but can be beyond the bounds of the rd_ptr of the first block. Only readable sections are considered
 * in this function.
 * @param outblk an emptry BufBlk pointer. Will be filled
 * @param size size of the are desired. If the size is too large false is returned.
 * @return returns true is success, or false if the operation was not possible, out of bound, etc.
 */
template <class ALLOC>
bool BufBlk<ALLOC>::getContigBlock( BufBlk *&outblk, int rd_offset, int size ) {
	bool ret = false;
	// find the block where rd_offset starts
	int need = rd_offset;
	BufBlk *look = this;
	while((need >= 0) && look) {
		if(look->length() > need)
			break; // look is the block where we begin.
		need -= look->length();
		look = look->_nextblk;
	}
	if(look && (look->length() > need)) { // if we were successful
		if(look->length() - need - size >= 0) {
			// cool - we can just duplicate that block, and return it
			outblk = new BufBlk(look->_blk);
//			outblk->_rd_ptr = look->_rd_ptr + rd_offset;
			outblk->_rd_ptr = look->_rd_ptr + need;
			outblk->setMark();
			outblk->_wr_ptr = outblk->_rd_ptr + size;
			ret = true;
		} else {
			if(look->_nextblk) {
				outblk = new BufBlk(size);
				int total = size;
				outblk->copyFrom(look->_rd_ptr + need, look->length() - need);
				total -= look->length() - need;
				look = look->_nextblk;
				while(look && (total > 0)) {
					need = (look->length() < total) ? look->length() : total;
					outblk->copyFrom(look->_rd_ptr, need);
					total -= need;
					look=look->_nextblk;
				}
				if(!total)
					ret = true;
				else {
					TW_DEBUG_L("Failure at getContigBlock: total %d need %d\n",total,need);
				}
			} // else, failure, b/c we don't have enough space
		}
	} else
		TW_DEBUG_L("Failure (2) at getContigBlock: need %d\n",need);
	return ret;
}

/**
 * removes from the BufBllk chain N bytes (by either adjusting rd_ptr or releasing blocks)
 * @param rd_offset offset where you would like the contiguous block to begin, from where the rd_ptr of the current block is(the rd_ptr will not be moved)
 * The offset needs to be in range of the entire block chain, but can be beyond the bounds of the rd_ptr of the first block. Only readable sections are considered
 * in this function.
 * @param outblk an emptry BufBlk pointer. Will be filled
 * @param size size of the are desired. If the size is too large false is returned.
 * @return returns the new head of the
 */
template <class ALLOC>
BufBlk<ALLOC> *BufBlk<ALLOC>::eatBytes( int rd_offset, int size ) {
	// find the block where rd_offset starts
	int need = rd_offset;
	BufBlk *head = this;
	BufBlk *look = this;
	BufBlk *prev = NULL;
	while((need >= 0) && look) {
		if(look->length() > need)
			break; // look is the block where we begin.
		need -= look->length();
		prev = look;
		look = look->_nextblk;
	}
	if(look && (look->length() > need)) { // if we were successful (we need more than rd_offset for sure...)
		if(look->length() - need - size > 0) {
			if(need > 0) {
				ALLOC::memmove(look->_rd_ptr + need, look->_rd_ptr + need + size, look->length() - size - need); // shift the remaining memory by the amount we want to get rid of...
				look->_wr_ptr -= size; // shift the write pointer back (to change the length - and ignore the space we shifted left)
			} else {
				look->_rd_ptr += size;
			}
		} else if(look->length() - need - size == 0) { // if it was exactly the same size as a block
			if(need > 0) {
				look->_wr_ptr = look->_rd_ptr + need; // else just save the 'need'
			} else {
				if(prev) prev->_nextblk = look->_nextblk;
				if(head == look) head = look->_nextblk; // if releasing the first block, then make head the next
				look->_nextblk = NULL;
				look->release();
			}
		} else {
			BufBlk *front = NULL; // front of the blocks to get rid of...
			prev = look;
			int frontsz = look->length() - need;
			if(need > 0) {
				look->_wr_ptr -= frontsz; // shift the write pointer back (eats bytes to right)
				front = look->_nextblk;
			} else
				front = look;
			size -= frontsz;
			look = look->_nextblk;
			BufBlk *back = front;
			while(look && size > 0){
				if(size < look->length()) {
					look->_rd_ptr += size; // eat the last bytes
					break;
				} else {
					size -= look->length();
				}
				back = look;
				look = look->_nextblk;
			}

			if(front) {
				if(front == head)
					head = look;  // fix the head if we are eating the blocks at the head
//				if(prev != look->_nextblk)
				if(back != front) {
					back->_nextblk = NULL; // disconnect block
					prev->_nextblk = look; // hook chain past the part we are dropping
//					TW_DEBUG("releasing front.",NULL);
					front->release();
				} else {
					if(prev != front) { // releasing one block
						if(look == front && size) {} // release nothing if something was remaining and there was only one block
						else {  // otherwise...
						front->_nextblk = NULL;
						prev->_nextblk = look; // hook chain past the part we are dropping
						front->release();
						}
					}
				}
			}
		}
	} else
		if(need > 0) TW_DEBUG_L("Failure (2) at eatBytes: need %d\n",need);
	return head;
}

/**
 * Create an iterator for the tw_bufblk. This lets you walk through
 * the buffer, from rd_ptr to wr_ptr.
 * NOTE: in order to walk through the tw_bufblk, the iter will
 * duplicate() the tw_bufblk. The tw_bufblkIter's destructor will
 * release() the block. The Iter will also walk the entire chain,
 * including any nexblk() blocks.
 * @param b at tw_bufblk
 * @see tw_bufblk
 */
template <class ALLOC>
BufBlkIter<ALLOC>::BufBlkIter(BufBlk<ALLOC> &b) :
_start( b.duplicate() ),
_cur( NULL ),
_rd_offset( 0 ) {
	_cur = _start;
}

/**
 *  walks through the attached tw_bufblk chain, returing the next chunk of <= size max.
 *  If max == -1, then the largest contigous chunk possible is returned.
 * @param start the pointer referenced is set to the start address of the chunk
 * @param size this value is set to the size of the chunk
 * @param max [in] the max size to give the caller. If -1 the chunk returned is max possible
 * @return true if a valid chunk is available, false if the iter is at the end, or has no tw_bufblk attached
 */
template <class ALLOC>
bool BufBlkIter<ALLOC>::getNextChunk( char *&start, int &size, int max ) {
	if (_cur) {
		if (max == -1) {
//			printf("[1]");
			size = _cur->length() - _rd_offset;
			start = _cur->rd_ptr() + _rd_offset;
			_rd_offset = 0;
			_cur = _cur->nexblk();
			return true;
		} else {
			if (max >= (_cur->length() - _rd_offset)) {
//				printf("[A]");
				size = _cur->length() - _rd_offset;
				start = _cur->rd_ptr() + _rd_offset;
				_rd_offset = 0;
				_cur = _cur->nexblk();
			} else {
//				printf("[B]");
				size = max;
				start = _cur->rd_ptr() + _rd_offset;
				_rd_offset += max;
				if(_rd_offset >= _cur->length()) // if at end, goto next block
					_cur = _cur->nexblk();
			}
			return true;
		}
	} else
		return false;
}

/**
 *  walks through the attached tw_bufblk chain, returing the next chunk of <= size max.
 *  If max == -1, then the largest contigous chunk possible is returned. This function
 *  does not move the Iter forward, however. Another call will return the exact same data...
 * @param start the pointer referenced is set to the start address of the chunk
 * @param size this value is set to the size of the chunk
 * @param max [in] the max size to give the caller. If -1 the chunk returned is max possible
 * @return true if a valid chunk is available, false if the iter is at the end, or has no tw_bufblk attached
 */
template <class ALLOC>
bool BufBlkIter<ALLOC>::previewNextChunk( char *&start, int &size, int max ) {
	if (_cur) {
		if (max == -1) {
			size = _cur->length() - _rd_offset;
			start = _cur->rd_ptr() + _rd_offset;
			//		_rd_offset = 0;
			//		_cur = _cur->nexblk();
			return true;
		} else {
			if (max > (_cur->length() - _rd_offset)) {
				size = _cur->length() - _rd_offset;
				start = _cur->rd_ptr() + _rd_offset;
				//			_rd_offset = 0;
				//			_cur = _cur->nexblk();
			} else {
				size = max;
				//			_rd_offset += max;
				start = _cur->rd_ptr() + _rd_offset;
			}
			return true;
		}
	} else
		return false;
}

/** Attempts to copy [space] of data in the chain, to address [t].
 * Will copy as much as possible, and will span multiple blocks if necessary.
 * This does move the Iter.
 * @param t destination of copy
 * @param space amount of space in bytes destination has
 * @param copied [out] the amount of bytes the function copies. This is updated after the copy occurs.
 * @return true if data copied, false if the Iter has no more data
 */
template <class ALLOC>
bool BufBlkIter<ALLOC>::copyNextChunks(char *t, int space, int &copied ) {
	char *walk = t;
//	int s = 0;
	int remain = space;
	if(!_cur)
		return false;
	while((remain > 0) && _cur) {
		if((_cur->length() - _rd_offset) <= remain) { // if space in target exists for everything left in this block...
#ifdef _DEBUG_TW_BUFBLK
			TW_DEBUG("memcpy - %x, %x, %d\n", walk, _cur->rd_ptr() + _rd_offset, (_cur->length() - _rd_offset));
#endif
			ALLOC::memcpy( walk, _cur->rd_ptr() + _rd_offset, (_cur->length() - _rd_offset) );
			remain -= (_cur->length() - _rd_offset);
			walk += (_cur->length() - _rd_offset);
			_rd_offset = 0;
			_cur = _cur->nexblk();
		} else {      // otherwise copy just the portion;
#ifdef _DEBUG_TW_BUFBLK
			TW_DEBUG("memcpy2 - %x, %x, %d\n", walk, _cur->rd_ptr() + _rd_offset, remain );
#endif
			ALLOC::memcpy( walk, _cur->rd_ptr() + _rd_offset, remain );
			_rd_offset += remain;
//			_rd_offset = 0;
//			_cur = _cur->nexblk();
			walk += remain;
//			remain -= (_cur->length() - _rd_offset);
			remain = 0;
		}
	}
	copied = space - remain;
	return true;
}



/**
 * rewinds the iter to the start of the attached tw_bufblk block chain.
 */
template <class ALLOC>
void BufBlkIter<ALLOC>::rewind() {
	_cur = _start;
	_rd_offset = 0;
}

template <class ALLOC>
BufBlkIter<ALLOC>::~BufBlkIter() {
	if(_start)
		_start->release();
}

























#endif /* TW_BUFBLK_H_ */
