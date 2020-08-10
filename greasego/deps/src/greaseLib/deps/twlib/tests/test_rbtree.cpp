/*
 * simple_khashtest.c
 *
 *  Created on: Mar 1, 2012
 *      Author: ed
 * (c) 2012, WigWag LLC
 */


#include <TW/tw_hashcommon.h>
#include <TW/tw_khash.h>
#include <TW/tw_rbtree.h>
#include <TW/tw_sema.h>
#include <TW/tw_utils.h>


#include <iostream>
#include <string>
#include <sstream>


//KHASH_MAP_INIT_INT(32, char); // a hash map with int as key, and char as value

using namespace TWlib;
using namespace std;

class TESTD {
public:
	int x;
	TESTD() {
		cout << "cstor." << endl;
	}
	TESTD(TESTD &d) : x(d.x) {
		cout << "cstor(" << x << ")" << endl;
	}
	TESTD &operator=(const TESTD &o) {
		x = o.x;
		cout << "operator=(" << o.x << ")" << endl;
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

struct string_cmpP {
int operator()(const string *l, const string *r) {
	return l->compare(*r);
}
};
struct string_eqstrPP {
	  inline int operator() (const string * const *l,
	                  const string * const *r) const
	  {
		  return ((*l)->compare(**r) == 0);
	  }
};

struct string_cmpPP {
int operator()(const string **l, const string **r) {
	return (*l)->compare(**r);
}
};

namespace TWlib {
	template<>
	struct tw_hash<TESTD *> {
		inline size_t operator()(const TESTD *v) const {
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
}



static const int loop = 10;

int main() {
//	TW_KHash_32<string, TESTD, TW_Mutex, string_eqstrP, TWlib::Allocator<Alloc_Std> > hashmap;
//	TW_KHash_32<string *, TESTD, TW_Mutex, string_eqstrPP, TWlib::Allocator<Alloc_Std> > hashmap2;

	TWlib::RB_Tree<string *, string_cmpP, TWlib::Allocator<TWlib::Alloc_Std> > tree1;

	/*	int ret, is_missing;
	khiter_t k;
	khash_t(32) *h = kh_init(32); // initializes the hash table. (khash_t(32) is type kh_32_t)

	k = kh_put(32, h, 5, &ret); // this creates key (int) 5, table h, of type '32' (yes, this is bizarre api design) - this returns an iterator
	if (!ret) kh_del(32, h, k);
	kh_value(h, k) = 10;
	k = kh_get(32, h, 10);
	is_missing = (k == kh_end(h));
	k = kh_get(32, h, 5);
	kh_del(32, h, k);
	for (k = kh_begin(h); k != kh_end(h); ++k)
		if (kh_exist(h, k)) kh_value(h, k) = 1;
	kh_destroy(32, h);
	return 0;
	*/

	string test_strs[loop];


	ostringstream out;

	for(int x=0;x<loop;x++ ) {
		out.str(test_strs[x]);
		out << "Val " << x;
		test_strs[x] = out.str();
//		hashmap.addNoreplace(test_strs[x],testd);
		string *temp = &test_strs[x];
		tree1.insert(temp);
		cout << "Added string \""<< test_strs[x] << "\"" << endl;
	}

	cout << "Lookup test." << endl;

	for(int x=0;x<loop;x++ ) {
		cout << test_strs[x] << endl;
		string *temp = &test_strs[x];
		string *s = tree1.find(temp);
		if(s) {
			cout << "Have string: " << *s << endl;
		} else
			cout << "NULL" << endl;
	}

	TWlib::RB_Tree<string *, string_cmpP, TWlib::Allocator<TWlib::Alloc_Std> >::Iter iter1(tree1);

	iter1.startMin();

	string *s2 = iter1.current();
	while(s2) {
		cout << "Iter next: " << *s2 << endl;
		s2 = iter1.getNext();
	}

	iter1.startMax();
	cout << "Iter from max..." << endl;
	s2 = iter1.current();
	while(s2) {
		cout << "Iter next: " << *s2 << endl;
		s2 = iter1.getPrev();
	}

	cout << "Remove odd number strings..." << endl;

	for(int x=0;x<loop;x++ ) {
		string *s3 = NULL;
		if((x % 2) > 0) {
			string *temp = &test_strs[x];
			s3 = tree1.remove(temp);
		}
		if(s3) {
			cout << "Removed " << *s3 << endl;
		}
	}

	iter1.startMin();

	s2 = iter1.current();
	while(s2) {
		cout << "Iter next: " << *s2 << endl;
		s2 = iter1.getNext();
	}

	iter1.startMax();
	cout << "Iter from max..." << endl;
	s2 = iter1.current();
	while(s2) {
		cout << "Iter next: " << *s2 << endl;
		s2 = iter1.getPrev();
	}


}


