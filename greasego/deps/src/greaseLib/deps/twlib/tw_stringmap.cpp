/*
 * tw_stringmap.cpp
 *
 *  Created on: Mar 8, 2012
 *      Author: ed
 * (c) 2012, WigWag LLC
 */

#include <TW/tw_stringmap.h>

using namespace TWlib;

// *
// * These functions override TW_KHash_32, so that we free both the char *, and what the char * is pointing to...
// *
/*
bool TW_StringStringMap::removeAll() {
	TW_StringStringMap::HashIterator iter(*this);
	while(iter.getNext()) {  // clear out the string data...
		if(iter.data() && *iter.data()) {
			__TW_HASH_DEBUG("Free data()\n",NULL);
			getAllocator()->i_free(*iter.data());
		}
		if(iter.key() && *iter.key()) {
			__TW_HASH_DEBUG("Free key()\n",NULL);
			getAllocator()->i_free(const_cast<char *>(*iter.key()));
		}
	}
	TW_KHash_32<char const*, char *, TW_NoMutex, StringMap_eqstrP, TWlib::Allocator<TWlib::Alloc_Std> >::removeAll();
}

char **TW_StringStringMap::addReplaceNew( const char * const & key ) {
	_lock.acquire();
	char **ret = NULL;

	khiter_t k = kh_get_A(KHASHMAP,key);
	if(k != kh_end(KHASHMAP)) { // don't have it
		char **old = kh_value(KHASHMAP, k);
		if (*old) _alloc->i_free(*old);
//		TW_DELETE_WALLOC(old,DATA,_alloc); // out w/ the old, in with the new
		_alloc->i_free(old);
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),char *,char *(),_alloc);
		ret = kh_value(KHASHMAP, k);
	} else {
		int r; // TODO - deal with collisions
		k = kh_put_A(KHASHMAP,key,&r);
		if(!r) __TW_HASH_DEBUGL("WARNING: collision\n",NULL);
		TW_NEW_WALLOC(kh_value(KHASHMAP, k),char *,char *(),_alloc);
		ret = kh_value(KHASHMAP, k);
	}
	_lock.release();
	return ret;
}

bool TW_StringStringMap::remove(const char * const & key) {
	char **d = NULL;
	bool ret = false;
//	uint32_t kval = hasher.operator ()(&key);
	_lock.acquire();
	khiter_t k = kh_get_A(KHASHMAP, key);
	if(k != kh_end(KHASHMAP)) { // don't have it
		char **d = kh_value(KHASHMAP, k);
		if (*d) _alloc->i_free(*d); // free what pointer to pointer is pointing to
		_alloc->i_free(d); // remove pointer to pointer
		kh_del_A(KHASHMAP, k);
		ret = true;
	}
	_lock.release();
	return ret;
}


*/


bool TW_StringStringMap::addReplace( const char *& key, char *& dat ) {
	CSTR *l = CSTR::wrap( key );
	CSTR *d = CSTR::wrap( dat );
	bool ret = hashT::addReplace( *l, *d );
	delete l; delete d;
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
bool TW_StringStringMap::addReplace( const char *& key, char *& dat, char *& olddat ) {
	CSTR *l = CSTR::wrap( key );
	CSTR *d = CSTR::wrap( dat );
	CSTR old;
	bool ret = hashT::addReplace( *l, *d, old );
	if(ret) {
		old.makeWeak();
		olddat = old.s;
	}
	delete l; delete d;
	return ret;
}

char **TW_StringStringMap::addReplaceNew( const char *& key ) {
	CSTR *l = CSTR::wrap( key );
	CSTR *d = hashT::addReplaceNew( *l );
	delete l;
	return &(d->s); // should return a pointer to a 'null' pointer (b/c string is blank)
}

char **TW_StringStringMap::addNoreplaceNew( const char *& key ) {
	CSTR *l = CSTR::wrap( key );
	CSTR *d = hashT::addNoreplaceNew( *l );
	delete l;
	return &(d->s); // should return a pointer to a 'null' pointer (b/c string is blank)
}

bool TW_StringStringMap::remove( const char*& key ) {
	CSTR *l = CSTR::wrap( key );
	bool ret = hashT::remove(*l);
	delete l;
	return ret;
}

// same issues as above..
bool TW_StringStringMap::remove( const char*& key, char*& fill ) {
	CSTR *l = CSTR::wrap( key );
	CSTR f;
	bool ret = hashT::remove(*l,f);
	if(ret) {
		f.makeWeak(); // so when the object goes, the string is still there...
		fill = f.s;
	}
	delete l;
	return ret;
}
/**
 * This function returns a pointer to the old data. It's the responsibility of the caller to free
 * this data.
 */
bool TW_StringStringMap::find( const char*& key, char*& fill ) {
	CSTR *l = CSTR::wrap( key );
	CSTR f;
	bool ret = hashT::find(*l,f);
	if(ret) {
		f.makeWeak(); // so when the object goes, the string is still there...
		fill = f.s;
	}
	delete l;
	return ret;
}

char **TW_StringStringMap::find( const char*& key ) {
	CSTR *l = CSTR::wrap( key );
	CSTR *r = hashT::find(*l);
	r->makeWeak();
	delete l;
	char **ret;
	if (r)
		return ret = &(r->s); // should return a pointer to a 'null' pointer (b/c string is blank)
	else
		return ret = NULL;
	delete r;
}

char **TW_StringStringMap::findOrNew( const char*& key ) {
	CSTR *l = CSTR::wrap( key );
	CSTR *r = hashT::findOrNew(*l);
	delete l;
	return &(r->s); // should return a pointer to a 'null' pointer (b/c string is blank)
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


TW_StringStringMap::~TW_StringStringMap() {}  //{ TW_StringStringMap::removeAll(); }

// ------------- Iterator

const char *TW_StringStringMap::StringIterator::key() {
	CSTR *r = hashT::HashIterator::key();
	return r->s;
}

char *TW_StringStringMap::StringIterator::data() {
	CSTR *r = hashT::HashIterator::data();
	if(r)
		return r->s;
	else
		return NULL;
}

