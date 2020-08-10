/*
 * test_twarray.cpp
 *
 *  Created on: Nov 28, 2011
 *      Author: ed
 * (c) 2011, WigWag LLC
 */


#include <string>
#include <gtest/gtest.h>

#include <malloc.h>

#include "tests/testutils.h"
#include <TW/tw_defs.h>
#include <TW/tw_types.h>
#include <TW/tw_log.h>
#include <TW/tw_bufblk.h>
#include <TW/tw_fifo.h>
#include <TW/tw_list.h>

#include <iostream>


using namespace std;

using namespace TWlib;

using ::testing::TestWithParam;
using ::testing::Values;



namespace TWlibTests {

typedef TWlib::Allocator<Alloc_Std> TESTAlloc;

typedef LList<int,TESTAlloc> TestList;


class LListTest : public ::testing::TestWithParam<int> {
protected:
	// Objects declared here can be used by all tests in the test case for Foo.

	TESTAlloc *_alloc;
 public:

  // You can remove any or all of the following functions if its body
  // is empty.

	  LListTest() :
   _alloc(NULL) {
    // setup work for test
		  _alloc = new TESTAlloc();
	  }

  virtual ~LListTest() {
    // You can do clean-up work that doesn't throw exceptions here.
	  delete _alloc;
  }

  // If the constructor and destructor are not enough for setting up
  // and cleaning up each test, you can define the following methods:

  virtual void SetUp() {
    // Code here will be called immediately after the constructor (right
    // before each test).
//	  TestZMemCollLayout *l = GetParam();
//	  l->
  }


  virtual void TearDown() {
    // Code here will be called immediately after each test (right
    // before the destructor).
  }

};

}

using namespace TWlibTests;



TEST_P(LListTest, CreateTest) {
	LList<int,TESTAlloc> list1;
	int L = GetParam();
	LList<int,TESTAlloc> list2(_alloc);

	for(int x=0;x<L;x++) {
		list1.addToHead(x);
	}
	int ans;
	ASSERT_TRUE(list1.peekHead(ans));
	ASSERT_EQ(ans,L-1);

	ASSERT_TRUE(list1.peekTail(ans));
	ASSERT_EQ(ans,0);


}


TEST_P(LListTest, TailTest) {
	LList<int,TESTAlloc> list1;
	int L = GetParam();
	LList<int,TESTAlloc> list2(_alloc);

	for(int x=0;x<L;x++) {
		list1.addToTail(x);
	}
	int ans;
	ASSERT_TRUE(list1.peekTail(ans));
	ASSERT_EQ(ans,L-1);

	ASSERT_TRUE(list1.peekHead(ans));
	ASSERT_EQ(ans,0);

}


TEST_P(LListTest, IterTest) {
	LList<int,TESTAlloc> list1;
	int L = GetParam();
	LList<int,TESTAlloc> list2(_alloc);

	for(int x=0;x<L;x++) {
		list1.addToTail(x);
	}
	int ans;

	LList<int,TESTAlloc>::iter titer;
	list1.startIterHead(titer);

	int y = 0; int z;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getCurrent(z));
		ASSERT_EQ(z,y);
		ASSERT_TRUE(titer.getNext(z));
		ASSERT_EQ(z,y);
		y++;
		cout << "*";
	}
	cout << endl;

	list1.startIterTail(titer);

	y = L-1;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getCurrent(z));
		ASSERT_EQ(z,y);
		ASSERT_TRUE(titer.getPrev(z));
		ASSERT_EQ(z,y);
		y--;
	}

	ASSERT_TRUE(list1.peekTail(ans));
	ASSERT_EQ(ans,L-1);

	ASSERT_TRUE(list1.peekHead(ans));
	ASSERT_EQ(ans,0);

}

TEST_P(LListTest, EndTest) {
	LList<int,TESTAlloc> list1;
	int L = GetParam();
	LList<int,TESTAlloc> list2(_alloc);

	for(int x=2;x<L+2;x++) {
		list1.addToTail(x);
	}
	ASSERT_EQ(list1.remaining(),L);
	int ans;
	ASSERT_TRUE(list1.peekTail(ans));
	ASSERT_EQ(ans,L+1);

	ASSERT_TRUE(list1.peekHead(ans));
	ASSERT_EQ(ans,2);

	// add stuff to the ends...
	int z = 1;
	list1.addToHead(z);

	z = L + 2;
	list1.addToTail(z);

	ASSERT_EQ(list1.remaining(),L+2);


	LList<int,TESTAlloc>::iter titer;
	list1.startIterHead(titer);
	int y = 1;

	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getNext(z));
		ASSERT_EQ(z,y);
		y++;
	}

	list1.startIterTail(titer);
	y = L+2;

	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getPrev(z));
		ASSERT_EQ(z,y);
		y--;
	}

}


TEST_P(LListTest, IterTestRemoval) {
	LList<int,TESTAlloc> list1;
	int L = GetParam();
	LList<int,TESTAlloc> list2(_alloc);

	for(int x=0;x<L;x++) {
		list1.addToTail(x);
	}
	int ans;

	int oldsize = list1.remaining();

	LList<int,TESTAlloc>::iter titer;
	list1.startIterHead(titer);

	int y = 0; int z;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getCurrent(z));
		ASSERT_EQ(z,y);
		if(y==2) {
			ASSERT_TRUE(titer.removeNext(z)); // remove one element, goto next
		} else {
			ASSERT_TRUE(titer.getNext(z));
		}
		ASSERT_EQ(z,y);
		y++;
		cout << "*";
	}
	cout << endl;

	ASSERT_EQ(list1.remaining(), oldsize-1);

	list1.startIterTail(titer);

	y = L-1;
	while(!titer.atEnd()) {
		if(y == 2) {
			y--;
			continue;
		}
		ASSERT_TRUE(titer.getCurrent(z));
		ASSERT_EQ(z,y);
		ASSERT_TRUE(titer.getPrev(z));
		ASSERT_EQ(z,y);
		y--;
	}

	ASSERT_TRUE(list1.peekTail(ans));
	ASSERT_EQ(ans,L-1);

	ASSERT_TRUE(list1.peekHead(ans));
	ASSERT_EQ(ans,0);

}


TEST_P(LListTest, IterRemovalAll) {
	LList<int,TESTAlloc> list1;
	int L = GetParam();
	LList<int,TESTAlloc> list2(_alloc);

	for(int x=0;x<L;x++) {
		list1.addToTail(x);
	}
	int ans;
	ASSERT_EQ(list1.remaining(), L);

	int oldsize = list1.remaining();

	LList<int,TESTAlloc>::iter titer;
	list1.startIterHead(titer);

	int y = 0; int z;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getCurrent(z));
		ASSERT_EQ(z,y);
		ASSERT_TRUE(titer.removeNext(z)); // remove one element, goto next
		ASSERT_EQ(z,y);
		y++;
		cout << "*";
	}
	cout << endl;

	ASSERT_EQ(list1.remaining(), 0);

	// and add some back, to ensure list is still ok

	for(int x=0;x<L;x++) {
		list1.addToTail(x);
	}
	ASSERT_EQ(list1.remaining(), L);

}

TEST_P(LListTest, IterRemovalAllBackwards) {
	LList<int,TESTAlloc> list1;
	int L = GetParam();
	LList<int,TESTAlloc> list2(_alloc);

	for(int x=0;x<L;x++) {
		list1.addToTail(x);
	}
	int ans;
	ASSERT_EQ(list1.remaining(), L);

	int oldsize = list1.remaining();

	LList<int,TESTAlloc>::iter titer;
	list1.startIterTail(titer);

	int y = L-1; int z;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getCurrent(z));
		ASSERT_EQ(z,y);
		ASSERT_TRUE(titer.removePrev(z)); // remove one element, goto next
		ASSERT_EQ(z,y);
		y--;
		cout << "*";
	}
	cout << endl;

	ASSERT_EQ(list1.remaining(), 0);

	// and add some back, to ensure list is still ok

	for(int x=0;x<L;x++) {
		list1.addToTail(x);
	}
	ASSERT_EQ(list1.remaining(), L);

}

TEST_P(LListTest, CopyConstr) {
	LList<int,TESTAlloc> list1;
	int L = GetParam();


	for(int x=0;x<L;x++) {
		list1.addToTail(x);
	}
	int ans;
	ASSERT_EQ(list1.remaining(), L);

	LList<int,TESTAlloc>::iter titer;
	list1.startIterTail(titer);

	int y = L-1; int z;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getPrev(z));
		ASSERT_EQ(z,y);
		y--;
		cout << "*";
	}
	cout << endl;

	LList<int,TESTAlloc> list2(list1);

// now iter list2...
	list2.startIterTail(titer);

	y = L-1;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getPrev(z));
		ASSERT_EQ(z,y);
		y--;
		cout << "+";
	}
	cout << endl;

// set the numbers to something else...

	list2.startIterTail(titer);

	y = L-1;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.setCurrentVal(9999));
		ASSERT_TRUE(titer.getPrev(z));
		ASSERT_EQ(z,9999);
		y--;
		cout << "+";
	}
	cout << endl;

	// Assure old list not affected.

	list1.startIterTail(titer);
	y = L-1;
	while(!titer.atEnd()) {
		ASSERT_TRUE(titer.getPrev(z));
		ASSERT_EQ(z,y);
		y--;
		cout << "*";
	}
	cout << endl;

}




/*

TEST_P(DynArrayTest, AssignTest) {
	DynArray<int,TESTAlloc> array1;
	int L = GetParam();
	DynArray<int,TESTAlloc> array2(L);

	for(int x = 0; x< L; x++ )
		array2.put(x,x);

	int a;
	for(int x = 0; x< L; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(x,a);
	}

	ASSERT_EQ(array1.size(),0);
	array1 = array2;
	ASSERT_EQ(array1.size(),array2.size());

	for(int x = 0; x< L; x++ ) {
		ASSERT_TRUE(array1.get(x,a));
		ASSERT_EQ(x,a);
	}
}

TEST_P(DynArrayTest, CopyConstr) {
//	DynArray<int,TESTAlloc> array1;
	int L = GetParam();
	DynArray<int,TESTAlloc> array2(L);

	for(int x = 0; x< L; x++ )
		array2.put(x,x);

	int a;
	for(int x = 0; x< L; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(x,a);
	}

	DynArray<int,TESTAlloc> array1(array2);
	ASSERT_EQ(array1.size(),array2.size());

	for(int x = 0; x< L; x++ ) {
		ASSERT_TRUE(array1.get(x,a));
		ASSERT_EQ(x,a);
	}
}


TEST_P(DynArrayTest, ResizeTest) {
	DynArray<int,TESTAlloc> array1;
	int L = GetParam();
	DynArray<int,TESTAlloc> array2(L);

	for(int x = 0; x< L; x++ )
		array2.put(x,x);

	int a;
	for(int x = 0; x< L; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(x,a);
	}

	array2.resize(L + 10);

	for(int x = L; x < (L+10); x++ ) {
		ASSERT_TRUE(array2.putByRef(x,x));
	}

	for(int x = 0; x < L+10; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(x,a);
	}
}

TEST_P(DynArrayTest, AddToEnd) {
	DynArray<int,TESTAlloc> array1(10);
	int L = GetParam();
	DynArray<int,TESTAlloc> array2(L);

	for(int x = 0; x< L; x++ )
		array2.put(x,x);

	int a;
	for(int x = 0; x< L; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(x,a);
	}


	for(int x = 0; x < 10; x++ ) {
		ASSERT_TRUE(array1.put(x,L+x));
	}

	array2.addToEnd(array1);
	ASSERT_EQ(array2.size(),L+10);

	for(int x = 0; x < L+10; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(x,a);
	}
}

TEST_P(DynArrayTest, InsertAtLoc) {
	DynArray<int,TESTAlloc> array1(10);
	int L = GetParam();
	DynArray<int,TESTAlloc> array2(L);

	int x;
	for(x = 0; x< L; x++ )
		array2.put(x,x);

	int a;
	for(x = 0; x< L; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(x,a);
	}

	const int fixval = 99999;

	for(x = 0; x < 10; x++ ) {
		ASSERT_TRUE(array1.put(x,fixval));
	}

	array2.insert(0,array1);
	ASSERT_EQ(array2.size(),L+10);

	for(x = 10; x < L+10; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(a,x-10);
	}

	for(x = 0; x < 10; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(a,fixval);
	}

}


TEST_P(DynArrayTest, InsertAtLocEnd) {
	DynArray<int,TESTAlloc> array1(10);
	int L = GetParam();
	DynArray<int,TESTAlloc> array2(L);

	int x;
	for(x = 0; x< L; x++ )
		array2.put(x,x);

	int a;
	for(x = 0; x< L; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(x,a);
	}

	const int fixval = 99999;

	for(x = 0; x < 10; x++ ) {
		ASSERT_TRUE(array1.put(x,fixval));
	}

	array2.insert(L,array1);
	ASSERT_EQ(array2.size(),L+10);

	for(x = 0; x < L; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(a,x);
	}

	for(x = L; x < L+10; x++ ) {
		ASSERT_TRUE(array2.get(x,a));
		ASSERT_EQ(a,fixval);
	}

}


*/


//DynArray<int,TESTAlloc> array1;

INSTANTIATE_TEST_CASE_P(BasicTests,
		LListTest,
		::testing::Values( 10
				));




int main(int argc, char **argv) {
	::testing::InitGoogleTest(&argc, argv);
	return RUN_ALL_TESTS();
}
