// WigWag LLC
// (c) 2010
// tw_sparsehash.h
// Author: ed
// Sep 30, 2010 - modified from zdb/sparsehash.h

/*
 * sparsehash.h
 *
 *  Created on: May 23, 2010
 *      Author: ed
 */

#ifndef TW_SPARSEHASH_H_
#define TW_SPARSEHASH_H_

//#include <ace/Guard_T.h>
//#include <ace/Global_Macros.h>
//#include <ace/Malloc_Allocator.h>
//#include <ace/Synch.h>


#include "google/sparse_hash_map"

#include "tw_macros.h"
#include "tw_fifo.h"
#include "tw_stack.h"
//#include "zstring.h"
//#include "dsrnstring.h"

#include <ext/hash_map>

using google::sparse_hash_map;      // namespace where class lives by default
#define _USE_GOOGLE_

using namespace TWlib;

// specialization for ZStrings
/*namespace __gnu_cxx
{
        template<> struct hash< std::string >
        {
                size_t operator()( const std::string& x ) const
                {
                        return hash< const char* >()( x.c_str() );
                }
        };
}
*/


namespace TWlib {
//
//// 'eqstr' struct - just a comparison function in a template
//struct ZStr_eqstr {
//	inline bool operator()(const ZString &s1, const ZString &s2) const {
//		if (zdb::ZString::compare(s1, s2) == 0)
//			return true;
//		else
//			return false;
//	}
//};
//
//struct ZStrP_eqstr {
//	inline bool operator()(const ZString *s1, const ZString *s2) const {
//		if((s1 != NULL) && (s2 != NULL)) {
///*		if (zdb::ZString::compare(*s1, *s2) == 0)
//			return true;
//		else
//			return false;
//*/
//		return s1->hashCRC32() == s2->hashCRC32();
//		}
//		return true;
//	}
//};
}



// specialize of the GNU hash function for ZStrings
namespace TWlib {

template<class T>
struct tw_hash {
	size_t operator()(T x) const {
		return 0;
	}
};

template<> struct tw_hash<zdb::ZString> {
	inline size_t operator()(const zdb::ZString& x) const {
		return (size_t) x.hashCRC32();
	}
};
template<> struct tw_hash<zdb::ZString *> {
	inline size_t operator()(zdb::ZString *x) const {
		return (size_t) x->hashCRC32();
	}
};

}



// wraps the google sparse_hash_map -
//
// tw_sparsehash manages the memory of the data it holds. When the Hash is deleted, it will delete data (and keys).
//
// MUTEX is a ACE_Thread_Mutex or ACE_Null_Mutex or similar
// DATA must implement:
// copy constructor
// assignment (operator =)
// default constructor
// destructor (seems to not deallocate right without)
//
// KEY must implement
// specialize __gnu_cxx::hash<KEY *> (if does not exist already)
// EQFUNC: implement the operator(KEY *) for eqstr struct (see above_...
// copy constructor
// destructor
template<class KEY, class DATA, class MUTEX, class EQFUNC>
class tw_sparsehash  {
public:
	tw_sparsehash(KEY& deletekey, TW_Allocator *alloc = NULL, int items=0);
	~tw_sparsehash();
	bool addReplace( KEY& key, DATA& dat );
	bool addNoreplace( KEY& key, DATA& dat );
	DATA *addReplaceNew( KEY& key );
	bool remove( KEY& key );
	bool remove( KEY& key, DATA& fill );
	bool find( KEY& key, DATA& fill );
	DATA *find( KEY& key );
	bool removeAll();

#ifdef _USE_GOOGLE_
	typedef typename sparse_hash_map<KEY *, DATA *, tw_hash<KEY *>, EQFUNC>::iterator internal_zhashiterator;
#else
//	typedef typename __gnu_cxx::hash_map<KEY *, DATA *, __gnu_cxx::hash<KEY *>, EQFUNC>::iterator internal_zhashiterator;
#endif

	class tw_hashiterator {
	public:
//		tw_hashiterator() { }
		tw_hashiterator(tw_sparsehash &map);
		KEY *key();
		DATA *data();
		bool getNext();
//		bool getPrev();
		friend class tw_sparsehash;
	protected:
		tw_sparsehash &_map;
		internal_zhashiterator _it;
//		~tw_hashiterator();
	};

//	tw_hashiterator getIter() { return tw_hashiterator(*this); }
	void gotoStart(tw_hashiterator &i);
//	void gotoEnd(tw_hashiterator &i);
	void releaseIter();

#ifdef _USE_GOOGLE_
	sparse_hash_map<KEY *, DATA *, zdb_hash<KEY *>, EQFUNC> values;
#else
//	__gnu_cxx::hash_map<KEY *, DATA *, __gnu_cxx::hash<KEY *>, EQFUNC> values;
#endif
	ACE_Allocator *getAllocator() { return _alloc; }
protected:
	MUTEX _lock;
	// the internal map is actually a map of pointer - this class
	// has responsibility for memeory management. I do this b/c I dont trust the STL implementations


	int _iterators_out;
	KEY *_emptykey;
	ACE_Allocator *_alloc;
};


template<class KEY, class DATA, class MUTEX, class EQFUNC>
tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::tw_sparsehash::tw_hashiterator::tw_hashiterator(tw_sparsehash<KEY,DATA,MUTEX,EQFUNC> &map) :
_map( map ) { }

template<class KEY, class DATA, class MUTEX, class EQFUNC>
DATA *tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::tw_sparsehash::tw_hashiterator::data() {
	if(_it != _map.values.end()) {
		return _it->second;
	} else {
		return NULL;
	}

}

template<class KEY, class DATA, class MUTEX, class EQFUNC>
KEY *tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::tw_sparsehash::tw_hashiterator::key() {
	if(_it != _map.values.end()) {
		return _it->first;
	} else {
		return NULL;
	}
}

template<class KEY, class DATA, class MUTEX, class EQFUNC>
bool tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::tw_sparsehash::tw_hashiterator::getNext() {
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

/*template<class KEY, class DATA, class MUTEX, class EQFUNC>
bool tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::tw_sparsehash::tw_hashiterator::getPrev() {
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

//template<class KEY, class DATA, class MUTEX, class EQFUNC>
//tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::tw_hashiterator tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::getIter() {
//	return tw_hashiterator(*this);
//}


template<class KEY, class DATA, class MUTEX, class EQFUNC>
void tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::gotoStart(tw_hashiterator &i) {
	_lock.acquire();
	_iterators_out++;
	i._it = values.begin();
	_lock.release();
}

/*template<class KEY, class DATA, class MUTEX, class EQFUNC>
void tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::gotoEnd(tw_hashiterator &i) {
	_lock.acquire();
	_iterators_out++;
	i._it = values.end();
	_lock.release();
}*/

template<class KEY, class DATA, class MUTEX, class EQFUNC>
void tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::releaseIter() {
	_lock.acquire();
	if(_iterators_out > 0)
		_iterators_out--;
	_lock.release();
}

// emptykey is required by sparse_hash_map
// ...this is a known key that will never be in the actual data set
template<class KEY, class DATA, class MUTEX, class EQFUNC>
tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::tw_sparsehash(KEY& emptykey, TW_Allocator *alloc, int items ) :
_iterators_out( 0 ),
_emptykey( NULL )
{
// http://google-sparsehash.googlecode.com/svn/trunk/doc/dense_hash_map.html#6
	if(items)
		values.resize(items);
	if(!alloc)
		_alloc = TW_Allocator::instance();
	else
		_alloc = alloc;

	ACE_NEW_MALLOC
	(_emptykey,
			(reinterpret_cast<KEY *>
			(this->_alloc->malloc (sizeof (KEY)))),
			(KEY) (emptykey) ); // copy the emptykey to our _emptykey - use our Allocator

#ifdef _USE_GOOGLE_
	values.set_deleted_key(_emptykey);
#endif
}

template<class KEY, class DATA, class MUTEX, class EQFUNC>
bool tw_sparsehash<KEY, DATA, MUTEX, EQFUNC>::removeAll() {
	_lock.acquire();
#ifdef _USE_GOOGLE_
	values.set_deleted_key(_emptykey);
#endif
	//	int c = 0;
	KEY *k;
	internal_zhashiterator it = values.begin();
	while (it != values.end()) {

		if (it->first) // key
			k = it->first;
		else
			k=NULL;

		ZDB_DEBUG("removing key: %x, val: %x\n",k,it->second);

		if (it->second) { // data
			ACE_DES_FREE( it->second, this->_alloc->free, DATA);
		}

		values.erase(it); // this order is critical - you have to erase pair, while the key is still valid...
		if(k) // ...now delete key
			ACE_DES_FREE( k, this->_alloc->free, KEY);

		it = values.begin();
//		it++;
		//			c++;
	}
	values.resize(0);
	//  printf("rec count %d\n",values.size());
	values.erase(values.begin(), values.end());
	_lock.release();
	//  printf("deleted: %d records\n", c );
	return true;
}

template<class KEY, class DATA, class MUTEX, class EQFUNC>
tw_sparsehash<KEY, DATA, MUTEX, EQFUNC>::~tw_sparsehash() {
	removeAll();
	if(_emptykey)
		ACE_DES_FREE( _emptykey, this->_alloc->free, KEY);
}

// replace the data if it exists.
template<class KEY, class DATA, class MUTEX, class EQFUNC>
bool tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::addReplace( KEY& key, DATA& dat ) {
	KEY *k = NULL;
	DATA *v = NULL;
	_lock.acquire();
	internal_zhashiterator it = values.find(&key);
	ACE_NEW_MALLOC_RETURN
			(v,	(reinterpret_cast<DATA *>
				(this->_alloc->malloc (sizeof (DATA)))),
				(DATA) (dat), false ); // copy data

	if (it == values.end()) {
		ACE_NEW_MALLOC_RETURN
		(k,	(reinterpret_cast<KEY *>
			(this->_alloc->malloc (sizeof (KEY)))),
			(KEY) (key), false ); // copy key - if new key

		values[k] = v; // add pair
	} else {
//		delete it->second; // delete old data
		ACE_DES_FREE( it->second, this->_alloc->free, DATA);
		it->second = v;    // put in new data
	}
	_lock.release();
	return true;
}

template<class KEY, class DATA, class MUTEX, class EQFUNC>
DATA *tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::addReplaceNew( KEY& key ) {
	KEY *k = NULL;
	DATA *v = NULL;
	bool iamempty = false;
	_lock.acquire();
	ACE_NEW_MALLOC_RETURN
			(v,	(reinterpret_cast<DATA *>
				(this->_alloc->malloc (sizeof (DATA)))),
				(DATA) (), false ); // copy data
	internal_zhashiterator it;
//	if(values.empty())
//		iamempty = true;
//	else
		it = values.find(&key);
	if (it == values.end()) {
		ACE_NEW_MALLOC_RETURN
		(k,	(reinterpret_cast<KEY *>
			(this->_alloc->malloc (sizeof (KEY)))),
			(KEY) (key), false ); // copy key - if new key

		values[k] = v;
	} else {
		ACE_DES_FREE( it->second, this->_alloc->free, DATA);
//		delete it->second; // delete old data
		it->second = v;    // put in new data
	}
	_lock.release();
	return v;
}


template<class KEY, class DATA, class MUTEX, class EQFUNC>
bool tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::addNoreplace( KEY& key, DATA& dat ) {
	KEY *k = NULL;
	DATA *v = NULL;
	_lock.acquire();
	internal_zhashiterator it = values.find(&key);
	if (it == values.end()) {
		ACE_NEW_MALLOC_RETURN
				(v,	(reinterpret_cast<DATA *>
					(this->_alloc->malloc (sizeof (DATA)))),
					(DATA) (dat), false ); // copy data

		ACE_NEW_MALLOC_RETURN
		(k,	(reinterpret_cast<KEY *>
			(this->_alloc->malloc (sizeof (KEY)))),
			(KEY) (key), false ); // copy key - if new key

		values[k] = v;
		_lock.release();
		return true;
	} else {
		_lock.release();
		return false; // and return...
	}
}

template<class KEY, class DATA, class MUTEX, class EQFUNC>
bool tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::remove( KEY& key, DATA& fill ) {
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
		ZDB_DEBUG("removing key: %x, val: %x\n",k,v);
		ACE_DES_FREE( k, this->_alloc->free, KEY);
		ACE_DES_FREE( v, this->_alloc->free, DATA);
//		it->first = NULL;
//		it->second = NULL;
//		if(k) delete k;
//		if(v) delete v;
		return true;
	}
}

template<class KEY, class DATA, class MUTEX, class EQFUNC>
bool tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::remove( KEY& key ) {
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
		ACE_DES_FREE( k, this->_alloc->free, KEY);
		ACE_DES_FREE( v, this->_alloc->free, DATA);
//		if(k) delete k;
//		if(v) delete v;
		return true;
	}

}

template<class KEY, class DATA, class MUTEX, class EQFUNC>
bool tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::find( KEY& key, DATA& fill ) {
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

template<class KEY, class DATA, class MUTEX, class EQFUNC>
DATA *tw_sparsehash<KEY,DATA,MUTEX,EQFUNC>::find( KEY& key ) {
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


#ifdef _NOTUSING_STUFF


template <class DATA, class LOCK>
class super_dataNode;

// this is a hash table of hash tables, which then stores a DATA
//
// data is managed and deleted by this table
//
// first we hash on the DS in DS:RN, this give us a table for each DS.
// we then have a hashtable for each namespace identifier in the DSRN.
// so for instance: framez.test.program:top.second.third.bottom will have a
// table for 'framez.test.program' and then a separate table for each namespace
// in the record name: top is the table framex.test.program points to, second
// then has a table where third is.. etc.
//
// MUTEX is a TW_Thread_Mutex or TW_Null_Mutex or similar
// DATA must implement:
// copy constructor
// assignment (operator =)
// default constructor
// copy constructor
// destructor (seems to not deallocate right without)
//

template <class DATA, class LOCK>
class ZDSRNSparseSupermap
 : protected tw_sparsehash<ZString, super_dataNode<DATA,LOCK>, TW_Null_Mutex, ZStrP_eqstr>
{
protected:
#ifdef _USE_GOOGLE_
	typedef typename sparse_hash_map<ZString *, super_dataNode<DATA,LOCK> *, zdb_hash<ZString *>, ZStrP_eqstr>::iterator internal_zhashiterator;
#else
	//	typedef typename __gnu_cxx::hash_map<ZString *, super_dataNode<DATA,LOCK> *, __gnu_cxx::hash<ZString *>, ZStrP_eqstr>::iterator internal_zhashiterator;
#endif
public:
	typedef tw_sparsehash<ZString, super_dataNode<DATA,LOCK>, TW_Null_Mutex, ZStrP_eqstr>  dsrnHashMap;
	class ListPair {
		friend class ZDSRNSparseSupermap<DATA,LOCK>;
	public:
		ListPair() : dat( NULL ), path() {};
		ListPair& operator= (const ListPair& rhs) { this->path = rhs.path; this->dat = rhs.dat; return *this; }
		ZString path;  // basically a pointer to the key
		DATA *dat;      // a pointer to the data
	};

	typedef typename TWlib::tw_FIFO<ListPair> PairList;
	typedef typename TWlib::tw_FIFO<DATA *> DataList;
	ZDSRNSparseSupermap(ZString& emptykey, TW_Allocator *alloc = NULL, TW_Allocator *localalloc = NULL);
	~ZDSRNSparseSupermap();
	bool addReplace( ZString& dsrn, DATA& dat );
	bool addOrGet( ZString& dsrn, DATA& dat );
	DATA *newOrGet( ZString& dsrn );

//	bool addNoreplace( ZString& dsrn, DATA& dat );
	DATA *addReplaceNew( ZString& dsrn );
	bool removePath( ZString& dsrn );
	bool removePath( ZString& dsrn, DATA &fill );
	bool findExplicitPath( ZString& dsrn, DATA& fill );
	DATA *findExplicitPath( ZString& dsrn );


//	bool findExplicitPath( ZString& dsrn, DataList &list );
	bool findFullPathPair( ZString& dsrn, PairList &list );
	bool findFullPath( ZString& dsrn, DataList &list );

	bool removeAll();

	class ZSuperIterator {
	public:
//		tw_hashiterator() { }
		ZSuperIterator(ZDSRNSparseSupermap<DATA,LOCK> &map);
		ZString *key();
		DATA *data();
		bool getNext();
//		bool getPrev();
		friend class ZDSRNSparseSupermap<DATA,LOCK>;
	protected:
		TWlib::tw_Stack<dsrnHashMap *> _mapdepth;
		TWlib::tw_Stack<internal_zhashiterator> _iterdepth;

		tw_sparsehash<ZString, super_dataNode<DATA,LOCK>, TW_Null_Mutex, ZStrP_eqstr> &_map;
		internal_zhashiterator _toplevel_iterator;
//		~tw_hashiterator();
	};

//	tw_hashiterator getIter() { return tw_hashiterator(*this); }
	void gotoStart(ZSuperIterator &i);
//	void gotoEnd(tw_hashiterator &i);
	void releaseIter();
protected:
	bool buildOrFindPath( ZString &dsrn, super_dataNode<DATA,LOCK> *&found );
	bool findPathList( ZString &dsrn, PairList &list );
	bool findNode( ZString &dsrn, super_dataNode<DATA,LOCK> *&found );
	void clearmap( dsrnHashMap *mapp );

	LOCK _lock;
	ZString *_emptykey;
	ACE_Allocator *_alloc;  // storage of the hashmap and all data
	ACE_Allocator *_localalloc; // this allocator use for temporary storage of results (like ListPair)
	tw_sparsehash <ZString, super_dataNode<DATA,LOCK>, TW_Null_Mutex, ZStrP_eqstr> &_DSmap; // the root map reference (which is ourselves - just here for ease of viewing)
//	dsrnHashMap &_DSmap;
};

template <class DATA, class LOCK>
class super_dataNode {
protected:
	friend class ZDSRNSparseSupermap<DATA,LOCK>;
//	ZDSRNSparseSupermap<DATA,LOCK>::ResultList _dataList;
//	TWlib::tw_FIFO<DATA *> _dataList;
	bool _datAssigned;
	DATA _d;
	tw_sparsehash <ZString, super_dataNode<DATA,LOCK>, TW_Null_Mutex, ZStrP_eqstr> *_map;
public:
	super_dataNode( DATA &d ) : _datAssigned( true ), _map( NULL ), _d( d ) {}
	super_dataNode() : _datAssigned( false ), _d(), _map( NULL ) {}
	super_dataNode( const super_dataNode &o ) : _datAssigned( true ) { this->_d = o.d; this->_map = o._map; }
	super_dataNode& operator=(const super_dataNode &rhs) { this->_datAssigned = true; this->_d = rhs._d; this->_map = rhs._map; }
	~super_dataNode() {
//		if(_map) delete _map;
	}
};



template <class DATA, class LOCK>
class ZDSRNDenseSupermap {

};



template <class DATA, class LOCK>
ZDSRNSparseSupermap<DATA,LOCK>::ZDSRNSparseSupermap(ZString& emptykey, TW_Allocator *alloc, TW_Allocator *localalloc) :
	tw_sparsehash<ZString, super_dataNode<DATA,LOCK>, TW_Null_Mutex, ZStrP_eqstr>( emptykey, alloc ),
	_DSmap(*this)
{
	if(!alloc)
		_alloc = TW_Allocator::instance();
	else
		_alloc = alloc;

	if(!localalloc)
		_localalloc = TW_Allocator::instance();
	else
		_localalloc = alloc;

	ACE_NEW_MALLOC
	(_emptykey,
			(reinterpret_cast<ZString *>
			(this->_alloc->malloc (sizeof (ZString)))),
			(ZString) (emptykey) ); // copy the emptykey to our _emptykey - use our Allocator

//	_DSmap.set_deleted_key(_emptykey); // handled init part of constructor
}

// returns true if it had to create the node - false if it found it
template<class DATA, class LOCK>
bool ZDSRNSparseSupermap<DATA,LOCK>::buildOrFindPath( ZString &dsrn, super_dataNode<DATA,LOCK> *&found ) {

	char debug[30];
	string dump;

	ZString look;
	super_dataNode<DATA, LOCK> *walk = NULL;
	super_dataNode<DATA, LOCK> *walkahead = NULL;
	super_dataNode<DATA, LOCK> *parent = NULL;
	super_dataNode<DATA, LOCK> *newentry = NULL;
	dsrnHashMap *newmap = NULL;

	bool newpath = false;
	ZStringIter iterzs;

	if (DSRNString::extractDSID(dsrn, look)) { // parse the DS
		ZDB_DEBUG("--------\n",NULL);
		ZDB_DEBUGLT("found DS: %s\n", look.toCString(debug,30));
		if ((walk = _DSmap.find(look)) == NULL) { // we dont have a DS key, so create one
			walk = _DSmap.addReplaceNew(look);

//			ZDB_DEBUG("_DSmap is size(): %d and max_size: %d \n", _DSmap.values.size(), _DSmap.values.max_size());

			ACE_NEW_MALLOC_RETURN(newmap,
					(reinterpret_cast< dsrnHashMap *>
							(this->_alloc->malloc (sizeof (dsrnHashMap)))),
					(dsrnHashMap) (*_emptykey, _alloc),
					-1);
			walk->_map = newmap;
			newpath = true;
			ZDB_DEBUG("built DS node\n",NULL);
		} else
			ZDB_DEBUG("found DS node\n",NULL);
		// 'newpath' is just there to make things faster - skip lookups if obviously not there...
		dsrn.initIter(iterzs);
		while (DSRNString::extractNextRNID(dsrn, iterzs, look)) {
			ZDB_DEBUGLT("found RN: <%s>  walk: %x\n", look.toCString(debug,30), walk);
//			ZDB_DEBUGLT("dump: %s\n",look.dumpMemory(dump).c_str());
			// we already have started making a new path or...
			// we dont have this RN, then...
			// or walk->_map is NULL (which should not ever happen)
			walkahead = walk->_map->find(look);

			if(walkahead == NULL)
				ZDB_DEBUG("failed to find RN on map %x\n",walk->_map);
			else
				ZDB_DEBUG("found exising RN on map %x\n",walk->_map);

			if (newpath || (!walk->_map) || ((walk->_map) && (walkahead == NULL))) { // we dont have a DS key, so create one
				if(newpath)
					ZDB_DEBUG("newpath\n",NULL);
				if(!walk->_map) {
					ZDB_DEBUGLT("error: walk->_map was NULL - should not happen\n",NULL);
					ACE_NEW_MALLOC_RETURN(walk->_map,
							(reinterpret_cast< dsrnHashMap *>
									(this->_alloc->malloc (sizeof (dsrnHashMap)))),
							(dsrnHashMap) (*_emptykey, _alloc, 5),
							-1);
					ZDB_DEBUG("!!had to build _map !!\n",NULL);
				}
				walkahead = walk->_map->addReplaceNew(look);
//				walk = walkahead;
				ACE_NEW_MALLOC_RETURN(newmap,
						(reinterpret_cast< dsrnHashMap *>
								(this->_alloc->malloc (sizeof (dsrnHashMap)))),
						(dsrnHashMap) (*_emptykey, _alloc ),   // was _alloc, 5),
						-1);
				walkahead->_map = newmap;
				ZDB_DEBUG("built node %x w/ map %x -> added to map %x\n",walkahead,walkahead->_map,walk->_map);
//				ZDB_DEBUG("new map is size(): %d and max_size: %d \n", walkahead->_map->values.size(), walkahead->_map->values.max_size());
				//				ZString verifystr;
				super_dataNode<DATA, LOCK> v;
				if(walk->_map->find( look, v )== true)
					ZDB_DEBUG("Verify -> look:<%s> node has map: %x\n", look.toCString(debug,30), v._map);
				else
					ZDB_DEBUG("Verify - failed\n",NULL);

				newpath = true;
			} else {
				ZDB_DEBUG("walked %x\n",walkahead);
//				walk = walkahead; // move to next
			}
			walk = walkahead; // move to next

//			found = walk;
			//			if(walk->_datAssigned)
			//walk->_d = dat;
//			return newpath;
		}
		found = walk;
		//			if(walk->_datAssigned)
		//walk->_d = dat;
		return newpath;
	} else {
		ZDB_ERROR("could not parse DS\n",NULL);
		found = NULL;
		return false;
	}

}

template<class DATA, class LOCK>
bool ZDSRNSparseSupermap<DATA,LOCK>::findNode( ZString &dsrn, super_dataNode<DATA,LOCK> *&found ) {
	ZString look;
	super_dataNode<DATA, LOCK> *walk = NULL;

	ZStringIter iterzs;

	if (DSRNString::extractDSID(dsrn, look)) { // parse the DS
		if ((walk = _DSmap.find(look)) == NULL) { // we dont have a DS key, so create one
			return false;
		}
		// 'newpath' is just there to make things faster - skip lookups if obviously not there...
		dsrn.initIter(iterzs);
		while (DSRNString::extractNextRNID(dsrn, iterzs, look)) {
			if ((walk = walk->_map->find(look)) == NULL) { // we dont have a DS key, so create one
				return false;
			}
		}
		found = walk;
		return true;
	} else {
		ZDB_ERROR("could not parse DS\n",NULL);
		return false;
	}

}

template<class DATA, class LOCK>
bool ZDSRNSparseSupermap<DATA, LOCK>::findPathList( ZString &dsrn, PairList &list ) {
// TODO
	bool ret = false;
	ZString look;
	super_dataNode<DATA, LOCK> *walk = NULL;
	super_dataNode<DATA, LOCK> *lookw = NULL;

//	DataList gather( usealloc );
	ZStringIter iterzs;
	ListPair *newpair = NULL;
	ZString currpath;

	if (DSRNString::extractDSID(dsrn, look)) { // parse the DS
		currpath.append(look);
		currpath.append(':');
		walk = _DSmap.find(look);
		if (walk == NULL) { // we dont have a DS key, return
			return false;
		}

		// skip adding one here - just the DS can't have data
//		newpair = list.addEmpty();
//		newpair->dat = &(walk->_d);
//		newpair->path = currpath; // copy the current path (shallow copy - like ZString copy constructor)

		// 'newpath' is just there to make things faster - skip lookups if obviously not there...
		bool _1st = true;
		dsrn.initIter(iterzs);
		while (DSRNString::extractNextRNID(dsrn, iterzs, look)) {
			if(!_1st)
				currpath.append('.');
			currpath.append(look);
			_1st = false;
			lookw = walk->_map->find(look);
			if (lookw == NULL) { // we dont have a DS key, so create one
				ZDB_DEBUG("failed to find RN on map %x\n",walk->_map);
				return false;
			} else {
				walk = lookw;
				newpair = list.addEmpty();
				newpair->dat = &(walk->_d);
				newpair->path = currpath; // copy the current path (shallow copy - like ZString copy constructor)
			}
		}

		return true;
	} else {
		ZDB_ERROR("could not parse DS\n",NULL);
		return false;
	}

}

template<class DATA, class LOCK>
bool ZDSRNSparseSupermap<DATA, LOCK>::findFullPathPair( ZString& dsrn, PairList &list ) {
	return findPathList( dsrn, list );
}

template<class DATA, class LOCK>
bool ZDSRNSparseSupermap<DATA, LOCK>::findFullPath( ZString& dsrn, DataList &list ) {
	bool ret = false;
	ZString look;
	super_dataNode<DATA, LOCK> *walk = NULL;
//	DataList gather( usealloc );
	ZStringIter iterzs;
//	ZString currpath;

	if (DSRNString::extractDSID(dsrn, look)) { // parse the DS
//		currpath.append(look);
//		currpath.append(':');
		if ((walk = _DSmap.find(look)) == NULL) { // we dont have a DS key, return
			return false;
		}

		// skip adding one here - just the DS can't have data
//		newpair = list.addEmpty();
//		newpair->dat = &(walk->_d);
//		newpair->path = currpath; // copy the current path (shallow copy - like ZString copy constructor)

		// 'newpath' is just there to make things faster - skip lookups if obviously not there...
		bool _1st = true;
		dsrn.initIter(iterzs);
		while (DSRNString::extractNextRNID(dsrn, iterzs, look)) {
//			if(!_1st)
//				currpath.append('.');
//			currpath.append(look);
//			_1st = false;
			walk = walk->_map->find(look);
			if (walk == NULL) { // we dont have a DS key, so create one
				ZDB_DEBUG("failed to find RN on map\n",NULL);
				return false;
			} else {
//				newpair = list.addEmpty();
				DATA *pd = &(walk->_d);
				list.add(pd);
			}
		}

		return true;
	} else {
		ZDB_ERROR("could not parse DS\n",NULL);
		return false;
	}


}




template<class DATA, class LOCK>
bool ZDSRNSparseSupermap<DATA, LOCK>::addReplace(ZString& dsrn, DATA& dat) {
	super_dataNode<DATA, LOCK> *entry = NULL;
	buildOrFindPath( dsrn, entry );
	if(entry != NULL) {
		entry->_d = dat;
		return true;
	}
}

// either find the data - and fills it into 'dat' or adds a new entry and puts 'dat'
// in it.
// this function is useful if you have a default data set to use - but are happy to have it replaced with the
// existing data if its there
template<class DATA, class LOCK>
bool ZDSRNSparseSupermap<DATA, LOCK>::addOrGet(ZString& dsrn, DATA& dat) {
	super_dataNode<DATA, LOCK> *entry = NULL;
	if(buildOrFindPath( dsrn, entry )) {
		// create node - so just add
		if(entry != NULL) {
			entry->_d = dat;
			return true;
		}
	} else { // found node - just return the data here
		dat = entry->_d;
		return true;
	}
	return false;
}

template<class DATA, class LOCK>
DATA *ZDSRNSparseSupermap<DATA, LOCK>::newOrGet(ZString& dsrn) {
	super_dataNode<DATA, LOCK> *entry = NULL;
	buildOrFindPath( dsrn, entry );

	if(entry != NULL) {
		return &(entry->_d);
	}
	return NULL;
}

template<class DATA, class LOCK>
bool ZDSRNSparseSupermap<DATA, LOCK>::findExplicitPath( ZString& dsrn, DATA& fill ) {
	super_dataNode<DATA, LOCK> *found = NULL;
	if(findNode(dsrn,found)) {
		fill = found->_d;
		return true;
	} else
		return false;
}

template<class DATA, class LOCK>
DATA *ZDSRNSparseSupermap<DATA, LOCK>::findExplicitPath( ZString& dsrn ) {
	super_dataNode<DATA, LOCK> *found = NULL;
	if(findNode(dsrn,found)) {
		return &(found->_d);
	} else
		return NULL;
}


template <class DATA, class LOCK>
ZDSRNSparseSupermap<DATA,LOCK>::~ZDSRNSparseSupermap() {
/*	TWlib::tw_Stack<dsrnHashMap *> mapstack;
	TWlib::tw_Stack<dsrnHashMap::tw_hashiterator *> iterstack;
	TWlib::tw_Stack<super_dataNode<DATA, LOCK> *> nodestack;

	super_dataNode<DATA, LOCK> *looknode = NULL;
	dsrnHashMap *map; //, *mapinner;

	cout << "iterate forward..." << endl;
	tw_sparsehash<ZString, testdat, TW_Thread_Mutex, ZStrP_eqstr>::tw_hashiterator iter(testhash);
	testhash.gotoStart(iter);
	do {
		cout << "key: " << iter.key()->toCPPString() << " data: " << iter.data()->x << endl;
	} while(iter.getNext());
	testhash.releaseIter();
//	ZDSRNSparseSupermap::tw_hashiterator

	dsrnHashMap::tw_hashiterator mainiter(*this);
	dsrnHashMap::tw_hashiterator *iter;
	this->gotoStart(mainiter);
	do {
		if(mainiter.data()) {
			looknode = iter.data();

			iter = new dsrnHashMap::tw_hashiterator(*(looknode->_map));
			looknode->_map->gotoStart(*iter);
			do {
				if(iter->data()) {
					if(looknode->_map) {
						mapstack.push(map);
						iterstack.push(iter);
						iter = new dsrnHashMap::tw_hashiterator(*map);
						looknode->map->gotoStart(*iter);
					}
//					looknode = iter->data();
				} else {
					// erase from map
					ZString *s = iter->key();
					DATA *d = iter->data();
					ACE_DES_FREE( s, this->_alloc->free, ZString);
				}
			} while(iter->getNext());
			map->releaseIter();
		}
	} while(mainiter.getNext());
	this->releaseIter();
*/

//	_DSmap.

	super_dataNode<DATA, LOCK> *looknode = NULL;
//	dsrnHashMap::tw_hashiterator mainiter(*this);
//	tw_sparsehash<ZString, super_dataNode<DATA,LOCK>, TW_Null_Mutex, ZStrP_eqstr>::tw_hashiterator mainiter(*this);

/*	this->gotoStart(mainiter);  // unwind the entire map
	do {
		if(mainiter.data()) {
			looknode = iter.data();
			delete looknode; // delete all nodes - maps will get deleted inside each node if they exist.
		}
			looknode->_map->gotoStart(*iter);
	} while(mainiter.getNext());
*/

	TWlib::tw_Stack<super_dataNode<DATA, LOCK> *> nodestack;

//	dsrnHashMap *lookmap = _DSmap;


/*	internal_zhashiterator it;
	it = this->values.begin();
	while(it != this->values.end()) {
		looknode = it->second;
		if(looknode) {
			ACE_DES_FREE( looknode->_map, this->_alloc->free, dsrnHashMap);
			ACE_DES_FREE_TEMPLATE2( looknode, this->_alloc->free, super_dataNode, DATA, LOCK);
//			delete looknode; // delete all nodes - maps will get deleted inside each node if they exist.
			it->second = NULL;
		}
		it++;
	}*/
	clearmap( this );
	if(_emptykey) {
		ACE_DES_FREE( _emptykey, this->_alloc->free, ZString);
	}
	// parent destructor should take care of rest...

}

template <class DATA, class LOCK>
void ZDSRNSparseSupermap<DATA,LOCK>::clearmap( dsrnHashMap *mapp ) {
// super_dataNode's dont delete their map themselves (DATA is on the stack of each node)
	super_dataNode<DATA, LOCK> *looknode = NULL;
	internal_zhashiterator it;
	it = mapp->values.begin();
	while(it != mapp->values.end()) {
		looknode = it->second;
		if(looknode) {
			if(looknode->_map) {
				clearmap( looknode->_map );
				ACE_DES_FREE( looknode->_map, mapp->getAllocator()->free, dsrnHashMap);
			}
//			ACE_DES_FREE_TEMPLATE2( looknode, mapp->_alloc->free, super_dataNode, DATA, LOCK);
//			delete looknode; // delete all nodes - maps will get deleted inside each node if they exist.
//			it->second = NULL;
		}
		it++;
	}
}

#endif


#endif /* TW_SPARSEHASH_H_ */
