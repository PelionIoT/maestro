/*
 *
 * WORK IN PROGRESS
 *
 * tw_disktable.h
 *
 *  Created on: Jan 19, 2012
 *      Author: ed
 * (c) 2012, WigWag LLC
 */


#ifndef TW_SPARSEHASH_H_
#define TW_SPARSEHASH_H_

#include <google/sparse_hash_map>

#include <TW/tw_alloc.h>
#include <TW/tw_fifo.h>
#include <TW/tw_stack.h>
#include <TW/tw_hashes.h>
#include <TW/tw_hashcommon.h>

#include <ext/hash_map>



#ifndef TW_DISKTABLE_H_
#define TW_DISKTABLE_H_




/**
 *
 * A disk based hash table for large amounts of data

 MUTEX is a ACE_Thread_Mutex or ACE_Null_Mutex or similar
 DATA must implement:
 copy constructor
 assignment (operator =)
 default constructor
 destructor (seems to not deallocate right without)

 KEY must implement
 specialize TWlib::tw_hash<KEY *> (if does not exist in the headers already)
 EQFUNC: implement the operator(KEY *) for eqstr struct (see above_...
 copy constructor
 destructor

 ALLOC Allocator style class for allocating DATA objects as they are pulled off
 the disk
 *
 */
template<typename KEY, typename DATA, typename EQFUNC, typename ALLOC>
// typename MUTEX,
class TWDiskHash  {
	TWDiskHash();
	bool open(int fides, int offset );
	bool close();
};



#endif /* TW_DISKTABLE_H_ */
