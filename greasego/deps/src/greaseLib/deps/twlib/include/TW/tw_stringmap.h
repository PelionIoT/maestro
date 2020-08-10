/*
 * tw_khashstring.h
 *
 *  Created on: Mar 8, 2012
 *      Author: ed
 * (c) 2012, WigWag LLC
 */

#ifndef TW_KHASHSTRING_H_
#define TW_KHASHSTRING_H_

#include <string.h>

#include <TW/tw_bufblk.h>
#include <TW/tw_khash.h>
#include <TW/tw_sema.h>
#include <TW/tw_utils.h>
#include <TW/tw_alloc.h>

#define TWSTRING_KEY_P( v ) const_cast<const char * &>( v )

namespace TWlib {

/**
 * simple class which holds a C String
 */
template <typename ALLOC>
class CStrCont {
public:
	char *s;
	bool weak;
	void copyfrom(const CStrCont &o) {
		if(o.s) { // if not weak, then deep copy it
			weak = false;
			s = (char *) ALLOC::malloc(strlen(o.s)+1);
			strcpy(s,o.s);
		} else
			s = NULL;
	}

	CStrCont() : s(NULL), weak(false) { }
	CStrCont( char const * cs) : s(NULL), weak(false) {
		s = ALLOC::malloc(strlen(s)+1);
		strcpy(s,cs);
	}
	static CStrCont *wrap( const char *s ) {
		CStrCont *ret = new CStrCont();
		ret->s = const_cast<char *>(s);
		ret->weak = true;
		return ret;
	}
	void makeWeak() {
		weak = true;
	}
	~CStrCont() {
		if (s && !weak) { ALLOC::free(s); s = NULL; }
	}
	CStrCont( const CStrCont &o ) : s(NULL), weak(false) { //weak(o.weak) {
//		weak = o.weak;
//		if(!o.weak) copyfrom(o); else s = o.s;
		copyfrom(o);
	}
	char *get() {
		return s;
	}
	const CStrCont<ALLOC>& operator= (const CStrCont<ALLOC> &o) {
		if(&o != this) {
			if(s && !weak) { ALLOC::free(s); s= NULL; }
			weak = o.weak;
			if(!o.weak) copyfrom(o); else s = o.s;
		}
		return *this;
	}
};


template<typename ALLOC> struct tw_hash<CStrCont<ALLOC> *> {
	inline size_t operator()(const CStrCont<ALLOC> *v) const {
//		TW_DEBUG("str: %s hash: %d\n",v->s, data_hash_Hsieh(v->s, strlen(v->s)));
		return (size_t) data_hash_Hsieh(v->s, strlen(v->s));
	}
};

//template<> struct tw_hash<zdb::ZIPCHandle *> {
//	inline size_t operator()(zdb::ZIPCHandle *h) const {
//		return (size_t) (h->hash());
//	}
//};

template<typename ALLOC>
struct StringMap_eqstrP {
	  inline int operator() (const CStrCont<ALLOC> *kt1,
	                  const CStrCont<ALLOC> *kt2) const
	  {
//		  TW_DEBUG("--------- COMPARE...\n",NULL);
		  if((kt1 == kt2) || (kt1->s == kt2->s))
			  return (1);
		  else
			  return (strcmp(kt1->s, kt2->s) == 0);
	  }
};


/**
 * This is a simple, non-template class built off TW_KHash_32 for C-string (null terminated)
 * Values and Keys are both null terminated strings.
 * Values and strings should be allocated using map.GetAllocator()
 *
 * This map manages pointers to string, not the string themselves. Except, if the pointers are not NULL on destor, it will try to free them with the
 * assigned allocator.
 */
class TW_StringStringMap : public TW_KHash_32<CStrCont<TWlib::Allocator<TWlib::Alloc_Std> >, CStrCont<TWlib::Allocator<TWlib::Alloc_Std> >,
                                              TW_NoMutex, StringMap_eqstrP<TWlib::Allocator<TWlib::Alloc_Std> >, TWlib::Allocator<TWlib::Alloc_Std> > {
public:
	typedef CStrCont<TWlib::Allocator<TWlib::Alloc_Std> > CSTR;
	typedef TW_KHash_32<CSTR, CSTR, TW_NoMutex, StringMap_eqstrP<TWlib::Allocator<TWlib::Alloc_Std> >, TWlib::Allocator<TWlib::Alloc_Std> > hashT; // shortname for our parent...

	//	typedef hashT::HashIterator HashIterator;

//	char **addReplaceNew(const char * const & key);
//	bool remove(const char * const & key);
//	bool removeAll();

	class StringIterator : public hashT::HashIterator {
	public:
		StringIterator(hashT &map) : hashT::HashIterator( map ) { } // _map.gotoStart(*this); }
		const char *key();
		char *data();
		using hashT::HashIterator::setData;
		using hashT::HashIterator::getNext;
		using hashT::HashIterator::atEnd;
		void release();
//		bool getPrev();
		friend class TW_KHash_32<CSTR, CSTR, TW_NoMutex, StringMap_eqstrP<TWlib::Allocator<TWlib::Alloc_Std> >, TWlib::Allocator<TWlib::Alloc_Std> >;
	protected:
		using hashT::HashIterator::_map;
		using hashT::HashIterator::_iter;
	};


	bool addReplace( const char *& key, char *& dat );
	bool addReplace( const char *& key, char *& dat, char *& olddat );
	bool addNoreplace( const char *& key, char *& dat );
	char **addReplaceNew( const char *& key );
	char **addNoreplaceNew( const char *& key );
	bool remove( const char*& key );
	bool remove( const char*& key, char*& fill );
	bool find( const char*& key, char*& fill );
	char **find( const char*& key );
	char **findOrNew( const char*& key );
//	bool removeAll();
//	int size();




	~TW_StringStringMap();
};

template<typename ALLOC>
struct StringMapG_eqstrP {
	  inline int operator() (const CStrCont<ALLOC> *kt1,
	                  const CStrCont<ALLOC> *kt2) const
	  {
		  return (strcmp(kt1->s, kt2->s) == 0);
	  }
};



/**
 * Generic C-string to DATA map
 */
template<typename DATA, typename MUTEX, typename ALLOC>
class TW_StringMapGeneric : public TW_KHash_32<CStrCont<ALLOC>, DATA, MUTEX, StringMapG_eqstrP<ALLOC>, ALLOC> {
public:
	typedef CStrCont<ALLOC> CSTR;
	typedef typename TWlib::TW_KHash_32<CStrCont<ALLOC>, DATA, MUTEX, StringMapG_eqstrP<ALLOC>, ALLOC> hashT; // shortname for our parent...
//	using hashT::KHASHMAP;
public:

//	char **addReplaceNew(const char * const & key);
//	bool remove(const char * const & key);
//	bool removeAll();

	class StringIterator : public hashT::HashIterator {
	public:
		StringIterator(hashT &map) : hashT::HashIterator( map ) { } // _map.gotoStart(*this); }
		const char *key();
		using hashT::HashIterator::data;
		using hashT::HashIterator::setData;
		using hashT::HashIterator::getNext;
		using hashT::HashIterator::atEnd;
		void release();
//		bool getPrev();
		friend class TW_StringMapGeneric<DATA,MUTEX,ALLOC>;
		friend class TW_KHash_32<CStrCont<ALLOC>, DATA, MUTEX, StringMapG_eqstrP<ALLOC>, ALLOC>;
	protected:
		using hashT::HashIterator::_map;
		using hashT::HashIterator::_iter;
	};

	bool addReplace( const char *& key, DATA& dat );
	bool addReplace( const char *& key, DATA& dat, DATA& olddat );
	bool addNoreplace( const char *& key, DATA& dat );
	DATA *addReplaceNew( const char *& key );
	DATA *addNoreplaceNew( const char *& key );
	bool remove( const char*& key );
	bool remove( const char*& key, DATA& fill );
	bool find( const char*& key, DATA& fill );
	DATA *find( const char*& key );
	DATA *findOrNew( const char*& key );
	using TW_KHash_32<CStrCont<ALLOC>, DATA, MUTEX, StringMapG_eqstrP<ALLOC>, ALLOC>::removeAll;
	using TW_KHash_32<CStrCont<ALLOC>, DATA, MUTEX, StringMapG_eqstrP<ALLOC>, ALLOC>::size;
	using TW_KHash_32<CStrCont<ALLOC>, DATA, MUTEX, StringMapG_eqstrP<ALLOC>, ALLOC>::getAllocator;
	using TW_KHash_32<CStrCont<ALLOC>, DATA, MUTEX, StringMapG_eqstrP<ALLOC>, ALLOC>::releaseIter;
	//	int size();



	~TW_StringMapGeneric();
protected:
	friend class TW_StringMapGeneric<DATA,MUTEX,ALLOC>::StringIterator;
	friend class TW_KHash_32<CStrCont<ALLOC>, DATA, MUTEX, StringMapG_eqstrP<ALLOC>, ALLOC>::HashIterator;
};

} // end namespace



// -----------------------------------------------------------------
// TW_StringMapGeneric
// -----------------------------------------------------------------



template<typename DATA, typename MUTEX, typename ALLOC>
bool TW_StringMapGeneric<DATA,MUTEX,ALLOC>::addReplace( const char *& key, DATA& dat ) {
	CSTR *l = CSTR::wrap( key );
	bool ret = hashT::addReplace( *l, dat );
	delete l;
	return ret;
}

/**
 * This function returns a pointer to the old data. It's the responsibility of the caller to free
 * this data.
 * @param key
 * @param dat
 * @param olddat Filled with a pointer to old data
 * @return
 */
template<typename DATA, typename MUTEX, typename ALLOC>
bool TW_StringMapGeneric<DATA,MUTEX,ALLOC>::addReplace( const char *& key, DATA& dat, DATA& olddat ) {
	CSTR *l = CSTR::wrap( key );
	bool ret = hashT::addReplace( *l, dat, olddat );
	delete l;
	return ret;
}

template<typename DATA, typename MUTEX, typename ALLOC>
DATA *TW_StringMapGeneric<DATA,MUTEX,ALLOC>::addReplaceNew( const char *& key ) {
	CSTR *l = CSTR::wrap( key );
	DATA *d = hashT::addReplaceNew( *l );
	delete l;
	return d; // should return a pointer to a 'null' pointer (b/c string is blank)
}

template<typename DATA, typename MUTEX, typename ALLOC>
DATA *TW_StringMapGeneric<DATA,MUTEX,ALLOC>::addNoreplaceNew( const char *& key ) {
	CSTR *l = CSTR::wrap( key );
	DATA *d = hashT::addNoreplaceNew( *l );
	delete l;
	return d; // should return a pointer to a 'null' pointer (b/c string is blank)
}

template<typename DATA, typename MUTEX, typename ALLOC>
bool TW_StringMapGeneric<DATA,MUTEX,ALLOC>::remove( const char*& key ) {
	CSTR *l = CSTR::wrap( key );
	bool ret = hashT::remove(*l);
	delete l;
	return ret;
}

// same issues as above..
template<typename DATA, typename MUTEX, typename ALLOC>
bool TW_StringMapGeneric<DATA,MUTEX,ALLOC>::remove( const char*& key, DATA& fill ) {
	CSTR *l = CSTR::wrap( key );
	bool ret = hashT::remove(*l,fill);
	delete l;
	return ret;
}

template<typename DATA, typename MUTEX, typename ALLOC>
bool TW_StringMapGeneric<DATA,MUTEX,ALLOC>::find( const char*& key, DATA& fill ) {
	CSTR *l = CSTR::wrap( key );
	bool ret = hashT::find(*l,fill);
	delete l;
	return ret;
}

template<typename DATA, typename MUTEX, typename ALLOC>
DATA *TW_StringMapGeneric<DATA,MUTEX,ALLOC>::find( const char*& key ) {
	CSTR *l = CSTR::wrap( key );
	DATA *r = hashT::find(*l);
	delete l;
	return r;
}

template<typename DATA, typename MUTEX, typename ALLOC>
DATA *TW_StringMapGeneric<DATA,MUTEX,ALLOC>::findOrNew( const char*& key ) {
	CSTR *l = CSTR::wrap( key );
	DATA *r = hashT::findOrNew(*l);
	delete l;
	return r; // should return a pointer to a 'null' pointer (b/c string is blank)
}

// These two should remian the same:
//  bool TW_StringStringMap::removeAll();
//  int TW_StringStringMap::size();


/*
bool TW_StringStringMap::addReplace( const char *& key, char *& dat, char *& olddat );
bool TW_StringStringMap::addNoreplace( const char *& key, char *& dat );
char **TW_StringStringMap::addReplaceNew( const char *& key );
char **TW_StringStringMap::addNoreplaceNew( const char *& key );
bool TW_StringStringMap::remove( const char*& key );
bool TW_StringStringMap::remove( const char*& key, char*& fill );
bool TW_StringStringMap::find( const char*& key, char*& fill );
char **TW_StringStringMap::find( const char*& key );
char **TW_StringStringMap::findOrNew( const char*& key );
bool TW_StringStringMap::removeAll();
int TW_StringStringMap::size();

*/

template<typename DATA, typename MUTEX, typename ALLOC>
TW_StringMapGeneric<DATA,MUTEX,ALLOC>::~TW_StringMapGeneric() {}  //{ TW_StringStringMap::removeAll(); }


// --------------------- Iterator

template<typename DATA, typename MUTEX, typename ALLOC>
const char *TW_StringMapGeneric<DATA,MUTEX,ALLOC>::StringIterator::key() {
//	if(_iter != kh_end(_map.KHASHMAP)) {
//		return kh_key(_map.KHASHMAP, _iter).s;
//	} else {
//		return NULL;
//	}
	CSTR *r = hashT::HashIterator::key();
	return r->s;
}




#endif /* TW_KHASHSTRING_H_ */
