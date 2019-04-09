/*
 * test_twarray.cpp
 *
 *  Created on: Nov 28, 2011
 *      Author: ed
 * (c) 2011, WigWag LLC
 */


#include <gtest/gtest.h>

#include <malloc.h>


// For lots of debug info use:
//#define TW_HASH_DEBUG

#include "tests/testutils.h"
#include <TW/tw_defs.h>
#include <TW/tw_types.h>
#include <TW/tw_log.h>
#include <TW/tw_sema.h>
#include <TW/tw_utils.h>

#include <TW/tw_khash.h>
#include <TW/tw_densehash.h>
#include <TW/tw_stringmap.h>

#include <iostream>
#include <string>
#include <sstream>


using namespace std;

using namespace TWlib;

using ::testing::TestWithParam;
using ::testing::Values;



namespace TWlibTests {

#define MAX_OBJECT_COUNT 10000*2

class ObjTracker {
protected:
	int counter;
	char track[MAX_OBJECT_COUNT];
	static ObjTracker _instance;
public:
	static void clear() {
		memset(&(_instance.track),0,MAX_OBJECT_COUNT);
		_instance.counter = 0;
	}
	ObjTracker() : counter(0) {
		clear();
	}
	static ObjTracker &instance() {
		return _instance;
	}
	static int registerMe() {

		_instance.counter++;
		_instance.track[_instance.counter] = 1;
		return _instance.counter;
	}
	static void clearMe(int x) {
		_instance.track[x] = 0;
	}

	static int objectsRemain() {
		int c = 0;
		for(int x=0;x<MAX_OBJECT_COUNT;x++) {
			if(_instance.track[x] > 0) c++;
		}
		return c;
	}
};

TWlibTests::ObjTracker TWlibTests::ObjTracker::_instance;

class TESTD {
public:
	int x;
	string s;
	int me;
	TESTD() : x(0), s(), me(0) {
//		cout << "cstor." << endl;
		me = ObjTracker::registerMe();
	}
	TESTD(TESTD &d) : x(d.x), s(), me(0) {
//		cout << "cstor(" << x << ")" << endl;
		me = ObjTracker::registerMe();
	}
	TESTD &operator=(const TESTD &o) {
		x = o.x;
		s.clear();
		return *this;
//		cout << "operator=(" << o.x << ")" << endl;
	}
	string &toString() {
		s.clear();
		string_printf(s,"x=%d", x);
		return s;
	}
	~TESTD() { ObjTracker::clearMe(me);
	}
};

typedef TWlib::Allocator<Alloc_Std> TESTAlloc;

/**
 * D must support public default constructor
 */
template <typename KT, typename DT, typename KEQSTR>
class HashTest : public ::testing::TestWithParam<int> {
protected:
	// Objects declared here can be used by all tests in the test case for Foo.
	  KT **keyArray;

	  typedef TW_KHash_32<KT, DT, TWlib::TW_Mutex, KEQSTR, TESTAlloc > hashType;
//	  typedef TWDenseHash<KT,DT,TWlib::TW_Mutex,KEQSTR, TESTAlloc > hashType;


	  TESTAlloc *_alloc;
 public:

  // You can remove any or all of the following functions if its body
  // is empty.

	  HashTest() :
		  _alloc(NULL), keyArray(NULL)
//		  ,hashmap()
	  {
    // setup work for test
		  _alloc = new TESTAlloc();

		  cout << "SIZE: " << GetParam() << endl;

	  }

  virtual ~HashTest() {
    // You can do clean-up work that doesn't throw exceptions here.
	  for(int x=0;x<GetParam();x++ ) {
		  KT *p = keyArray[x];
		  delete p;
	  }
	  free(keyArray);
	  delete _alloc;
  }

  // If the constructor and destructor are not enough for setting up
  // and cleaning up each test, you can define the following methods:


  virtual void SetUp() {
	  keyArray = (KT **) _alloc->i_calloc((tw_size) GetParam(), sizeof(KT *));
	  if(!keyArray) {
		 FAIL() << "Could not allocate memory";
	  }
	  for(int x=0;x<GetParam();x++ ) {
		  keyArray[x] = new KT(); // placement new on new D
	  }
	  ObjTracker::clear();
    // Code here will be called immediately after the constructor (right
    // before each test).
//	  TestZMemCollLayout *l = GetParam();
//	  l->

/*		for(int x=0;x<N;x++ ) {
			out.str(test_strs[x]);
			out << "Val " << x;
			test_strs[x] = out.str();
			testd.x = x;
			hashmap.addNoreplace(test_strs[x],testd);
			cout << "Added hash \""<< test_strs[x] << "\"" << endl;
		}
		*/

  }


  virtual void TearDown() {
    // Code here will be called immediately after each test (right
    // before the destructor).
  }



};




struct TESTD_eqstrP {
	  inline int operator() (const TESTD *kt1,
	                  const TESTD *kt2) const
	  {
	    return true; // dummy
	  }
};

struct string_eqstrP {
	  inline int operator() (const string *l,
	                  const string *r) const
	  {
		  return (l->compare(*r) == 0);
	  }
};

struct string_eqstrPP {
	  inline int operator() (const string **l,
	                  const string **r) const
	  {
		  return ((*l)->compare(**r) == 0);
	  }
};

struct int_eqstrP {
	  inline int operator() (const int *l,
	                  const int *r) const
	  {
		  return (*l==*r);
	  }
};


} // end namespace TWlibTests

namespace TWlib {
	template<>
	struct tw_hash<TWlibTests::TESTD *> {
		inline size_t operator()(const TWlibTests::TESTD *v) const {
			return (size_t) v->x;
		}
	};

	template<>
	struct tw_hash<std::string *> {
		inline size_t operator()(const std::string *s) const {
			return (size_t) TWlib::data_hash_Hsieh(s->c_str(),s->length());
		}
	};

	template<>
	struct tw_hash<std::string **> {
		inline size_t operator()(std::string* const * s) const { // const function, parameter is constant pointer to non-constant C++ string
			return (size_t) TWlib::data_hash_Hsieh((*s)->c_str(),(*s)->length());
		}
	};

	template<>
	struct tw_hash<int *> {
		inline size_t operator()(int const * s) const { // const function, parameter is constant pointer to non-constant C++ string
			return (size_t) *s;
		}
	};

}

namespace TWlibTests {

string emptyString("");
string nullString("0");

class StringTest : public HashTest<string, TWlibTests::TESTD, string_eqstrP > {
public:
    hashType hashmap;
	StringTest() : HashTest<string, TWlibTests::TESTD, string_eqstrP >(), hashmap(emptyString, nullString) { }
	virtual void SetUp() {
			HashTest<string, TWlibTests::TESTD, string_eqstrP >::SetUp();
			ostringstream out;
			TESTD testd;
			for(int x=0;x<GetParam();x++ ) {
				out.str(*keyArray[x]);
				out << "Val " << x;
				*keyArray[x] = out.str();
			}
	  }
};

int emptyInt = -10;
int delInt = -20;

class HashIntTest : public HashTest<int, TWlibTests::TESTD, int_eqstrP > {
public:
    hashType hashmap;
	HashIntTest() : HashTest<int, TWlibTests::TESTD, int_eqstrP >(), hashmap(emptyInt, delInt) { }
	virtual void SetUp() {
		HashTest<int, TWlibTests::TESTD, int_eqstrP >::SetUp();
		TESTD testd;
		for(int x=0;x<GetParam();x++ ) {
			*keyArray[x] = x;
//				out.str(*keyArray[x]);
//				out << "Val " << x;
//				*keyArray[x] = out.str();
		}
	}
};


static const char *CSTRING_TEST_PREFIX_STR = "Test189728172";

class CStringHashTest : public ::testing::TestWithParam<int> {
public:
	char **cstrings;
	TW_StringStringMap hashmap;
	typedef TW_StringStringMap hashtype;
	CStringHashTest() : hashmap() {}
	virtual void SetUp() {
		cstrings = (char **) malloc(GetParam() * sizeof(char *));

		for(int x=0;x<GetParam();x++) {
			cstrings[x] = (char *) malloc(strlen(CSTRING_TEST_PREFIX_STR) + 10);
			snprintf(cstrings[x],strlen(CSTRING_TEST_PREFIX_STR) + 10,"%s-%d",CSTRING_TEST_PREFIX_STR,x);
		}

	}

	virtual void TearDown() {
		for(int x=0;x<GetParam();x++) {
			free(cstrings[x]);
		}

		free(cstrings);
	}
};


class CStringGenericHashTest : public ::testing::TestWithParam<int> {
public:
	char **cstrings;
	TW_StringMapGeneric<TESTD, TWlib::TW_Mutex, TESTAlloc> hashmap;
	typedef TW_StringMapGeneric<TESTD, TWlib::TW_Mutex, TESTAlloc> hashtype;
	CStringGenericHashTest() : hashmap() {}
	virtual void SetUp() {
		cstrings = (char **) malloc(GetParam() * sizeof(char *));

		for(int x=0;x<GetParam();x++) {
			cstrings[x] = (char *) malloc(strlen(CSTRING_TEST_PREFIX_STR) + 10);
			snprintf(cstrings[x],strlen(CSTRING_TEST_PREFIX_STR) + 10,"%s-%d",CSTRING_TEST_PREFIX_STR,x);
		}

	}

	virtual void TearDown() {
		for(int x=0;x<GetParam();x++) {
			free(cstrings[x]);
		}

		free(cstrings);
	}
};


}


using namespace TWlibTests;


TEST_P(HashIntTest, FillNEmpty) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
//		cout << "getparam() " << x << " - " << GetParam() << endl;
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		d->x = x;
	}

	ASSERT_EQ(GetParam(), hashmap.size() );

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		d->x = x;
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());

	ASSERT_EQ(0,ObjTracker::objectsRemain());
}

TEST_P(HashIntTest, FillNEmptyReplace) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}
	// do it again, replacing them
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addReplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		d->x = x;
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(HashIntTest, FillNEmptyNoReplace) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		ASSERT_TRUE(d == NULL);
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(HashIntTest, FillNEmptyRemove) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());
	TESTD fill;

	for(int x=0;x<GetParam();x++) {
		ASSERT_TRUE(hashmap.remove(*(keyArray[x]), fill));
		ASSERT_EQ(x, fill.x);
	}

	ASSERT_EQ(0,hashmap.size());

	ASSERT_TRUE(hashmap.removeAll());

}

TEST_P(HashIntTest, FillNEmptyRemove2) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());
	TESTD fill;

	for(int x=0;x<GetParam();x++) {
		ASSERT_TRUE(hashmap.remove(*(keyArray[x])));
	}

	ASSERT_EQ(0,hashmap.size());

	ASSERT_TRUE(hashmap.removeAll());

}

TEST_P(HashIntTest, FillNEmptyFind2) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	TESTD fill;
	for(int x=0;x<GetParam();x++) {
		ASSERT_TRUE(hashmap.find(*(keyArray[x]),fill));
		ASSERT_EQ(x, fill.x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(HashIntTest, FillAndNoFindRemove) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	TESTD fill;
	int junk = 1234567;
	ASSERT_FALSE(hashmap.find(junk,fill));
	ASSERT_FALSE(hashmap.remove(junk,fill));

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(HashIntTest, FillNFindOrNew) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	int junk = 1234567;
	TESTD *d = NULL;
	TESTD fill;
	d = hashmap.findOrNew(junk);
	ASSERT_TRUE(d);
	d->x = 10001;
	ASSERT_TRUE(hashmap.find(junk,fill));
	ASSERT_EQ(fill.x,10001);

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}


// --------------------------------------------------------------------

TEST_P(StringTest, FillNEmpty) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		d->x = x;
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(StringTest, FillNEmptyReplace) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}
	// do it again, replacing them
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addReplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		d->x = x;
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(StringTest, FillNEmptyNoReplace) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		ASSERT_TRUE(d == NULL);
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(StringTest, FillNEmptyRemove) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());
	TESTD fill;

	for(int x=0;x<GetParam();x++) {
		ASSERT_TRUE(hashmap.remove(*(keyArray[x]), fill));
		ASSERT_EQ(x, fill.x);
	}

	ASSERT_EQ(0,hashmap.size());

	ASSERT_TRUE(hashmap.removeAll());

}

TEST_P(StringTest, FillNEmptyRemove2) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());
	TESTD fill;

	for(int x=0;x<GetParam();x++) {
		ASSERT_TRUE(hashmap.remove(*(keyArray[x])));
	}

	ASSERT_EQ(0,hashmap.size());

	ASSERT_TRUE(hashmap.removeAll());

}

TEST_P(StringTest, FillNEmptyFind2) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	TESTD fill;
	for(int x=0;x<GetParam();x++) {
		ASSERT_TRUE(hashmap.find(*(keyArray[x]),fill));
		ASSERT_EQ(x, fill.x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(StringTest, FillAndNoFindRemove) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	TESTD fill;
	string junk("JUNKJUNK");
	ASSERT_FALSE(hashmap.find(junk,fill));
	ASSERT_FALSE(hashmap.remove(junk,fill));

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(StringTest, FillNFindOrNew) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	string junk("JUNKJUNK");
	TESTD *d = NULL;
	TESTD fill;
	d = hashmap.findOrNew(junk);
	ASSERT_TRUE(d);
	d->x = 10001;
	ASSERT_TRUE(hashmap.find(junk,fill));
	ASSERT_EQ(fill.x,10001);

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(StringTest, IterAll) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		if(x==1)
			cout << "at 1" << endl;
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	StringTest::hashType::HashIterator iter(this->hashmap);

	int c = 0;

	do {
#ifdef TW_HASH_DEBUG
		string k = *iter.key();
		cout << "Key <<" << c << ": " << k << " = " << iter.data() << endl;
#endif
		c++;
	} while(iter.getNext());

	ASSERT_EQ(GetParam(),c);

	string junk("JUNKJUNK");
	TESTD *d = NULL;
	TESTD fill;
	d = hashmap.findOrNew(junk);
	ASSERT_TRUE(d);
	d->x = 10001;
	ASSERT_TRUE(hashmap.find(junk,fill));
	ASSERT_EQ(fill.x,10001);

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}


const char *CSTR_TEST_VAL="Helloval";
/*
TEST_P(CStringHashTest, FillNEmpty) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
//		cout << "getparam() " << x << " - " << GetParam() << endl;
		char **p = hashmap.addNoreplaceNew(cstrings[x]);
		*p = (char *) hashmap.getAllocator()->i_malloc(strlen(CSTR_TEST_VAL) + 10);
		snprintf(*p,strlen(CSTR_TEST_VAL) + 10,"%s-%d",CSTR_TEST_VAL, x);
	}

	ASSERT_EQ(GetParam(), hashmap.size() );

	for(int x=0;x<GetParam();x++) {
		char **s = hashmap.find(cstrings[x]);
		ASSERT_TRUE(*s != NULL);
	}

	TW_StringMap::HashIterator iter(hashmap);
//	while(iter.getNext()) {
////		printf("iter.data() %p\n ", iter.data());
////		printf("    *iter.data() %p\n ", *iter.data());
//		if(iter.data()
//				&& *iter.data()) {
//#ifdef _DEBUG_CONFFILE_CPP
//			TW_DEBUG("Free data() - %s\n",*iter.data());
//#endif
//			hashmap.getAllocator()->i_free(const_cast<char *>(*iter.data()));
//		}
//		if(iter.key()
//				&& *iter.key()) {
//#ifdef _DEBUG_CONFFILE_CPP
//			TW_DEBUG("Free key() - %s\n",*iter.key());
//#endif
//			hashmap.getAllocator()->i_free(const_cast<char *>(*iter.key()));
//		}
//	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());

	ASSERT_EQ(0,ObjTracker::objectsRemain());
}
*/
// same as above but test with a null data string
TEST_P(CStringHashTest, FillNEmptyNullTest) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
//		cout << "getparam() " << x << " - " << GetParam() << endl;
		char **p = hashmap.addNoreplaceNew(const_cast<const char * &>(cstrings[x]));
		*p = (char *) hashmap.getAllocator()->i_malloc(strlen(CSTR_TEST_VAL) + 10);
		snprintf(*p,strlen(CSTR_TEST_VAL) + 10,"%s-%d",CSTR_TEST_VAL, x);
	}

	ASSERT_EQ(GetParam(), hashmap.size() );

	for(int x=0;x<GetParam();x++) {
		char **s = hashmap.find(const_cast<const char * &>(cstrings[x]));
		ASSERT_TRUE(*s != NULL);
	}

	char *nullp = 0;
	char *oldp = NULL;
	ASSERT_TRUE(hashmap.addReplace(const_cast<const char * &>(cstrings[0]),nullp, oldp));
	ASSERT_NE(oldp, (char *) NULL);

	hashmap.getAllocator()->i_free(oldp);

	// remove one element
	if(GetParam() > 1) {
		ASSERT_TRUE(hashmap.remove(const_cast<const char * &>(cstrings[1])));
		ASSERT_EQ(GetParam() - 1 ,hashmap.size());
	}

//	TW_StringMap::HashIterator iter(hashmap);

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());

	ASSERT_EQ(0,ObjTracker::objectsRemain());
}

TEST_P(CStringHashTest, IterAll) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
//		cout << "getparam() " << x << " - " << GetParam() << endl;
		char **p = hashmap.addNoreplaceNew(const_cast<const char * &>(cstrings[x]));
		*p = (char *) hashmap.getAllocator()->i_malloc(strlen(CSTR_TEST_VAL) + 10);
		snprintf(*p,strlen(CSTR_TEST_VAL) + 10,"%s-%d",CSTR_TEST_VAL, x);
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	hashtype::StringIterator iter(this->hashmap);

	int c = 0;

	do {
//#ifdef TW_HASH_DEBUG
		const char *k = iter.key();
		cout << "Key <<" << c << ": " << k << " = " << iter.data() << endl;
//#endif
		c++;
	} while(iter.getNext());

	ASSERT_EQ(GetParam(),c);

	static const char *junk = "JUNKJUNK";
	TESTD *d = NULL;
	char *fill;
	char **ss = hashmap.findOrNew(junk);
	ASSERT_TRUE(ss);
	ASSERT_FALSE(*ss);
	*ss = (char *) hashmap.getAllocator()->i_malloc(strlen(CSTR_TEST_VAL) + 10);
	snprintf(*ss,strlen(CSTR_TEST_VAL) + 10,"%s-TEST",CSTR_TEST_VAL);

	ASSERT_TRUE(hashmap.find(junk,fill));
	ASSERT_STREQ(*ss,fill);
	hashmap.getAllocator()->i_free(fill); // dones with that string from the query

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}


TEST_P(CStringGenericHashTest, FillNEmpty) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
//		cout << "getparam() " << x << " - " << GetParam() << endl;
		TESTD *d = hashmap.addNoreplaceNew(const_cast<const char * &>(cstrings[x]));
		ASSERT_TRUE(d != NULL);
		d->x = x;
	}

	ASSERT_EQ(GetParam(), hashmap.size() );

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(TWSTRING_KEY_P(cstrings[x]));
		ASSERT_TRUE(d != NULL);
		d->x = x;
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());

	ASSERT_EQ(0,ObjTracker::objectsRemain());
}


TEST_P(CStringGenericHashTest, IterAll) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		if(x==1)
			cout << "at 1" << endl;
		TESTD *d = hashmap.addNoreplaceNew(TWSTRING_KEY_P(cstrings[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	hashtype::StringIterator iter(this->hashmap);

	int c = 0;

	do {
//#ifdef TW_HASH_DEBUG
		const char *k = iter.key();
		cout << "Key <<" << c << ": " << k << " = " << iter.data()->x << endl;
//#endif
		c++;
	} while(iter.getNext());

	ASSERT_EQ(GetParam(),c);

	static const char *junk = "JUNKJUNK";
	TESTD *d = NULL;
	d = hashmap.findOrNew(junk);
	ASSERT_TRUE(d);
	d->x = 10001;
	TESTD fill;
	static const char *junk2 = "JUNKJUNK";
	ASSERT_TRUE(hashmap.find(junk2,fill));
	ASSERT_EQ(fill.x,10001);

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

// This test ensures keys are truly stored independently
TEST_P(CStringGenericHashTest, 1item) {
	hashtype map;
	char *junk2 = (char *) malloc(3);
	strcpy(junk2,"ID");

	TESTD d;
	d.x = 101;
	ASSERT_TRUE(map.addReplace(TWSTRING_KEY_P(junk2),d));
	ASSERT_EQ(1,map.size());

	strcpy(junk2,"XX");

	TESTD d2;
	static const char *junk3 = "ID";

	ASSERT_TRUE(map.find(junk3,d2));
	ASSERT_EQ(101,d2.x);

	ASSERT_TRUE(map.removeAll());
	free(junk2);
}

/*
TEST_P(CStringHashTest, FillNEmptyReplace) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}
	// do it again, replacing them
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addReplaceNew(*(keyArray[x]));
		d->x = x;
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		d->x = x;
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

TEST_P(CStringHashTest, FillNEmptyNoReplace) {
	cout << "Entries: " << GetParam() << endl;
	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		d->x = x;
	}

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.addNoreplaceNew(*(keyArray[x]));
		ASSERT_TRUE(d == NULL);
	}

	ASSERT_EQ(hashmap.size(), GetParam());

	for(int x=0;x<GetParam();x++) {
		TESTD *d = hashmap.find(*(keyArray[x]));
		ASSERT_TRUE(d != NULL);
		ASSERT_EQ(x, d->x);
	}

	ASSERT_TRUE(hashmap.removeAll());

	ASSERT_EQ(0,hashmap.size());
}

*/

INSTANTIATE_TEST_CASE_P(10Strings,
		StringTest,
		::testing::Values( 1, 10  ,20,22,24, 3 , 10  , 20, 1000   //));
				 ,10000  )); //, 20000 ));
//::testing::Values( 1, 10  ,20,22,24 ));


INSTANTIATE_TEST_CASE_P(10Ints,
		HashIntTest,
		::testing::Values( 10, 1000   //));
				 , 10000 ));

// FIXME - this test has some problem with malloc() - but I think its the test itself, not the containers

INSTANTIATE_TEST_CASE_P(10CStringTests,
		CStringHashTest,
		::testing::Values( 10 )); //, 1000   //));
			//	 , 10000 ));

INSTANTIATE_TEST_CASE_P(10CStringTests,
		CStringGenericHashTest,
		::testing::Values( 10 )); //, 1000   //));
			//	 , 10000 ));


int main(int argc, char **argv) {
	::testing::InitGoogleTest(&argc, argv);
	return RUN_ALL_TESTS();
}
