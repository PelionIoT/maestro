// WigWag LLC
// (c) 2010
// tw_densehash.h
// Author: ed
// May 23, 2010

#ifndef TW_DENSEHASH_H_
#define TW_DENSEHASH_H_

#include <google/dense_hash_map>


#include <TW/tw_alloc.h>
#include <TW/tw_fifo.h>
#include <TW/tw_stack.h>
#include <TW/tw_hashes.h>
#include <TW/tw_hashcommon.h>

#include <ext/hash_map>

using google::dense_hash_map;      // namespace where class lives by default
#define _USE_GOOGLE_

// specialize of the GNU hash function for ZStrings
namespace TWlib {

/**
 *
 * wraps the google sparse_hash_map -

 TWSparseHash manages the memory of the data it holds. When the Hash is deleted, it will delete data (and keys).

 MUTEX is a TW_Mutex or TW_NoMutex or similar
 DATA must implement:
 copy constructor
 assignment (operator =)
 default constructor
 destructor (seems to not deallocate right without)

 KEY must implement
 specialize TWlib::tw_hash<KEY *> (if does not exist already, or convert to size_t inherently)
 EQFUNC: implement the operator(KEY *) for eqstr struct
 copy constructor
 destructor


  Example eqstr:
  struct Cache_SEQ_eqstrP {
	  inline int operator() (const ZPK_SEQ *kt1,
	                  const ZPK_SEQ *kt2) const
	  {
	    return (*kt1 == *kt2);
	  }
  };


  Example tw_hash specialization:
  template<> struct TWlib::tw_hash<zdb::ZString> {
	inline size_t operator()(const Z_DWORD *v) const {
		return (size_t) *v;
	}
};
 */
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
class TWDenseHash  {
public:
	TWDenseHash(const KEY& deletekey, const KEY& emptykey, ALLOC *alloc = NULL, int items=0);
//	TWDenseHash(TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC> &other);
//	TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC> &operator=(const TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>& rhs);
	~TWDenseHash();
	bool addReplace( KEY& key, DATA& dat );
	bool addReplace( KEY& key, DATA& dat, DATA& olddat );
	bool addNoreplace( KEY& key, DATA& dat );
	DATA *addReplaceNew( KEY& key );
	DATA *addNoreplaceNew( KEY& key );
	bool remove( KEY& key );
	bool remove( KEY& key, DATA& fill );
	bool find( KEY& key, DATA& fill );
	DATA *find( KEY& key );
	DATA *findOrNew( KEY& key );
	bool removeAll();
	int size();

#ifdef _USE_GOOGLE_
	typedef typename dense_hash_map<KEY *, DATA *, tw_hash<KEY *>, EQFUNC>::iterator internal_zhashiterator;
#else
//	typedef typename __gnu_cxx::hash_map<KEY *, DATA *, __gnu_cxx::hash<KEY *>, EQFUNC>::iterator internal_zhashiterator;
#endif

	class HashIterator {
	public:
//		HashIterator() { }
		HashIterator(TWDenseHash &map);
		KEY *key();
		DATA *data();
		void setData(DATA *);
		bool getNext();
		bool atEnd();
		void release();
//		bool getPrev();
		friend class TWDenseHash;
	protected:
		TWDenseHash &_map;
		internal_zhashiterator _it;
//		~HashIterator();
	};

//	HashIterator getIter() { return HashIterator(*this); }
//	void gotoEnd(HashIterator &i);
	void releaseIter();

#ifdef _USE_GOOGLE_
	dense_hash_map<KEY *, DATA *, tw_hash<KEY *>, EQFUNC> values;
#else
//	__gnu_cxx::hash_map<KEY *, DATA *, __gnu_cxx::hash<KEY *>, EQFUNC> values;
#endif
	ALLOC *getAllocator() { return _alloc; }
protected:
	friend class HashIterator;
	void gotoStart(HashIterator &i);
	MUTEX _lock;
	// the internal map is actually a map of pointer - this class
	// has responsibility for memeory management. I do this b/c I dont trust the STL implementations


	int _iterators_out;
	KEY *_emptykey;
	KEY *_deletedkey; // Google dense_hash_map needs to unique keys, never to be used. See: http://google-sparsehash.googlecode.com/svn/trunk/doc/dense_hash_map.html#new
	ALLOC *_alloc;
};

} // end namespace

using namespace TWlib;

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::HashIterator(TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC> &map) :
_map( map ) {
	_map.gotoStart(*this);
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::data() {
	if(_it != _map.values.end()) {
		return _it->second;
	} else {
		return NULL;
	}

}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::setData(DATA *d) {
	if(_it != _map.values.end()) {
		_it->second = d;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
KEY *TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::key() {
	if(_it != _map.values.end()) {
		return _it->first;
	} else {
		return NULL;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::getNext() {
	if(_it == _map.values.end()) {
		return false;
	} else {
		_it++;
		if(_it == _map.values.end())
			return false;
		else
			return true;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::atEnd() {
	if(_it == _map.values.end()) {
		return true;
	} else
		return false;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator::release() {
	_map.releaseIter();
}


/*template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::TWDenseHash::HashIterator::getPrev() {
	if(_it == _map.values.begin()) {
		return false;
	} else {
		--_it;
		if(_it == _map.values.begin())
			return false;
		else
			return true;
	}
}*/

//template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
//TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::HashIterator TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::getIter() {
//	return HashIterator(*this);
//}


template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::gotoStart(HashIterator &i) {
	_lock.acquire();
	_iterators_out++;
	i._it = values.begin();
	_lock.release();
}

/*template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::gotoEnd(HashIterator &i) {
	_lock.acquire();
	_iterators_out++;
	i._it = values.end();
	_lock.release();
}*/

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
void TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::releaseIter() {
	_lock.acquire();
	if(_iterators_out > 0)
		_iterators_out--;
	_lock.release();
}

// emptykey is required by sparse_hash_map
// ...this is a known key that will never be in the actual data set
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::TWDenseHash(const KEY& deletekey, const KEY& emptykey, ALLOC *alloc, int items ) :
_iterators_out( 0 ),
_emptykey( NULL )
{
// http://google-sparsehash.googlecode.com/svn/trunk/doc/dense_hash_map.html#6
	if(items)
		values.resize(items);
	if(!alloc)
		_alloc = ALLOC::getInstance();
	else
		_alloc = alloc;


	TW_NEW_WALLOC(_emptykey, KEY, KEY(emptykey), _alloc );
	TW_NEW_WALLOC(_deletedkey, KEY, KEY(deletekey), _alloc );

//	ACE_NEW_MALLOC
//	(_emptykey,
//			(reinterpret_cast<KEY *>
//			(this->_alloc->malloc (sizeof (KEY)))),
//			(KEY) (emptykey) ); // copy the emptykey to our _emptykey - use our Allocator

#ifdef _USE_GOOGLE_
	values.set_empty_key(_emptykey);
	values.set_deleted_key(_deletedkey);
#endif
}

/*
TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::TWDenseHash(TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC> &other) {
	removeAll();
	_iterators_out = other._iterators_out;
	_emptykey = other._emptykey;
}

TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC> &TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::operator=(const TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>& rhs) {

}
*/
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY, DATA, MUTEX, EQFUNC,ALLOC>::removeAll() {
	_lock.acquire();
#ifdef _USE_GOOGLE_
	values.set_deleted_key(_deletedkey);
#endif
	//	int c = 0;
	KEY *k;
	internal_zhashiterator it = values.begin();
	while (it != values.end()) {

		if (it->first) // key
			k = it->first;
		else
			k=NULL;

		__TW_HASH_DEBUG("removing key: %x, val: %x\n",k,it->second);

		if (it->second) { // data
			TW_DELETE_WALLOC(it->second, DATA, this->_alloc);
		}

		values.erase(it); // this order is critical - you have to erase pair, while the key is still valid...
		if(k) // ...now delete key
			TW_DELETE_WALLOC( k, KEY, this->_alloc);

		it = values.begin();
//		it++;
	}
	values.resize(0);
	//  printf("rec count %d\n",values.size());
	values.erase(values.begin(), values.end());
	_lock.release();
	//  printf("deleted: %d records\n", c );
	return true;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
int TWDenseHash<KEY, DATA, MUTEX, EQFUNC,ALLOC>::size() {
	return (int) values.size();
}


template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
TWDenseHash<KEY, DATA, MUTEX, EQFUNC, ALLOC>::~TWDenseHash() {
	removeAll();
	if(_emptykey)
		TW_DELETE(_emptykey, KEY, ALLOC);
}

// replace the data if it exists.
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addReplace( KEY& key, DATA& dat, DATA& olddat ) {
	KEY *k = NULL;
	DATA *v = NULL;
	_lock.acquire();
	internal_zhashiterator it = values.find(&key);
	TW_NEW_WALLOC(v, DATA, DATA(dat), this->_alloc);
	if(!v) return false;
//--	ACE_NEW_MALLOC_RETURN
//--			(v,	(reinterpret_cast<DATA *>
//--				(this->_alloc->malloc (sizeof (DATA)))),
//--				(DATA) (dat), false ); // copy data

	if (it == values.end()) {
		TW_NEW_WALLOC(k, KEY, KEY(key), this->_alloc);
		if(!k) return false;

//--		ACE_NEW_MALLOC_RETURN
//--		(k,	(reinterpret_cast<KEY *>
//--			(this->_alloc->malloc (sizeof (KEY)))),
//--			(KEY) (key), false ); // copy key - if new key

		values[k] = v; // add pair
	} else {
		olddat = *it->second;
//		delete it->second; // delete old data
		TW_DELETE_WALLOC(it->second, DATA, this->_alloc);
//		ACE_DES_FREE( it->second, this->_alloc->free, DATA);
		it->second = v;    // put in new data
	}
	_lock.release();
	return true;
}

// replace the data if it exists.
template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addReplace( KEY& key, DATA& dat ) {
	KEY *k = NULL;
	DATA *v = NULL;
	_lock.acquire();
	internal_zhashiterator it = values.find(&key);
	TW_NEW_WALLOC(v, DATA, DATA(dat), this->_alloc);
	if(!v) return false;
//--	ACE_NEW_MALLOC_RETURN
//--			(v,	(reinterpret_cast<DATA *>
//--				(this->_alloc->malloc (sizeof (DATA)))),
//--				(DATA) (dat), false ); // copy data

	if (it == values.end()) {
		TW_NEW_WALLOC(k, KEY, KEY(key), this->_alloc);
		if(!k) return false;

//--		ACE_NEW_MALLOC_RETURN
//--		(k,	(reinterpret_cast<KEY *>
//--			(this->_alloc->malloc (sizeof (KEY)))),
//--			(KEY) (key), false ); // copy key - if new key

		values[k] = v; // add pair
	} else {
//		delete it->second; // delete old data
		TW_DELETE_WALLOC(it->second, DATA, this->_alloc);
//		ACE_DES_FREE( it->second, this->_alloc->free, DATA);
		it->second = v;    // put in new data
	}
	_lock.release();
	return true;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addReplaceNew( KEY& key ) {
	KEY *k = NULL;
	DATA *v = NULL;
//	bool iamempty = false;
	_lock.acquire();
	TW_NEW_WALLOC(v, DATA, DATA(), this->_alloc);
	if(!v) return false;
//--	ACE_NEW_MALLOC_RETURN
//--			(v,	(reinterpret_cast<DATA *>
//--				(this->_alloc->malloc (sizeof (DATA)))),
//--				(DATA) (), false ); // copy data
	internal_zhashiterator it;
//	if(values.empty())
//		iamempty = true;
//	else
		it = values.find(&key);
	if (it == values.end()) {
		TW_NEW_WALLOC(k, KEY, KEY(key), this->_alloc);
		if(!k) return false;
//--		ACE_NEW_MALLOC_RETURN
//--		(k,	(reinterpret_cast<KEY *>
//--			(this->_alloc->malloc (sizeof (KEY)))),
//--			(KEY) (key), false ); // copy key - if new key

		values[k] = v;
	} else {
		TW_DELETE_WALLOC(it->second, DATA, this->_alloc);
//--		ACE_DES_FREE( it->second, this->_alloc->free, DATA);
//		delete it->second; // delete old data
		it->second = v;    // put in new data
	}
	_lock.release();
	return v;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addNoreplaceNew( KEY& key ) {
	KEY *k = NULL;
	DATA *v = NULL;
//	bool iamempty = false;
	internal_zhashiterator it;
	_lock.acquire();
	it = values.find(&key);
	if(it == values.end()) {
		TW_NEW_WALLOC(v, DATA, DATA(), this->_alloc);
		if(!v) return NULL;
//--	ACE_NEW_MALLOC_RETURN
//--			(v,	(reinterpret_cast<DATA *>
//--				(this->_alloc->malloc (sizeof (DATA)))),
//--				(DATA) (), false ); // copy data
//	if(values.empty())
//		iamempty = true;
//	else
//	if (it == values.end()) {
		TW_NEW_WALLOC(k, KEY, KEY(key), this->_alloc);
		if(!k) {
			TW_DELETE_WALLOC(v,DATA,this->_alloc);
			return NULL;
		}
//--		ACE_NEW_MALLOC_RETURN
//--		(k,	(reinterpret_cast<KEY *>
//--			(this->_alloc->malloc (sizeof (KEY)))),
//--			(KEY) (key), false ); // copy key - if new key

		values[k] = v;
//	} else {
//		ACE_DES_FREE( it->second, this->_alloc->free, DATA);
//		delete it->second; // delete old data
//		it->second = v;    // put in new data
	}

	_lock.release();
	return v;
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::findOrNew( KEY& key ) {
	KEY *k = NULL;
	DATA *v = NULL;
//	bool iamempty = false;
	internal_zhashiterator it;
	_lock.acquire();
	it = values.find(&key);
	if(it == values.end()) {
		TW_NEW_WALLOC(v, DATA, DATA(), this->_alloc);
		if(!v) return false;
//--	ACE_NEW_MALLOC_RETURN
//--			(v,	(reinterpret_cast<DATA *>
//--				(this->_alloc->malloc (sizeof (DATA)))),
//--				(DATA) (), false ); // copy data
//	if(values.empty())
//		iamempty = true;
//	else
//	if (it == values.end()) {
		TW_NEW_WALLOC(k, KEY, KEY(key), this->_alloc);
		if(!k) return false;

//--		ACE_NEW_MALLOC_RETURN
//--		(k,	(reinterpret_cast<KEY *>
//--			(this->_alloc->malloc (sizeof (KEY)))),
//--			(KEY) (key), false ); // copy key - if new key

		values[k] = v;
	} else {
		v = it->second;
	}

	_lock.release();
	return v;
}


template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::addNoreplace( KEY& key, DATA& dat ) {
	KEY *k = NULL;
	DATA *v = NULL;
	_lock.acquire();
	internal_zhashiterator it = values.find(&key);
	if (it == values.end()) {
		TW_NEW_WALLOC(v, DATA, DATA(), this->_alloc);
		if(!v) return false;
//--		ACE_NEW_MALLOC_RETURN
//--				(v,	(reinterpret_cast<DATA *>
//--					(this->_alloc->malloc (sizeof (DATA)))),
//--					(DATA) (dat), false ); // copy data
		TW_NEW_WALLOC(k, KEY, KEY(key), this->_alloc);
		if(!k) return false;

//--		ACE_NEW_MALLOC_RETURN
//--		(k,	(reinterpret_cast<KEY *>
//--			(this->_alloc->malloc (sizeof (KEY)))),
//--			(KEY) (key), false ); // copy key - if new key

		values[k] = v;
		_lock.release();
		return true;
	} else {
		_lock.release();
		return false; // and return...
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::remove( KEY& key, DATA& fill ) {
	KEY *k = NULL;
	DATA *v = NULL;
	_lock.acquire();
	internal_zhashiterator it = values.find(&key);
	if (it == values.end()) {
		_lock.release();
		return false;
	} else {
		k = it->first;
		fill = *it->second; // requires assignement operator
		v = it->second;
//		values.erase(&key);
		values.erase(it);
		if(_iterators_out == 0) // if no iterators are out - then compact the table
			values.resize(0);
		_lock.release();
		__TW_HASH_DEBUG("removing key: %x, val: %x\n",k,v);
		TW_DELETE_WALLOC(k, KEY, this->_alloc);
		TW_DELETE_WALLOC(v, DATA, this->_alloc);
//--		ACE_DES_FREE( k, this->_alloc->free, KEY);
//--		ACE_DES_FREE( v, this->_alloc->free, DATA);
//		it->first = NULL;
//		it->second = NULL;
//		if(k) delete k;
//		if(v) delete v;
		return true;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::remove( KEY& key ) {
	KEY *k = NULL;
	DATA *v = NULL;
	_lock.acquire();
	internal_zhashiterator it = values.find(&key);
	if (it == values.end()) {
		_lock.release();
		return false;
	} else {
		k = it->first;
		v = it->second;
		values.erase(&key);
		_lock.release();
		TW_DELETE_WALLOC(k, KEY, this->_alloc);
		TW_DELETE_WALLOC(v, DATA, this->_alloc);
//--		ACE_DES_FREE( k, this->_alloc->free, KEY);
//--		ACE_DES_FREE( v, this->_alloc->free, DATA);
//		if(k) delete k;
//		if(v) delete v;
		return true;
	}

}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
bool TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::find( KEY& key, DATA& fill ) {
	DATA *v = NULL;
	internal_zhashiterator it = values.find(&key);
	if (it == values.end()) {
		_lock.release();
		return false;
	} else {
		v = it->second;
		_lock.release();
		fill = *v;    // put in new data (must support assignment)
		return true;
	}
}

template<typename KEY, typename DATA, typename MUTEX, typename EQFUNC, typename ALLOC>
DATA *TWDenseHash<KEY,DATA,MUTEX,EQFUNC,ALLOC>::find( KEY& key ) {
	DATA *ret = NULL;
//	if(values.empty())
//		return NULL;
	internal_zhashiterator it = values.find(&key);
	if (it == values.end()) {
		_lock.release();
	} else {
		ret = it->second;
		_lock.release();
	}
	return ret;
}




#endif /* TW_DENSEHASH_H_ */
