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
#include <TW/tw_array.h>

#include <iostream>


using namespace std;

using namespace TWlib;

using ::testing::TestWithParam;
using ::testing::Values;



namespace TWlibTests {

typedef TWlib::Allocator<Alloc_Std> TESTAlloc;

typedef DynArray<int,TESTAlloc> TestArray;


class DynArrayTest : public ::testing::TestWithParam<int> {
protected:
	// Objects declared here can be used by all tests in the test case for Foo.

	TESTAlloc *_alloc;
 public:

  // You can remove any or all of the following functions if its body
  // is empty.

	  DynArrayTest() :
   _alloc(NULL) {
    // setup work for test
		  _alloc = new TESTAlloc();
	  }

  virtual ~DynArrayTest() {
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



TEST_P(DynArrayTest, CreateTest) {
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

}


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





DynArray<int,TESTAlloc> array1;

INSTANTIATE_TEST_CASE_P(BasicTests,
		DynArrayTest,
		::testing::Values( 10, 100, 1000
				));




int main(int argc, char **argv) {
	::testing::InitGoogleTest(&argc, argv);
	return RUN_ALL_TESTS();
}
