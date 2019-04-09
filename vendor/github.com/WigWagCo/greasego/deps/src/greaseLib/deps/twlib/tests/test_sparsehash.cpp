// WigWag LLC
// (c) 2010
// testzsparsehash.cpp
// Author: ed

#include <iostream>

#include <TW/tw_sparsehash.h>
#include <TW/tw_log.h>

// for strings and testomg
#include <string>
#include <iostream>
#include <locale>


#define NUMSTRINGS1 10
#define NUMDSRNS1 14


using namespace std;
using namespace TWlib;

namespace TWlibTESTS {

typedef Allocator<Alloc_Std> TESTAlloc;

char *teststrs1[] = { // 10 strings
		"AAA",
		"AAAB",
		"AAABC",
		"AAABCD",
		"AAAAAA",
		"6ABABAB",
		"7ABAVAV",
		"8ABABAB",
		"10ZZZAK",
		"ZZZZZZ"
};

char *dsrns1[] = {
		"ds.test.one:rn.one.two",
		"ds.test.one:rn.one",
		"ds.test.one:rn",
		"ds.test.one:rn.one.3",
		"ds.test.one:rn.one.4",
		"ds.test.one:rn.one.5",
		"ds.test.one:rn.one.6",
		"ds.test.one:rn.one.7",
		"ds.test.one:rn.one.8",
		"ds.test.one:rn.one.9",
		"ds.test.one:rn.one.10",
		"ds.other.one:rn.one.two",
		"ds.other.one:rn.one.three",
		"ds.other.one:rn.one.four",
};

char *emptykey = "~#~@@#~#!"; // something invalid for a DSRN

class testdat {
public:
	testdat() { x = -1; }
	~testdat() { }
	testdat( testdat& d ) { x = d.x; }
	testdat& operator=(const testdat &o) { this->x = o.x; return *this; }
	int x;
};

string *zemptykey;
string *zstrs1[NUMSTRINGS1];
string *zstrdsrns1[NUMDSRNS1];
testdat *dat1[NUMSTRINGS1];


int loop1 =1;
int outerloop = 10000;

struct string_eqstr {
	inline bool operator()(const string *s1, const string *s2) const {
		if((s1 != NULL) && (s2 != NULL)) {
			if(s1->compare(*s2) == 0)
				return true;
			else
				return false;
		}
		return true;
	}
};

struct hashStdString {
	static locale loc;
	static const collate<char>& coll;
	static size_t hash(const std::string *s) {
		return (size_t) coll.hash(s->data(), s->data() + s->length()); // get hash on whole string
	}
};

} // end namespace TWlibTESTS

namespace TWlib { // specialize a corresponding hash function for the TWSparseHash below, needs to be defined in TWlib namespace
template<>
struct tw_hash<std::string> {
	inline size_t operator()(const std::string* s) const {
		return (size_t) TWlibTESTS::hashStdString::hash(s);
	}
};
}

locale TWlibTESTS::hashStdString::loc; // must put this first...
const collate<char>& TWlibTESTS::hashStdString::coll = use_facet<collate<char> >(loc); // then this... (order of init)


using namespace TWlibTESTS;

int main() {
	cout << "Test ZSparseHash" << endl;
	zemptykey = new string(emptykey);
//	string zemptykey2(emptykey, Z_DEFAULT_CP );
//	ZString zemptykey;
//	zemptykey = zemptykey2;
//	zemptykey = emptykey
	// stack test
	TWSparseHash<string, testdat, TW_Mutex, TWlibTESTS::string_eqstr, TESTAlloc> testhash( *zemptykey );

	// heap test
	TWSparseHash<string, testdat, TW_Mutex, TWlibTESTS::string_eqstr, TESTAlloc> *heaphash;

	int x;
	for (x=0;x<NUMSTRINGS1;x++) {
		zstrs1[x] = new string(teststrs1[x]);
	}

	for (x=0;x<NUMSTRINGS1;x++) {
		testdat *d = testhash.addReplaceNew( *zstrs1[x] );
		d->x = x;
	}

	testdat fill;

	for (x=0;x<NUMSTRINGS1;x++) {
		if(testhash.find( *zstrs1[x], fill ))
			cout << "found " << *zstrs1[x] << " = " << fill.x << endl;
		else
			cout << "!!didnt find: " << *zstrs1[x] << endl;
	}

	cout << "----" << endl;

	for (x=0;x<NUMSTRINGS1;x+=2) { // now remove even ones
		if(testhash.remove( *zstrs1[x], fill )) {
			cout << "deleting: " << *zstrs1[x] << " = " << fill.x << endl;

		}
	}

	for (x=0;x<NUMSTRINGS1;x++) {
		if(testhash.find( *zstrs1[x], fill ))
			cout << "found " << *zstrs1[x] << " = " << fill.x << endl;
		else
			cout << "!!didnt find: " << *zstrs1[x] << endl;
	}

	cout << "iterate forward..." << endl;
	TWSparseHash<string, testdat, TW_Mutex, TWlibTESTS::string_eqstr, TESTAlloc>::HashIterator iter(testhash);
//	testhash.gotoStart(iter); // no longer needed - we call this in the ZHashIterator constructor
	do {
		cout << "key: " << *(iter.key()) << " data: " << iter.data()->x << endl;
	} while(iter.getNext());
	testhash.releaseIter();


	for (x=0;x<NUMSTRINGS1;x++) {
		delete zstrs1[x];
	}


/*	cout << "iterate backward..." << endl;
	ZSparseHash<ZString, testdat, ACE_Thread_Mutex, ZStrP_eqstr>::ZHashIterator iter2(testhash);
	testhash.gotoEnd(iter2);
	do {
		cout << "key: " << iter2.key()->toCPPString() << " data: " << iter2.data()->x << endl;
	} while(iter2.getPrev());
	testhash.releaseIter();
*/




	testdat fill2;

	int loop, loopb, loopc;
	TWSparseHash<string, testdat, TW_Mutex, TWlibTESTS::string_eqstr, TESTAlloc> *heaphashes[outerloop];

	cout << "-------Heap test - " << loop1 << " runs-----" << endl;

	for(loopb = outerloop-1; loopb >= 0; loopb--)
	for(loop = loop1-1; loop >= 0;loop--) {
		heaphashes[loopb] = new TWSparseHash<string, testdat, TW_Mutex, TWlibTESTS::string_eqstr, TESTAlloc>( *zemptykey );


		for (x=0;x<NUMSTRINGS1;x++) {
			zstrs1[x] = new string(teststrs1[x] );
		}

		for (x=0;x<NUMSTRINGS1;x++) {
			testdat *d = heaphashes[loopb]->find( *zstrs1[x] );
		}


		for (x=0;x<NUMSTRINGS1;x++) {
			testdat *d = NULL;
			if((x % 2) > 0)
				d = heaphashes[loopb]->addReplaceNew( *zstrs1[x] );
			else
				d = heaphashes[loopb]->addNoreplaceNew( *zstrs1[x] );
			if(d == NULL)
				cout << " ----------COULD NOT ADD !!!!!!!! -----------" << endl;
			else {
				cout << " added DATA" << endl;
				d->x = x+100;
			}
		}


		for (x=0;x<NUMSTRINGS1;x++) {
			if(heaphashes[loopb]->find( *zstrs1[x], fill2 ))
				cout << "found " << *zstrs1[x] << " = " << fill2.x << endl;
			else
				cout << "!!didnt find: " << *zstrs1[x] << endl;
		}

		cout << "----" << endl;

		for (x=0;x<NUMSTRINGS1;x+=2) { // now remove even ones
			if(heaphashes[loopb]->remove( *zstrs1[x], fill2 )) {
				cout << "deleting: " << *zstrs1[x] << " = " << fill2.x << endl;

			}
		}

		for (x=0;x<NUMSTRINGS1;x++) {
			if(heaphashes[loopb]->find( *zstrs1[x], fill2 ))
				cout << "found " << *zstrs1[x] << " = " << fill2.x << endl;
			else
				cout << "!!didnt find: " << *zstrs1[x] << endl;
		}

		for (x=0;x<NUMSTRINGS1;x++) {
			delete zstrs1[x];
		}


	}

	for(loopb = outerloop-1; loopb >= 0; loopb--)
		delete heaphashes[loopb];

/*

	heaphash
			= new ZSparseHash<ZString, testdat, ACE_Thread_Mutex, ZStrP_eqstr> (
					zemptykey);

	for (x = 0; x < NUMSTRINGS1; x++) {
		zstrs1[x] = new ZString(teststrs1[x], Z_DEFAULT_CP);
	}

	for (x = 0; x < NUMSTRINGS1; x++) {
		testdat *d = heaphash->addReplaceNew(*zstrs1[x]);
		d->x = x;
	}

	heaphash->addReplaceNew( zemptykey ); // should fail

	for (x = 0; x < NUMSTRINGS1; x++) {
		if (heaphash->find(*zstrs1[x], fill2))
			cout << "found " << zstrs1[x]->toCPPString() << " = " << fill2.x
					<< endl;
		else
			cout << "!!didnt find: " << zstrs1[x]->toCPPString() << endl;
	}

	cout << "----" << endl;

	for (x = 0; x < NUMSTRINGS1; x += 2) { // now remove even ones
		if (heaphash->remove(*zstrs1[x], fill2)) {
			cout << "deleting: " << zstrs1[x]->toCPPString() << " = "
					<< fill2.x << endl;

		}
	}

	for (x = 0; x < NUMSTRINGS1; x++) {
		if (heaphash->find(*zstrs1[x], fill2))
			cout << "found " << zstrs1[x]->toCPPString() << " = " << fill2.x
					<< endl;
		else
			cout << "!!didnt find: " << zstrs1[x]->toCPPString() << endl;
	}

	for (x = 0; x < NUMSTRINGS1; x++) {
		delete zstrs1[x];
	}

	delete heaphash;


*/



/** ZDSRNSparseSupermap TEST - -----------------------------------
	cout << "the end. isnt that special?" << endl;

	cout << "-----------" << endl;

	cout << "test ZDSRNSparseSupermap<DATA,LOCK>" << endl;

	ZDSRNSparseSupermap<testdat,ACE_Null_Mutex> *testSuper;


	for(loopb = outerloop-1; loopb >= 0; loopb--)   {

	testSuper = new ZDSRNSparseSupermap<testdat,ACE_Null_Mutex>( zemptykey );

	for (x=0;x<NUMDSRNS1;x++) {
		zstrdsrns1[x] = new ZString(dsrns1[x], Z_DEFAULT_CP );
	}

	for (x=0;x<NUMDSRNS1;x++) {
		testdat *datp;
		cout << "adding: " << zstrdsrns1[x]->toCPPString() << endl;
		datp = testSuper->newOrGet( *zstrdsrns1[x] );
		if(datp) {
			datp->x = x + 100;
		} else
			cout << "FAIL - Error - NULL pointer to a testdat" << endl;
	}

	for (x=0;x<NUMDSRNS1;x++) {
		if(testSuper->findExplicitPath(*zstrdsrns1[x], fill)) {
			cout << "found: " << zstrdsrns1[x]->toCPPString() << " x= " << fill.x << endl;
		} else {
			cout << "FAIL - error - could not find: " << zstrdsrns1[x]->toCPPString() << endl;
		}
	}

	ZDSRNSparseSupermap<testdat,ACE_Null_Mutex>::PairList pairlist;
	ZDSRNSparseSupermap<testdat,ACE_Null_Mutex>::ListPair pair;


	testSuper->findFullPathPair(*zstrdsrns1[0],pairlist);

	while(pairlist.remove(pair)) {
		cout << "list - path: " << pair.path.toCPPString() << ", dat: x=" << pair.dat->x << endl;
	}

	ZDSRNSparseSupermap<testdat,ACE_Null_Mutex>::DataList datalist;
	testdat *d;

	testSuper->findFullPath(*zstrdsrns1[0],datalist);

	while(datalist.remove(d)) {
		cout << "list - dat: x=" << d->x << endl;
	}

	for (x=0;x<NUMDSRNS1;x++) {
		delete zstrdsrns1[x];
	}


	delete testSuper;

	}
END ZDSRNSparseSupermap TEST */



	// cleanup




}
