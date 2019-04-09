// WigWag LLC
// (c) 2010
// tw_bufblk.cpp
// Author: ed
#include <TW/tw_log.h>
#include <malloc.h>
#include <string.h>
#include <pthread.h>
#include <limits.h>
#include <stdio.h>

#include <string>
#include <sstream>
#include <iostream>


#include <TW/tw_bufblk.h>
//#define _DEBUG_TW_BUFBLK 1


// THIS FILE IS DEPRECATED

#ifdef _DONT_DEFINE_ME_

using namespace TWlib;
using namespace std;

MemBlk::MemBlk(int size) :
	_ref_count( 1 ),
	_base( 0 ),
	_size( 0 ),
	_manage( true ),
	_proper( true )
{
	_base = ::malloc( size );
	if(_base) {
		_size = size;
	} else
		_proper = false;
	pthread_mutex_init( &_refMutex, NULL );
}

MemBlk::MemBlk(void *data, int size, bool manage ) :
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
int MemBlk::getRefCount() {
	int ret = 0;
	pthread_mutex_lock(&_refMutex);
	ret = _ref_count;
	pthread_mutex_unlock(&_refMutex);
	return ret;
}

/**
 * tells tw_memblk that another tw_bufblk is referencing it, _ref_count++
 */
void MemBlk::checkout() {
	pthread_mutex_lock(&_refMutex);
	_ref_count++;
	pthread_mutex_unlock(&_refMutex);
}

void MemBlk::release() {             // decreases the ref count
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
			::free( _base ); // free the memory block
		delete this;         // good bye...
	}
}

int MemBlk::size() {                 // report block size
	return _size;
}

void *MemBlk::base() {               // report base of block
	return _base;
}

int MemBlk::_concatHexDump(std::ostringstream &outs, char *rdptr, char *wrtptr, int max) {
	int left = max;
	if(!max)
		left = INT_MAX;
	char *walk = (char *) _base;
	int c = 0;
	outs << "[";
	outs.setf(std::ios_base::hex,std::ios_base::basefield);
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
		outs << (int) *walk; // send as in so its hex
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

std::string &MemBlk::hexDump(std::string &out) {
	std::ostringstream outs;
	out.clear();
	outs.str(out);
	_concatHexDump(outs,0,0,0);
	out = outs.str();
	return out;
}


/** Default constructor. Object unusuable unless assigned to another.
 */
BufBlk::BufBlk(  ) :
_blk( NULL ),
_wr_ptr( NULL ),
_rd_ptr( NULL ),
_nextblk( NULL )
{ }

/** internal constructor... dups the tw_memblk.
 */
BufBlk::BufBlk( MemBlk *m ) :
_blk( m ),
_wr_ptr( NULL ),
_rd_ptr( NULL ),
_nextblk( NULL )
{
	_blk->checkout();
	_wr_ptr = (char *) _blk->_base; // set pointers at base
	_rd_ptr = (char *) _blk->_base;
}


/** Copy constructor.
 * Reference count on the underlying tw_memblk is incremented 1.
 *
 * @param o
 * @return
 */
BufBlk::BufBlk( BufBlk &o ) :
_blk( o._blk ),
_wr_ptr( o._wr_ptr ),
_rd_ptr( o._rd_ptr ),
_nextblk( NULL )
{
	assignFromBlk( o );
}

/** internal only - this is used when the tw_bufblk is duplicate()'d..
 * assumes all internal values are empty.
 * copies all data from [o]. copies all data, duplicates entire chain
 * @param o tw_bufblk to get data from.
 */
void BufBlk::assignFromBlk( const BufBlk &o ) {
	if(o._blk)
		o._blk->checkout(); // increase ref count on memblk
	_blk = o._blk;
	_nextblk = NULL;
	_rd_ptr = o._rd_ptr;
	_wr_ptr = o._wr_ptr;
	BufBlk *walko = o._nextblk;
	BufBlk *walk = this;
	while(walko) { // copy the chain... (non-recursive more efficient)
		walk->_nextblk = new BufBlk( walko->_blk );
		walk->_nextblk->_rd_ptr = walko->_rd_ptr;
		walk->_nextblk->_wr_ptr = walko->_wr_ptr;
		walk = walk->_nextblk;
		walko = walko->_nextblk;
 	}
}


/**
 * Assignement operator.
 * Reference count on the underlying tw_memblk is incremented 1.
 * @param o
 * @return
 */
BufBlk& BufBlk::operator=(const BufBlk &o) {
	if(_nextblk)
		_nextblk->release(); // release next block also
	if(_blk)
		_blk->release(); // release underlying memblk

	assignFromBlk( o );
}


int BufBlk::getRefCount() {
	if(_blk)
		return _blk->getRefCount();
	else
		return 0;
}

BufBlk *BufBlk::duplicate() {
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
BufBlk::BufBlk( void *data, int size, bool manage ) :
_blk( NULL ),
_rd_ptr( NULL ),
_wr_ptr( NULL ),
_nextblk( NULL )
{  // def: manage = true
	_blk = new MemBlk( data, size, manage );
	_rd_ptr = (char *) _blk->_base;
	_wr_ptr = (char *) _blk->_base;
}

/**
 * Vanilla constructor, create an underlying tw_memblk for the amount size. Reference count starts at 1.
 * When release() is called, the ref count decrements, and this object deletes itself.
 * On ref count 0, the underlying memory block deletes itself.
 * @param size in bytes of memory required
 */
BufBlk::BufBlk( int size ) :
_blk( NULL ),
_rd_ptr( NULL ),
_wr_ptr( NULL ),
_nextblk( NULL )
{  // def: manage = true
	_blk = new MemBlk( size );
	_rd_ptr = (char *) _blk->_base;
	_wr_ptr = (char *) _blk->_base;
}

/**
 * destructor also release next block, and underlying block.
 * Do not call ->release() and the destructor;
 */
BufBlk::~BufBlk() {
	if (_nextblk)
		_nextblk->release(); // release next block also
	if (_blk)
		_blk->release(); // release underlying memblk
}

/**
 *  releases the underlying memory block - meaning reference count decreases.
 *  This function deletes this object - the object is no longer valid after calling it.
 *  calling release() also releases any chained blocks.
 *  Warning: you can't call this if you are on the stack.
 */
void BufBlk::release() {
	if(_nextblk)
		_nextblk->release(); // release next block also
	if(_blk)
		_blk->release(); // release underlying memblk
	_nextblk = NULL;
	_blk = NULL;
	delete this;     // goodbye
}
/**
 * Returns the next chained tw_bufblk.
 * @return a pointer to the next chained tw_bufblk
 */
BufBlk *BufBlk::nexblk() {
	return _nextblk;
}

/**
 * Assign the block to follow this block. This next block will be chained to this one, and will release() when this one does.
 * DOES NOT increase reference count.
 * @param b BufBlk pointer
 */
void BufBlk::setNexblk( BufBlk *b ) {
	_nextblk = b;
}

/**
 * Adds the 'b' BufBlk to end of chain of blocks.
 * DOES NOT increase reference count.
 * @param b BufBlk pointer
 */
void BufBlk::addToEnd( BufBlk *b ) {
	BufBlk *look = this;
	while(look->_nextblk) {
		look=look->_nextblk;
	}
	look->_nextblk = b;
}

/** Copy data into the buffer. This moves the rd_ptr 'size' forward (or less, if not enough space)
 *
 * @param src base pointer of data
 * @param size in bytes to copy
 * @return the amount of data copied in bytes
 */
int BufBlk::copyFrom( void *src, int size ) {
	int s = size;
	if(BufBlk::freespace() < size)
		s = BufBlk::freespace();
	::memcpy(_wr_ptr, src, s);
	_wr_ptr = _wr_ptr + s;
	return s;
}

/**
 * Makes rd_ptr = base, and wr_ptr = base.
 */
void BufBlk::reset() {
	_rd_ptr = (char *) _blk->base();
	_wr_ptr = (char *) _blk->base();
}

bool BufBlk::isValid() {
	if(_blk) return true;
	else return false;
}

/**
 * sets rd_ptr to base() + offset [bytes]
 * @param offset bytes from the base
 */
void BufBlk::rewind(int offset) {
	_rd_ptr = ((char *) _blk->base()) + offset;
}
/**
 * wr_ptr - rd_ptr
 * @return
 */
int BufBlk::length() {
	return (int) ((unsigned long) _wr_ptr - (unsigned long) _rd_ptr);
}

int BufBlk::total_length() {
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
int BufBlk::capacity() {
	return _blk->size();
}

int BufBlk::total_capacity() {
	BufBlk *look = this;
	int ret = 0;
	while(look) {
		ret+=look->capacity();
		look=look->_nextblk;
	}
	return ret;
}


/**
 * dumps the entire chain of memory blocks, as a list of hexadecimal number (per byte)
 * @param in string object the dump will be placed in
 * @param max number of chars to put in string (0 is no max, dump everything in)
 */
std::string &BufBlk::hexDump(std::string &out, int max) {
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
int BufBlk::freespace() {
	return (int) _blk->size() - ((unsigned long) _wr_ptr - (unsigned long) _blk->base());
}

/**
 * @return the base of the underlying memory block
 */
void *BufBlk::base() {
	return _blk->base();
}

char *BufBlk::rd_ptr() {
	return _rd_ptr;
}

char *BufBlk::wr_ptr() {
	return _wr_ptr;
}

void BufBlk::inc_rd_ptr( int x ) {
	_rd_ptr += x;
}

void BufBlk::inc_wr_ptr( int x ) {
	_wr_ptr += x;
}


/**
 * @param p - rd_ptr is set to p
 */
void BufBlk::rd_ptr(char *p) {
	_rd_ptr = p;
}
/**
 * @param p - wr_ptr is set to p
 */
void BufBlk::wr_ptr(char *p) {
	_wr_ptr = p;
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
bool BufBlk::getContigBlock( BufBlk *&outblk, int rd_offset, int size ) {
	bool ret = false;
	// find the block where rd_offset starts
	int need = rd_offset;
	BufBlk *look = this;
	while(need >= 0) {
		if(look->length() > need)
			break; // look is the block where we begin.
		need -= look->length();
		look = look->_nextblk;
	}
	if(look->length() > need) { // if we were successful
		if(look->length() - need - size >= 0) {
			// cool - we can just duplicate that block, and return it
			outblk = new BufBlk(look->_blk);
			outblk->_rd_ptr = look->_rd_ptr + rd_offset;
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
	}
	return ret;
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
BufBlkIter::BufBlkIter(BufBlk &b) :
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
bool BufBlkIter::getNextChunk( char *&start, int &size, int max ) {
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
bool BufBlkIter::previewNextChunk( char *&start, int &size, int max ) {
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
bool BufBlkIter::copyNextChunks(char *t, int space, int &copied ) {
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
			::memcpy( walk, _cur->rd_ptr() + _rd_offset, (_cur->length() - _rd_offset) );
			remain -= (_cur->length() - _rd_offset);
			walk += (_cur->length() - _rd_offset);
			_rd_offset = 0;
			_cur = _cur->nexblk();
		} else {      // otherwise copy just the portion;
#ifdef _DEBUG_TW_BUFBLK
			TW_DEBUG("memcpy2 - %x, %x, %d\n", walk, _cur->rd_ptr() + _rd_offset, remain );
#endif
			::memcpy( walk, _cur->rd_ptr() + _rd_offset, remain );
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
void BufBlkIter::rewind() {
	_cur = _start;
	_rd_offset = 0;
}

BufBlkIter::~BufBlkIter() {
	if(_start)
		_start->release();
}

#endif


