/*
 * tw_mmanager.h
 *
 *  Created on: Jan 19, 2012
 *      Author: ed
 * (c) 2012, WigWag LLC
 */

#ifndef TW_MMANAGER_H_
#define TW_MMANAGER_H_


template <class ALLOC>
class TWMemInfo {
public:
	bool inuse;
	MemBlk<ALLOC> *_blk;
	TWMemInfo() : _blk(NULL), inuse(true) {}
	TWMemInfo(const TWMemInfo &o) : _blk(o._blk), inuse(o.inuse) {}
	TWMemInfo &operator=(const TWMemInfo &o) { _blk = o._blk; inuse = o.inuse; return *this; }
};


/**
 * TWMemManager is a non-forceful memory manager of MemBlk's.
 * It will release a MemBlk if it is not in-use by anyone else,
 * and the TWMemManager is near its maximum allowed memory use.
 *
 * MemBlk's MUST be created with TWMemManager for this to work.
 */
template <class ALLOC>
class TWMemManager {
protected:

public:
	TWMemManager( int maxmem ) {}
	MemBlk<ALLOC> *newMemBlock( int size );

};



#endif /* TW_MMANAGER_H_ */
