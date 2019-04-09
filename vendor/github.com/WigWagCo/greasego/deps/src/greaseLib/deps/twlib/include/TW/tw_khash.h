/*
 * tw_khash.h
 *
 *  Created on: Mar 1, 2012
 *      Author: ed
 * (c) 2012, WigWag LLC
 *
 *

LICENSE

Some portions of code from klib, under this license:
The MIT License

   Copyright (c) 2008, 2009, 2011 by Attractive Chaos <attractor@live.co.uk>

   Permission is hereby granted, free of charge, to any person obtaining
   a copy of this software and associated documentation files (the
   "Software"), to deal in the Software without restriction, including
   without limitation the rights to use, copy, modify, merge, publish,
   distribute, sublicense, and/or sell copies of the Software, and to
   permit persons to whom the Software is furnished to do so, subject to
   the following conditions:

   The above copyright notice and this permission notice shall be
   included in all copies or substantial portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
   EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
   MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
   NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
   BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
   ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
   SOFTWARE.

 */

#ifndef TW_KHASH_H_
#define TW_KHASH_H_

#include <TW/tw_alloc.h>
#include <TW/tw_fifo.h>
#include <TW/tw_stack.h>
#include <TW/tw_hashes.h>
#include <TW/tw_hashcommon.h>

#include <TW/khash.h>


#include <memory>
#include <new>


/**
 *
 * wraps the kash macros @ http://attractivechaos.awardspace.com/khash.h.html
 * http://attractivechaos.wordpress.com/2008/09/02/implementing-generic-hash-library-in-c/
 * http://attractivechaos.wordpress.com/2008/08/28/comparison-of-hash-table-libraries/
 * Always update with latests code at: https://github.com/attractivechaos/klib/blob/master/test/khash_keith.c


 TW_KHash_XXX manages the memory of the data it holds. When the Hash is deleted, it will delete data (and keys).

 MUTEX is a TW_Mutex or TW_NoMutex or similar
 DATA must implement:
 copy constructor
 assignment (operator =)
 default constructor
 destructor (seems to not deallocate right without)

 KEY must implement
 specialize TWlib::tw_hash<const KEY *> (if does not exist already, or convert to size_t inherently)
 EQFUNC: implement the operator(KEY *) for eqstr struct
 copy constructor
 destructor
 assignment (operator =)

*/

namespace TWlib {


// uses a 32 bit integer as core key, with the kash.h hashtable macros.
// your tw_hash specialization must return a 32 bit int.
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
class TW_KHash_32  {
	public:
		// this constructor availble for drop in compatibility with TWDensehash
		TW_KHash_32(KEY &deletekey, KEY &emptykey, ALLOC *alloc = NULL, int items=0);
		// normal constructor
		TW_KHash_32(ALLOC *alloc = NULL, int items=0);
	//	TW_KHash_32(TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC> &other);
	//	TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC> &operator=(const TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>& rhs);
		~TW_KHash_32();
		bool addReplace( const KEY& key, DATA& dat );
		bool addReplace( const KEY& key, DATA& dat, DATA& olddat );
		bool addNoreplace( const KEY& key, DATA& dat );
		DATA *addReplaceNew( const KEY& key );
		DATA *addNoreplaceNew( const KEY& key );
		bool remove( const KEY& key );
		bool remove( const KEY& key, DATA& fill );
		bool find( const KEY& key, DATA& fill );
		DATA *find( const KEY& key );
		DATA *findOrNew( const KEY& key );
		bool removeAll();
		int size();

//	#ifdef _USE_GOOGLE_
//		typedef typename dense_hash_map<KEY *, DATA *, tw_hash<KEY *>, EQFUNC>::iterator internal_zhashiterator;
//	#else
//	//	typedef typename __gnu_cxx::hash_map<KEY *, DATA *, __gnu_cxx::hash<KEY *>, EQFUNC>::iterator internal_zhashiterator;
//	#endif


		void releaseIter();

//	#ifdef _USE_GOOGLE_
//		dense_hash_map<KEY *, DATA *, tw_hash<KEY *>, EQFUNC> values;
//	#else
//	//	__gnu_cxx::hash_map<KEY *, DATA *, __gnu_cxx::hash<KEY *>, EQFUNC> values;
//	#endif
		ALLOC *getAllocator() { return _alloc; }
	protected:
		void init(ALLOC *alloc, int items);
//		TWlib::tw_hash<KEY *> hasher;


		// KHASH_MAP_INIT_INT( 32, DATA );

		// KHASH_INIT(A, uint32_t, khval_t, 1, kh_int_hash_func, kh_int_hash_equal)

		//#define KHASH_INIT(name, khkey_t, khval_t, kh_is_map, __hash_func, __hash_equal)



		// why this? KHASHMAP (below) does not hold the key - it holds the tw_hash of the key, so this
		// holds key and value
		class Pair {
		public:
			KEY key;
			DATA val;
			Pair() : key(), val() {}
			Pair(const KEY &src) : key(src), val() {}
			Pair(const KEY &src, DATA &dat) : key(src), val(dat) {}
			Pair(Pair &o) : key(o.key), val(o.val) {}
			Pair &operator=(const Pair &o) {
				if(this != &o) { // no point in copying self
					key = o.key;
					val = o.val;
				}
				return *this;
			}
			DATA *newDATA() {
				val.~DATA();
				new (&val) DATA();
				return &val;
			}
			~Pair() {} // explicit destructor
		};


		static inline khint32_t __hash_func( const KEY &key ) {
			TWlib::tw_hash<KEY *> hash;
// GCC ONLY!!            On 64-bit platforms, size_t is 64 bits wide... so we need to get this down to 32 bits
#if __x86_64__
/* 64-bit */
			return kh_int64_hash_func(hash.operator ()(const_cast<KEY *>(&key)));
#else
			return hash.operator ()(const_cast<KEY *>(&key));
#endif
		}

//		#define khkey_t uint32_t
		// hkval_t is DATA
		#define kh_is_map 1
//		#define __hash_func kh_int_hash_func
//		#define __hash_equal kh_int_hash_equal

		static inline bool __hash_equal( KEY &l, const KEY &r ) {
			EQFUNC eq;
			return eq.operator ()(&l,&r);
		}

//		typedef Pair* khval_t;
		typedef DATA* khval_t;
		typedef KEY khkey_t;
		    typedef struct {
		        khint_t n_buckets, size, n_occupied, upper_bound;
		        uint32_t *flags;
		        khkey_t *keys;
		        khval_t *vals;
		    } kh_A_t;


		    static inline kh_A_t *kh_init_A() {
		        return (kh_A_t*)ALLOC::calloc(1, sizeof(kh_A_t));
		    }
			static inline void kh_clear_A(kh_A_t *h)
			{
				if (h && h->flags) {
					ALLOC::memset(h->flags, 0xaa, __ac_fsize(h->n_buckets) * sizeof(khint32_t));
					h->size = h->n_occupied = 0;
				}
			}
		    static inline void kh_destroy_A(kh_A_t *h)
		    {
		        if (h) {
		        	ALLOC::free(h->keys); ALLOC::free(h->flags);
		        	ALLOC::free(h->vals);
		        	ALLOC::free(h);
		        }
		    }
			static inline khint_t kh_get_A(const kh_A_t *h, const KEY &key) // this dbl const crap is b/c of C++'s weird issue with const pointer vs const values
			{
				if (h->n_buckets) {
					khint_t inc, k, i, last, mask;
					mask = h->n_buckets - 1;
					k = __hash_func(key); i = k & mask;
					inc = __ac_inc(k, mask); last = i; /* inc==1 for linear probing */
					while (!__ac_isempty(h->flags, i) && (__ac_isdel(h->flags, i) || !__hash_equal(h->keys[i], key))) {
						i = (i + inc) & mask;
						if (i == last) return h->n_buckets;
					}
					return __ac_iseither(h->flags, i)? h->n_buckets : i;
				} else return 0;
			}

			static inline void kh_resize_A(kh_A_t *h, khint_t new_n_buckets)
			{ /* This function uses 0.25*n_bucktes bytes of working space instead of [sizeof(key_t+val_t)+.25]*n_buckets. */
				__TW_HASH_DEBUGL(" -- Resize \n",NULL);
				khint32_t *new_flags = 0;
				khint_t j = 1;
				{
					kroundup32(new_n_buckets);
					if (new_n_buckets < 4) new_n_buckets = 4;
					if (h->size >= (khint_t)(new_n_buckets * __ac_HASH_UPPER + 0.5)) j = 0;	/* requested size is too small */
					else { /* hash table size to be changed (shrink or expand); rehash */
						new_flags = (khint32_t*)ALLOC::malloc(__ac_fsize(new_n_buckets) * sizeof(khint32_t));
						memset(new_flags, 0xaa, __ac_fsize(new_n_buckets) * sizeof(khint32_t));
						if (h->n_buckets < new_n_buckets) {	/* expand */
							h->keys = (khkey_t*)ALLOC::realloc(h->keys, new_n_buckets * sizeof(khkey_t));
							if (kh_is_map) h->vals = (khval_t*)ALLOC::realloc(h->vals, new_n_buckets * sizeof(khval_t));
						} /* otherwise shrink */
					}
				}
				if (j) { /* rehashing is needed */
					__TW_HASH_DEBUGL(" -- Rehash: %d\n", new_n_buckets);
					for (j = 0; j != h->n_buckets; ++j) {
						if (__ac_iseither(h->flags, j) == 0) {
							khkey_t key = h->keys[j];
							khval_t val;
							khint_t new_mask;
							new_mask = new_n_buckets - 1;
							if (kh_is_map) val = h->vals[j];
							__ac_set_isdel_true(h->flags, j);
							while (1) { /* kick-out process; sort of like in Cuckoo hashing */
								khint_t inc, k, i;
								k = __hash_func(key);
								i = k & new_mask;
								inc = __ac_inc(k, new_mask);
								while (!__ac_isempty(new_flags, i)) i = (i + inc) & new_mask;
								__ac_set_isempty_false(new_flags, i);
								if (i < h->n_buckets && __ac_iseither(h->flags, i) == 0) { /* kick out the existing element */
									{ khkey_t tmp = h->keys[i];
									h->keys[i] = key;
									key = tmp; }
									if (kh_is_map) { khval_t tmp = h->vals[i]; h->vals[i] = val; val = tmp; }
									__ac_set_isdel_true(h->flags, i); /* mark it as deleted in the old hash table */
								} else { /* write the element and jump out of the loop */
									//h->keys[i] = key;
									new (&(h->keys[i])) KEY(key);
									if (kh_is_map) h->vals[i] = val;
									break;
								}
							}
						}
					}
					if (h->n_buckets > new_n_buckets) { /* shrink the hash table */
						h->keys = (khkey_t*)ALLOC::realloc(h->keys, new_n_buckets * sizeof(khkey_t));
						if (kh_is_map) h->vals = (khval_t*)ALLOC::realloc(h->vals, new_n_buckets * sizeof(khval_t));
					}
					ALLOC::free(h->flags); /* free the working space */
					h->flags = new_flags;
					h->n_buckets = new_n_buckets;
					h->n_occupied = h->size;
					h->upper_bound = (khint_t)(h->n_buckets * __ac_HASH_UPPER + 0.5);
				}
			}

			static inline khint_t kh_put_A(kh_A_t *h, const KEY &key, int *ret)
			{
				khint_t x;
				if (h->n_occupied >= h->upper_bound) { /* update the hash table */
					if (h->n_buckets > (h->size<<1)) kh_resize_A(h, h->n_buckets - 1); /* clear "deleted" elements */
					else kh_resize_A(h, h->n_buckets + 1); /* expand the hash table */
				} /* TODO: to implement automatically shrinking; resize() already support shrinking */
				{
					khint_t inc, k, i, site, last, mask = h->n_buckets - 1;
					x = site = h->n_buckets; k = __hash_func(key); i = k & mask;
					if (__ac_isempty(h->flags, i)) x = i; /* for speed up */
					else {
						inc = __ac_inc(k, mask); last = i;
						while (!__ac_isempty(h->flags, i) && (__ac_isdel(h->flags, i) || !__hash_equal(h->keys[i], key))) {
							if (__ac_isdel(h->flags, i)) site = i;
							i = (i + inc) & mask;
							if (i == last) { x = site; break; }
						}
						if (x == h->n_buckets) {
							if (__ac_isempty(h->flags, i) && site != h->n_buckets) x = site;
							else x = i;
						}
					}
				}
				if (__ac_isempty(h->flags, x)) { /* not present at all */
//					h->keys[x] = key;
					new (&(h->keys[x])) KEY(key);   // to support objects correctly
					__ac_set_isboth_false(h->flags, x);
					++h->size; ++h->n_occupied;
					*ret = 1;
				} else if (__ac_isdel(h->flags, x)) { /* deleted */
//					h->keys[x] = key;
					new (&(h->keys[x])) KEY(key);
					__ac_set_isboth_false(h->flags, x);
					++h->size;
					*ret = 2;
				} else *ret = 0; /* Don't touch h->keys[x] if present and not deleted */
				return x;
			}

		    static inline void kh_del_A(kh_A_t *h, khint_t x)
		    {
		        if (x != h->n_buckets && !__ac_iseither(h->flags, x)) {
		            __ac_set_isdel_true(h->flags, x);
		            --h->size;
		        }
		    }



public:
			class HashIterator {
			public:
		//		HashIterator() { }
				HashIterator(TW_KHash_32 &map);
				KEY *key();
				DATA *data();
				void setData(DATA *);
				bool getNext();
				bool atEnd();
				void release();
		//		bool getPrev();
				friend class TW_KHash_32;

			protected:
				khiter_t _iter;
				TW_KHash_32 &_map;
			};

protected:

		// END KHASH
		#undef khkey_t
		// hkval_t is DATA
		#undef kh_is_map
		#undef __hash_func
		#undef __hash_equal

		kh_A_t *KHASHMAP;   // the map itself


		friend class HashIterator;
		void gotoStart(HashIterator &i);


		int _iterators_out;
		KEY *_emptykey;
		KEY *_deletedkey; // Google dense_hash_map needs to unique keys, never to be used. See: http://google-sparsehash.googlecode.com/svn/trunk/doc/dense_hash_map.html#new
		ALLOC *_alloc;

		// the internal map is actually a map of pointer - this class
		// has responsibility for memeory management. I do this b/c I dont trust the STL implementations
		MUTEX _lock;
};

} // end namespace

using namespace TWlib;

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::HashIterator(TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC> &map) :
_map( map ) {
	_map.gotoStart(*this);
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::data() {
	if(_iter != kh_end(_map.KHASHMAP)) {
		return kh_value(_map.KHASHMAP, _iter);
	} else {
		return NULL;
	}

}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::setData(DATA *d) {
	if(_iter != kh_end(_map.KHASHMAP)) {
//		_it->second = d;
//		DATA &oldd = kh_value(KHASHMAP, k);
//		oldd.~DATA();
//		kh_del_A(KHASHMAP, k);
		kh_value(_map.KHASHMAP, _iter) = d;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
KEY *TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::key() {
	if(_iter != kh_end(_map.KHASHMAP)) {
		return &(kh_key(_map.KHASHMAP, _iter));
	} else {
		return NULL;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::getNext() {
	if(_iter == kh_end(_map.KHASHMAP)) {
		return false;
	} else {
		_iter++;
		while(_iter != kh_end(_map.KHASHMAP) && !kh_exist(_map.KHASHMAP,_iter)) { // the iterator goes through empty buckets, so this deals with passing those up also
			_iter++;
		}
		if(_iter == kh_end(_map.KHASHMAP))
			return false;
		else
			return true;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::atEnd() {
	if(_iter == kh_end(_map.KHASHMAP)) {
		return true;
	} else {
		return false;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::release() {
	_map.releaseIter();
}




// /*template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
//bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::TW_KHash_32::HashIterator::getPrev() {
//	if(_it == _map.values.begin()) {
//		return false;
//	} else {
//		--_it;
//		if(_it == _map.values.begin())
//			return false;
//		else
//			return true;
//	}
//}*/

//template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
//TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::getIter() {
//	return HashIterator(*this);
//}


template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::gotoStart(HashIterator &i) {
	_lock.acquire();
	_iterators_out++;
//	i._it = values.begin();
	i._iter = kh_begin(KHASHMAP);
	while(i._iter != kh_end(KHASHMAP) && !kh_exist(KHASHMAP,i._iter)) { // the iterator goes through empty buckets, so this deals with passing those up also
		i._iter++;
	}

	_lock.release();
}

/*template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::gotoEnd(HashIterator &i) {
	_lock.acquire();
	_iterators_out++;
	i._it = values.end();
	_lock.release();
}*/

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::releaseIter() {
	_lock.acquire();
	if(_iterators_out > 0)
		_iterators_out--;
	_lock.release();
}


template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::TW_KHash_32( KEY &deletekey, KEY &emptykey, ALLOC *alloc, int items ) :
KHASHMAP(NULL), _iterators_out( 0 ), _lock()
{
	init(alloc,items);
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::init(ALLOC *alloc, int items) {
	if(!alloc)
		_alloc = ALLOC::getInstance();
	else
		_alloc = alloc;

	KHASHMAP = kh_init_A();

	if(items > 0)
		kh_resize_A(KHASHMAP,items);
}

// emptykey is required by sparse_hash_map
// ...this is a known key that will never be in the actual data set
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::TW_KHash_32( ALLOC *alloc, int items ) :
KHASHMAP(NULL), _iterators_out( 0 ), _lock()
{
// http://google-sparsehash.googlecode.com/svn/trunk/doc/dense_hash_map.html#6
//	if(items)
//		values.resize(items);
	init(alloc,items);
//	TW_NEW_WALLOC(_emptykey, KEY, KEY(emptykey), _alloc );
//	TW_NEW_WALLOC(_deletedkey, KEY, KEY(deletekey), _alloc );

//	ACE_NEW_MALLOC
//	(_emptykey,
//			(reinterpret_cast<KEY *>
//			(this->_alloc->malloc (sizeof (KEY)))),
//			(KEY) (emptykey) ); // copy the emptykey to our _emptykey - use our Allocator

//#ifdef _USE_GOOGLE_
//	values.set_empty_key(_emptykey);
//	values.set_deleted_key(_deletedkey);
//#endif
}

/*
TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::TW_KHash_32(TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC> &other) {
	removeAll();
	_iterators_out = other._iterators_out;
	_emptykey = other._emptykey;
}

TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC> &TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::operator=(const TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>& rhs) {

}
*/
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY, DATA, MUTEX, EQFUNC,ALLOC>::removeAll() {
	_lock.acquire();
//#ifdef _USE_GOOGLE_
//	values.set_deleted_key(_deletedkey);
//#endif
	int c = 0;
	if(KHASHMAP) {
		khiter_t k;
		for (k = kh_begin(KHASHMAP); k != kh_end(KHASHMAP); ++k)
			if (kh_exist(KHASHMAP, k)) {
				DATA *d = kh_value(KHASHMAP,k);
				if(d) TW_DELETE_WALLOC(d,DATA,_alloc);
				kh_key(KHASHMAP,k).~KEY(); // explicitly run destructor on key (key is stored in hash table array)
//				kh_del_A(KHASHMAP,k);
				c++;
			}
//		kh_destroy_A(KHASHMAP);
		kh_clear_A(KHASHMAP);
//		KHASHMAP = NULL;
	}

	_lock.release();
	__TW_HASH_DEBUG("deleted: %d records\n", c );

	return true;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
int TW_KHash_32<KEY, DATA, MUTEX, EQFUNC,ALLOC>::size() {
	if(KHASHMAP)
		return kh_size(KHASHMAP);
	else
		return 0;
	//	return (int) values.size();
}


template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
TW_KHash_32<KEY, DATA, MUTEX, EQFUNC, ALLOC>::~TW_KHash_32() {
	removeAll();
	kh_destroy_A(KHASHMAP);
}

// replace the data if it exists.
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addReplace( const KEY& key, DATA& dat, DATA& oldref ) {
	_lock.acquire();
	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k != kh_end(KHASHMAP)) { // have this key?
		DATA *old = kh_value(KHASHMAP, k);
		if(old) oldref = *old;
		TW_DELETE_WALLOC(old,DATA,_alloc); // out w/ the old, in with the new
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(dat),_alloc);
	} else {
		int r; // TODO - deal with collisions
		k = kh_put_A(KHASHMAP,key,&r);
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(dat),_alloc);
	}
	_lock.release();
	return true;
}

// replace the data if it exists.
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addReplace( const KEY& key, DATA& dat ) {
	_lock.acquire();
	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k != kh_end(KHASHMAP)) { // don't have it
		DATA *old = kh_value(KHASHMAP, k);
		TW_DELETE_WALLOC(old,DATA,_alloc); // out w/ the old, in with the new
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(dat),_alloc);
	} else {
		int r; // TODO - deal with collisions
		k = kh_put_A(KHASHMAP,key,&r);
		if(!r) __TW_HASH_DEBUGL("WARNING: collision\n",NULL);
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(dat),_alloc);
	}
	_lock.release();
	return true;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addReplaceNew( const KEY& key ) {
	_lock.acquire();
	DATA *ret = NULL;

	khiter_t k = kh_get_A(KHASHMAP,key);
	if(k != kh_end(KHASHMAP)) { // don't have it
		DATA *old = kh_value(KHASHMAP, k);
		TW_DELETE_WALLOC(old,DATA,_alloc); // out w/ the old, in with the new
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(),_alloc);
		ret = kh_value(KHASHMAP, k);
	} else {
		int r; // TODO - deal with collisions
		k = kh_put_A(KHASHMAP,key,&r);
		if(!r) __TW_HASH_DEBUGL("WARNING: collision\n",NULL);
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(),_alloc);
		ret = kh_value(KHASHMAP, k);
	}
	_lock.release();
	return ret;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addNoreplaceNew( const KEY& key ) {
	_lock.acquire();
	DATA *ret = NULL;

	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k == kh_end(KHASHMAP)) { // don't have it
		__TW_HASH_DEBUGL("add: hash: %d\n", k);

		int r; // TODO - deal with collisions
		k = kh_put_A(KHASHMAP,key,&r);
		if(!r) __TW_HASH_DEBUGL("WARNING: collision\n",NULL);
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(),_alloc);
		ret = kh_value(KHASHMAP, k);
	}
	_lock.release();
	return ret;

	}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::findOrNew( const KEY& key ) {
	_lock.acquire();
	DATA *ret = NULL;
//	uint32_t kval = hasher.operator ()(&key);
	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k != kh_end(KHASHMAP)) { // don't have it
		ret = kh_value(KHASHMAP, k);
	} else {
		int r; // TODO - deal with collisions
		k = kh_put_A(KHASHMAP,key,&r);
		if(!r) __TW_HASH_DEBUGL("WARNING: collision\n",NULL);
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(),_alloc);
		ret = kh_value(KHASHMAP, k);
	}
	_lock.release();
	return ret;

}


template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addNoreplace( const KEY& key, DATA& dat ) {
	DATA *d = NULL;
	bool ret = false;
	_lock.acquire();
	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k == kh_end(KHASHMAP)) { // don't have it
		int r; // TODO - deal with collisions
		k = kh_put_A(KHASHMAP,key,&r);
		if(!r) __TW_HASH_DEBUGL("WARNING: collision\n",NULL);
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),DATA,DATA(dat),_alloc);
		ret = true;
	}
	_lock.release();
	return ret;

}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::remove( const KEY& key, DATA& fill ) {
	DATA *d = NULL;
	bool ret = false;

	_lock.acquire();
	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k != kh_end(KHASHMAP)) { // don't have it
		DATA *d = kh_value(KHASHMAP, k);
		if(d) {
			fill = *d;
			TW_DELETE_WALLOC(d,DATA,this->_alloc);
		}
		kh_del_A(KHASHMAP, k);
		ret = true;
	}
	_lock.release();
	return ret;

}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::remove( const KEY& key ) {

//	DATA *d = NULL;
	bool ret = false;
//	uint32_t kval = hasher.operator ()(&key);
	_lock.acquire();
	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k != kh_end(KHASHMAP)) { // don't have it
		DATA *d = kh_value(KHASHMAP, k);
		if(d)
			TW_DELETE_WALLOC(d,DATA,this->_alloc);
		kh_del_A(KHASHMAP, k);
		ret = true;
	}
	_lock.release();
	return ret;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::find( const KEY& key, DATA& fill ) {
	bool ret = false;
//	TW_DEBUG_L("looking. %p\n",KHASHMAP);
	_lock.acquire();
	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k != kh_end(KHASHMAP)) { // have it?
//		TW_DEBUG_L("has it. %p\n",KHASHMAP);
		if(kh_value(KHASHMAP,k)) {
			fill = *(kh_value(KHASHMAP,k)); // uses DATA::operator=()
			ret = true;
		}
	}
	_lock.release();
	return ret;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TW_KHash_32<KEY,DATA,MUTEX,EQFUNC,ALLOC>::find( const KEY& key ) {

	DATA *ret = NULL;

	_lock.acquire();

	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k != kh_end(KHASHMAP)) { // don't have it
		ret = kh_value(KHASHMAP, k);
	}
	_lock.release();
	return ret;
}


#endif /* TW_KHASH_H_ */
