/*
 * regr_tw_bufblk.cpp
 *
 *  Created on: Nov 15, 2011
 *      Author: ed
 * (c) 2011 Framez, LLC
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

#include <iostream>

using namespace std;

using namespace TWlib;

using ::testing::TestWithParam;
using ::testing::Values;

namespace TWlibTests {

// max of about 10k of memory
const tw_size MAX_TEST_BUF_SZ = 1000;
const tw_size MAX_TEST_BUFS = 100;



struct TestBuf {
char buf[MAX_TEST_BUF_SZ];
tw_size size;
};

template <class A>
class TWMemBlkBufData {
public:
	struct TestBuf *bufs[MAX_TEST_BUFS];
	tw_size numBufs;
	int repeats; // use in TWBufBlkRepeatTest
	TWMemBlkBufData() : numBufs( 0 ), repeats( 1 ) {
		for(int x=0;x<MAX_TEST_BUFS;x++) {
			bufs[x] = NULL;
		}
	}
	~TWMemBlkBufData() {
		for(int x=0;x<MAX_TEST_BUFS;x++) {
			if(bufs[x]) // cleanup
				delete bufs[x];
		}
	}

	// returns a single chain, with a single BugBlk link for every
	// test TestBuf data block.
	TWlib::BufBlk<A> *makeOneBigChain() {
		BufBlk<A> *ret = NULL;
		BufBlk<A> *look = NULL;
		for(int y=0;y<repeats;y++)
		for(int x=0;x<MAX_TEST_BUFS;x++) {
			if(bufs[x]) {
				if(!ret) {
					ret = new BufBlk<A>(bufs[x]->size);
					look = ret;
				} else {
					look->setNexblk(new BufBlk<A>(bufs[x]->size));
					look = look->nexblk();
				}
				look->copyFrom(bufs[x]->buf,bufs[x]->size);
			}
		}
		return ret;
	}

	char *makeOneBigBlock(char *&bigbuf, int &size ) {
		BufBlk<A> *ret = NULL;
		BufBlk<A> *look = NULL;
		size = 0;
		for(int x=0;x<MAX_TEST_BUFS;x++) {
			if(bufs[x]) {
				size += bufs[x]->size;
			}
		}
		bigbuf = (char *) malloc(size * repeats);
		char *walk = bigbuf;
		for(int y=0;y<repeats;y++)
		for(int x=0;x<MAX_TEST_BUFS;x++) {
			if(bufs[x]) {
				memcpy(walk,bufs[x]->buf,bufs[x]->size);
				walk+=bufs[x]->size;
			}
		}
		size = size * repeats;
		return bigbuf;
	}

};

template <class A>
class MemBlkTestMaker {
public:
	static TWMemBlkBufData<A> *makeStringData(int n, ... ) {
		TWMemBlkBufData<A> *l = new TWMemBlkBufData<A>();
		l->numBufs = n;
		va_list list;
		va_start( list, n );
		for(int x=0;x<n;x++) {
			char *s = va_arg( list, char * );
			int c = strlen(s);
			c = (c >= MAX_TEST_BUF_SZ) ? MAX_TEST_BUF_SZ - 1 : c;
			l->bufs[x] = new struct TestBuf;
			memcpy(l->bufs[x]->buf,s,c);
			*(l->bufs[x]->buf + c) = '\0';
			l->bufs[x]->size = c + 1;
		}
		va_end( list );
		return l;
	}

	static TWMemBlkBufData<A> *makeStringRepeatData(int repeat, int n, ... ) {
		TWMemBlkBufData<A> *l = new TWMemBlkBufData<A>();
		l->numBufs = n;
		va_list list;
		va_start( list, n );
		for(int x=0;x<n;x++) {
			char *s = va_arg( list, char * );
			int c = strlen(s);
			c = (c >= MAX_TEST_BUF_SZ) ? MAX_TEST_BUF_SZ - 1 : c;
			l->bufs[x] = new struct TestBuf;
			memcpy(l->bufs[x]->buf,s,c);
			*(l->bufs[x]->buf + c) = '\0';
			l->bufs[x]->size = c + 1;
		}
		va_end( list );
		l->repeats = repeat;
		return l;
	}


};


/**
 * A test class with a consumer / producer thread set
 */
template <class A>
class TWBufTwoThreadTest : public ::testing::TestWithParam<TWMemBlkBufData<A> *> {
protected:
	// Objects declared here can be used by all tests in the test case for Foo.
	  A *_alloc;

 public:
  // You can remove any or all of the following functions if its body
  // is empty.

  TWBufTwoThreadTest() :
   _alloc(NULL) {
    // setup work for test
		  _alloc = new A();
	  }

  virtual ~TWBufTwoThreadTest() {
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


// TEST TYPES
typedef TWlib::Allocator<Alloc_Std> TESTAlloc;
typedef TWMemBlkBufData<TESTAlloc> TEST_DATA;

// TEST CASES


class TWBufBlkBasicTest : public ::testing::TestWithParam<TEST_DATA *> {
protected:
	// Objects declared here can be used by all tests in the test case for Foo.
	  TESTAlloc *_alloc;
	  // a memory block with all the memory in this test, contiguous.
	  char *big_block;
	  int big_block_size;
 public:
  // You can remove any or all of the following functions if its body
  // is empty.

  TWBufBlkBasicTest() :
   _alloc(NULL), big_block(NULL), big_block_size( 0 ) {
    // setup work for test
		  _alloc = new TESTAlloc();
	  }

  virtual ~TWBufBlkBasicTest() {
    // You can do clean-up work that doesn't throw exceptions here.
	  delete _alloc;
  }

  // If the constructor and destructor are not enough for setting up
  // and cleaning up each test, you can define the following methods:

  virtual void SetUp() {
    // Code here will be called immediately after the constructor (right
    // before each test).
	  TEST_DATA *l = const_cast<TEST_DATA *>(GetParam());
	  big_block = l->makeOneBigBlock(big_block,big_block_size);
  }


  virtual void TearDown() {
    // Code here will be called immediately after each test (right
    // before the destructor).
	  if(big_block)
		  free(big_block);
  }

};

/**
 * Just a test which makes a much larger block of memory, by repeating the block X number of times.
 */
class TWBufBlkRepeatTest : public TWBufBlkBasicTest {
protected:
	  int num_of_times;
 public:
  // You can remove any or all of the following functions if its body
  // is empty.

  TWBufBlkRepeatTest() :
	  TWBufBlkBasicTest(), num_of_times(0) {
	  }

  virtual ~TWBufBlkRepeatTest() {
    // You can do clean-up work that doesn't throw exceptions here.
  }

  // If the constructor and destructor are not enough for setting up
  // and cleaning up each test, you can define the following methods:

  virtual void SetUp() {
    // Code here will be called immediately after the constructor (right
    // before each test).
	  TWBufBlkBasicTest::SetUp();
	  TEST_DATA *l = const_cast<TEST_DATA *>(GetParam());
	  num_of_times = l->repeats;
  }

/*
  virtual void TearDown() {
    // Code here will be called immediately after each test (right
    // before the destructor).
	  if(big_block)
		  free(big_block);
  }
*/
};

typedef TWBufBlkBasicTest BasicTest;
typedef TWBufBlkRepeatTest RepeatTest;


TEST_P(BasicTest, SimpleWalkCompare) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	char *chaincopy = (char *) _alloc->malloc(head->total_length());

	BufBlk<TESTAlloc> *look = head;
	char *walkcopy = chaincopy;
	while(look) {
		memcpy(walkcopy,look->rd_ptr(), look->length());
		walkcopy += look->length();
		look = look->nexblk();
	}

	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	head->release();

}


TEST_P(BasicTest, SimpleCopyNextChunks) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	char *chaincopy = (char *) _alloc->malloc(head->total_length());
	int copyleft = head->total_length();
	int out = 0;
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *head );
	char *walkcopy = chaincopy;
	while(it->copyNextChunks( walkcopy,copyleft,out)) {
		copyleft -= out;
		walkcopy += out;
	}

	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	head->release();

	delete it; // yes, you can do this. The iterator has a reference to the BufBlk

	_alloc->free(chaincopy);
}

TEST_P(BasicTest, SimpleCopyNextChunks10) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;
	cout << "big_block_size " << big_block_size << endl;
	char *chaincopy = (char *) _alloc->malloc(head->total_length());
	int copyleft = head->total_length();
	int out = 0;
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *head );
	char *walkcopy = chaincopy;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;
	while(it->copyNextChunks(walkcopy,10,out)) {  // same as above, but asks for chunks of 10 bytes.
//		cout << "Out was: " << out << endl;
		ASSERT_TRUE(out <= 10) << "Copied too much.";
		copyleft -= out;
		walkcopy += out;
	}

	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	head->release();

	delete it; // yes, you can do this. The iterator has a reference to the BufBlk

	_alloc->free(chaincopy);
}

TEST_P(BasicTest, SimpleCopyNextChunks1) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;
	cout << "big_block_size " << big_block_size << endl;
	char *chaincopy = (char *) _alloc->malloc(head->total_length());
	int copyleft = head->total_length();
	int out = 0;
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *head );
	char *walkcopy = chaincopy;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;
	while(it->copyNextChunks(walkcopy,1,out)) {  // same as above, but asks for chunks of 10 bytes.
//		cout << "Out was: " << out << endl;
		ASSERT_TRUE(out <= 1) << "Copied too much.";
		copyleft -= out;
		walkcopy += out;
	}

	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	head->release();

	delete it; // yes, you can do this. The iterator has a reference to the BufBlk

	_alloc->free(chaincopy);
}

TEST_P(BasicTest, SimpleCopyNextChunks13) {
	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());
	BufBlk<TESTAlloc> *head = l->makeOneBigChain();
	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;
	cout << "big_block_size " << big_block_size << endl;
	char *chaincopy = (char *) _alloc->malloc(head->total_length());
	int copyleft = head->total_length();
	int out = 0;
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *head );
	char *walkcopy = chaincopy;
//	string temps;
//	cout << head->hexDump(temps,0) << endl;
	while(it->copyNextChunks(walkcopy,13,out)) {  // same as above, but asks for chunks of 10 bytes.
//		cout << "Out was: " << out << endl;
		ASSERT_TRUE(out <= 13) << "Copied too much.";
		copyleft -= out;
		walkcopy += out;
	}
	ASSERT_EQ(0,copyleft);
	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;
	head->release();
	delete it; // yes, you can do this. The iterator has a reference to the BufBlk
	_alloc->free(chaincopy);
}


TEST_P(BasicTest, SimpleGetNextChunk10) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;
	cout << "big_block_size " << big_block_size << endl;
	char *chaincopy = (char *) _alloc->malloc(head->total_length());
	int copyleft = head->total_length();
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *head );
	char *walkcopy = chaincopy;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;
	char *chunkcopy = NULL;
	int chunksize = 10;
	while(it->getNextChunk(chunkcopy,chunksize,10)) {  // same as above, but asks for chunks of 10 bytes.
		ASSERT_TRUE(chunksize <= 10) << "Copied too much.";
		ASSERT_TRUE(chunkcopy); // chunkcopy should not be NULL
		memcpy(walkcopy,chunkcopy,chunksize);
		copyleft -= chunksize;
		walkcopy += chunksize;
	}
	ASSERT_EQ(0,copyleft);
	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	head->release();

	delete it; // yes, you can do this. The iterator has a reference to the BufBlk

	_alloc->free(chaincopy);
}

TEST_P(BasicTest, SimpleGetNextChunk1) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;
//	cout << "big_block_size " << big_block_size << endl;
	char *chaincopy = (char *) _alloc->malloc(head->total_length());
	int copyleft = head->total_length();
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *head );
	char *walkcopy = chaincopy;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;
	char *chunkcopy = NULL;
	int chunksize = 10;
	while(it->getNextChunk(chunkcopy,chunksize,1)) {  // same as above, but asks for chunks of 10 bytes.
		ASSERT_TRUE(chunksize <= 1) << "Copied too much.";
		ASSERT_TRUE(chunkcopy); // chunkcopy should not be NULL
//		cout << "chunksize " << chunksize << endl;
		memcpy(walkcopy,chunkcopy,chunksize);
		copyleft -= chunksize;
		walkcopy += chunksize;
	}
	ASSERT_EQ(0,copyleft);
	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	head->release();

	delete it; // yes, you can do this. The iterator has a reference to the BufBlk

	_alloc->free(chaincopy);
}

TEST_P(BasicTest, SimpleGetNextChunk1DeepCopy) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	BufBlk<TESTAlloc> *headCopy = head->deepCopy();
	ASSERT_EQ(head->total_length(),headCopy->total_length()) << "Mismatch on total_length() orig versys deepCopy()." << endl;
	ASSERT_EQ(big_block_size,headCopy->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	head->release(); // dont need this guy now

//	cout << "big_block_size " << big_block_size << endl;
	char *chaincopy = (char *) _alloc->malloc(headCopy->total_length());
	int copyleft = headCopy->total_length();
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *headCopy );
	char *walkcopy = chaincopy;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;
	char *chunkcopy = NULL;
	int chunksize = 10;
	while(it->getNextChunk(chunkcopy,chunksize,1)) {  // same as above, but asks for chunks of 10 bytes.
		ASSERT_TRUE(chunksize <= 1) << "Copied too much.";
		ASSERT_TRUE(chunkcopy); // chunkcopy should not be NULL
//		cout << "chunksize " << chunksize << endl;
		memcpy(walkcopy,chunkcopy,chunksize);
		copyleft -= chunksize;
		walkcopy += chunksize;
	}
	ASSERT_EQ(0,copyleft);
	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	headCopy->release();

	delete it; // yes, you can do this. The iterator has a reference to the BufBlk

	_alloc->free(chaincopy);
}

TEST_P(BasicTest, SimpleGetNextChunk1DeepCopy2) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	BufBlk<TESTAlloc> *headCopy = head->deepCopy(TESTAlloc::getInstance());
	ASSERT_EQ(head->total_length(),headCopy->total_length()) << "Mismatch on total_length() orig versys deepCopy()." << endl;
	ASSERT_EQ(big_block_size,headCopy->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	head->release(); // dont need this guy now

//	cout << "big_block_size " << big_block_size << endl;
	char *chaincopy = (char *) _alloc->malloc(headCopy->total_length());
	int copyleft = headCopy->total_length();
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *headCopy );
	char *walkcopy = chaincopy;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;
	char *chunkcopy = NULL;
	int chunksize = 10;
	while(it->getNextChunk(chunkcopy,chunksize,1)) {  // same as above, but asks for chunks of 10 bytes.
		ASSERT_TRUE(chunksize <= 1) << "Copied too much.";
		ASSERT_TRUE(chunkcopy); // chunkcopy should not be NULL
//		cout << "chunksize " << chunksize << endl;
		memcpy(walkcopy,chunkcopy,chunksize);
		copyleft -= chunksize;
		walkcopy += chunksize;
	}
	ASSERT_EQ(0,copyleft);
	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	headCopy->release();

	delete it; // yes, you can do this. The iterator has a reference to the BufBlk

	_alloc->free(chaincopy);
}

TEST_P(BasicTest, SimpleGetContigBlock) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

//	char *chaincopy = (char *) _alloc->malloc(head->total_length());
	string temps;
	cout << head->hexDump(temps,0) << endl;

	BufBlk<TESTAlloc> *contig = NULL;
	ASSERT_TRUE(head->getContigBlock(contig,0,head->total_length()));
	cout << contig->hexDump(temps,0) << endl;

	ASSERT_EQ(0,memcmp(contig->rd_ptr(),big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	head->release();
	contig->release();
}


TEST_P(BasicTest, SimpleEat10Bytes) {
	const char *DATA_1 = "1234567890";

	const char *DATA_2 = "abcdefghjk";

	BufBlk<TESTAlloc> *blk1 = new BufBlk<TESTAlloc>(10);
	blk1->copyFrom(DATA_1,10);

	BufBlk<TESTAlloc> *blk2 = new BufBlk<TESTAlloc>(10);
	blk2->copyFrom(DATA_2,10);

	blk1->setNexblk(blk2);

	ASSERT_EQ(20,blk1->total_length()) << "totalsize mismatch" << endl;

	BufBlk<TESTAlloc> *newhead = blk1->eatBytes(0,10);

	ASSERT_EQ(blk2->base(), newhead->base());

	blk2->release();
}

TEST_P(BasicTest, SimpleEatLast10Bytes) {
	const char *DATA_1 = "1234567890";

	const char *DATA_2 = "abcdefghjk";

	BufBlk<TESTAlloc> *blk1 = new BufBlk<TESTAlloc>(10);
	blk1->copyFrom(DATA_1,10);

	BufBlk<TESTAlloc> *blk2 = new BufBlk<TESTAlloc>(10);
	blk2->copyFrom(DATA_2,10);

	blk1->setNexblk(blk2);

	ASSERT_EQ(20,blk1->total_length()) << "totalsize mismatch" << endl;

	BufBlk<TESTAlloc> *newhead = blk1->eatBytes(10,10);

	ASSERT_EQ(blk1->base(), newhead->base());
	ASSERT_EQ(NULL,blk1->nexblk());
	newhead->release();
}

TEST_P(BasicTest, SimpleEatLast11Bytes) {
	const char *DATA_1 = "1234567890";

	const char *DATA_2 = "abcdefghjk";

	BufBlk<TESTAlloc> *blk1 = new BufBlk<TESTAlloc>(10);
	blk1->copyFrom(DATA_1,10);

	BufBlk<TESTAlloc> *blkmid = new BufBlk<TESTAlloc>(10);
	blkmid->copyFrom(DATA_1,10);

	BufBlk<TESTAlloc> *blk2 = new BufBlk<TESTAlloc>(10);
	blk2->copyFrom(DATA_2,10);

	blk1->addToEnd(blkmid);
	blk1->addToEnd(blk2);

	ASSERT_EQ(30,blk1->total_length()) << "totalsize mismatch" << endl;

	BufBlk<TESTAlloc> *newhead = blk1->eatBytes(19,11);
	ASSERT_EQ(19, blk1->total_length());
	ASSERT_EQ(blk1->base(), newhead->base());
	ASSERT_EQ(NULL,newhead->nexblk()->nexblk());
	newhead->release();
}

TEST_P(BasicTest, SimpleEat11Bytes) {
	const char *DATA_1 = "1234567890";

	const char *DATA_2 = "abcdefghjk";

	BufBlk<TESTAlloc> *blk1 = new BufBlk<TESTAlloc>(10);
	blk1->copyFrom(DATA_1,10);

	BufBlk<TESTAlloc> *blk2 = new BufBlk<TESTAlloc>(10);
	blk2->copyFrom(DATA_2,10);

	blk1->setNexblk(blk2);

	ASSERT_EQ(20,blk1->total_length()) << "totalsize mismatch" << endl;

	BufBlk<TESTAlloc> *newhead = blk1->eatBytes(0,11);

	ASSERT_EQ(blk2->length(), 9);
	ASSERT_EQ(blk2->base(), newhead->base());

	newhead->release();
}

TEST_P(BasicTest, SimpleMiddle6Bytes) {
	const char *DATA_1 = "1234567890";

	const char *DATA_2 = "abcdefghjk";

	BufBlk<TESTAlloc> *blk1 = new BufBlk<TESTAlloc>(10);
	blk1->copyFrom(DATA_1,10);

	BufBlk<TESTAlloc> *blk2 = new BufBlk<TESTAlloc>(10);
	blk2->copyFrom(DATA_2,10);

	blk1->setNexblk(blk2);

	ASSERT_EQ(20,blk1->total_length()) << "totalsize mismatch" << endl;

	BufBlk<TESTAlloc> *newhead = blk1->eatBytes(7,6); // offset 7, eat 6

	ASSERT_EQ(7, blk2->length());
	ASSERT_EQ(7, blk1->length());
	ASSERT_EQ(14, blk1->total_length());
	ASSERT_EQ(blk1->base(), newhead->base()); // first block should still exist
	ASSERT_TRUE(!memcmp(DATA_1, blk1->rd_ptr(), 7));
	ASSERT_TRUE(!memcmp(DATA_2 + 3, blk2->rd_ptr(), 7));

	newhead->release();
}

TEST_P(BasicTest, SimpleMiddle6Bytes2) {
	const char *DATA_1 = "1234567890";

	const char *DATA_2 = "abcdefghjk";

	BufBlk<TESTAlloc> *blk1 = new BufBlk<TESTAlloc>(10);
	blk1->copyFrom(DATA_1,10);

	BufBlk<TESTAlloc> *blkmid = new BufBlk<TESTAlloc>(10);
	blkmid->copyFrom(DATA_1,10);


	BufBlk<TESTAlloc> *blk2 = new BufBlk<TESTAlloc>(10);
	blk2->copyFrom(DATA_2,10);

	blk1->addToEnd(blkmid);
	blk1->addToEnd(blk2);

	ASSERT_EQ(30,blk1->total_length()) << "totalsize mismatch" << endl;

	BufBlk<TESTAlloc> *newhead = blk1->eatBytes(7,16); // offset 7, eat 6

	ASSERT_EQ(7, blk2->length());
	ASSERT_EQ(7, blk1->length());
	ASSERT_EQ(14, blk1->total_length());
	ASSERT_EQ(blk1->base(), newhead->base()); // first block should still exist
	ASSERT_TRUE(!memcmp(DATA_1, blk1->rd_ptr(), 7));
	ASSERT_TRUE(!memcmp(DATA_2 + 3, blk1->nexblk()->rd_ptr(), 7));

	newhead->release();
}


TEST_P(BasicTest, SimpleMiddle10Bytes) {
	const char *DATA_1 = "1234567890";

	const char *DATA_2 = "abcdefghjk";
	const char *MID =    "**********";
	BufBlk<TESTAlloc> *blk1 = new BufBlk<TESTAlloc>(10);
	blk1->copyFrom(DATA_1,10);

	BufBlk<TESTAlloc> *blkmid = new BufBlk<TESTAlloc>(10);
	blkmid->copyFrom(MID,10);


	BufBlk<TESTAlloc> *blk2 = new BufBlk<TESTAlloc>(10);
	blk2->copyFrom(DATA_2,10);

	blk1->addToEnd(blkmid);
	blk1->addToEnd(blk2);

	ASSERT_EQ(30,blk1->total_length()) << "totalsize mismatch" << endl;

	BufBlk<TESTAlloc> *newhead = blk1->eatBytes(10,10); // offset 10, eat 10

	ASSERT_EQ(10, blk2->length());
	ASSERT_EQ(10, blk1->length());
	ASSERT_EQ(20, blk1->total_length());
	ASSERT_EQ(blk1->base(), newhead->base()); // first block should still exist
	ASSERT_TRUE(!memcmp(DATA_1, blk1->rd_ptr(), 10));
	ASSERT_TRUE(!memcmp(DATA_2, blk1->nexblk()->rd_ptr(), 10));

	newhead->release();
}

TEST_P(BasicTest, SimpleGetContigBlockInc1) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();
	head->inc_rd_ptr(1);// make the first link in chain 1 less in size

	ASSERT_EQ(big_block_size-1,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;

	BufBlk<TESTAlloc> *contig = NULL;
	ASSERT_TRUE(head->getContigBlock(contig,0,head->total_length()));
//	cout << contig->hexDump(temps,0) << endl;

	ASSERT_EQ(0,memcmp(contig->rd_ptr(),big_block+1,big_block_size-1)) << "Walk through memory was not the same" << endl;

	head->release();
	contig->release();
}

TEST_P(BasicTest, SimpleGetContigBlockOffset1) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;

	BufBlk<TESTAlloc> *contig = NULL;
	ASSERT_TRUE(head->getContigBlock(contig,1,head->total_length()-1));
//	cout << contig->hexDump(temps,0) << endl;

	ASSERT_EQ(0,memcmp(contig->rd_ptr(),big_block+1,big_block_size-1)) << "Walk through memory was not the same" << endl;

	head->release();
	contig->release();
}

TEST_P(BasicTest, ContigOfContigOffset1) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;

	BufBlk<TESTAlloc> *contig = NULL;
	ASSERT_TRUE(head->getContigBlock(contig,1,head->total_length()-1));
//	cout << contig->hexDump(temps,0) << endl;

	ASSERT_EQ(0,memcmp(contig->rd_ptr(),big_block+1,big_block_size-1)) << "Walk through memory was not the same" << endl;

	// OK, now build yet another contiguous block... but off the existing contigous block...
	// this one will be one less than original 'contig' which was 1 less than the control group.
	BufBlk<TESTAlloc> *contig2;
	ASSERT_TRUE(contig->getContigBlock(contig2,1,contig->total_length()-1));
	ASSERT_EQ(0,memcmp(contig2->rd_ptr(),contig->rd_ptr() +1,contig->total_length()-1)) << "Walk through memory was not the same" << endl;
	ASSERT_EQ(0,memcmp(contig2->rd_ptr(),big_block+2,big_block_size-2)) << "Walk through memory was not the same" << endl;


	head->release();
	contig->release();
}


TEST_P(BasicTest, SimpleGetContigBlockOffset13) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();
	if(head->total_length() > 13) {

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;

	BufBlk<TESTAlloc> *contig = NULL;
	ASSERT_TRUE(head->getContigBlock(contig,13,head->total_length()-13));
//	cout << contig->hexDump(temps,0) << endl;

	ASSERT_EQ(0,memcmp(contig->rd_ptr(),big_block+13,big_block_size-13)) << "Walk through memory was not the same" << endl;

	contig->release();

	} else {
		const ::testing::TestInfo* const test_info =
		  ::testing::UnitTest::GetInstance()->current_test_info();
		printf("      >>> Skipping test %s of testcase %d. Control group size was too small for test\n",
		       test_info->name(), test_info->test_case_name());
	}

	head->release();

}

/**
 * Test buffer navigation functions: rewind(), reset(), freespace(), length(), capacity(), inc_wr_ptr(), etc...
 */
TEST_P(BasicTest, NavigationResetRewind) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();
	head->inc_rd_ptr(1);// make the first link in chain 1 less in size
	ASSERT_EQ(big_block_size-1,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	const char *addthis = "12345";
	char *addbuf = (char *) _alloc->malloc(strlen(addthis) + 1);
	BufBlk<TESTAlloc> *addblk = new BufBlk<TESTAlloc>(addbuf, strlen(addthis) + 1, true);
	ASSERT_TRUE(addblk);
	addblk->copyFrom((void *) addthis,strlen(addthis) + 1);
	ASSERT_TRUE(addblk->length() == (strlen(addthis) + 1));
	ASSERT_TRUE(addblk->freespace() == 0);
	addblk->reset();
	addblk->inc_wr_ptr(strlen(addthis) + 1);
	ASSERT_TRUE(addblk->length() == (strlen(addthis) + 1));
	addblk->inc_rd_ptr(2);
	ASSERT_TRUE(addblk->length() == (strlen(addthis) - 1));
	addblk->reset();
	ASSERT_TRUE(addblk->length() == 0);
	ASSERT_TRUE(addblk->freespace() == addblk->capacity());
	addblk->inc_wr_ptr(1);
	ASSERT_TRUE(addblk->freespace() == addblk->capacity() -1 );
	ASSERT_TRUE(addblk->freespace() == strlen(addthis) );
	addblk->inc_wr_ptr(strlen(addthis));
	ASSERT_TRUE(addblk->length() == (strlen(addthis) + 1));
	addblk->inc_rd_ptr(strlen(addthis) + 1);
	ASSERT_TRUE(addblk->length() == 0);
	const int last_advance = 3;
	addblk->rewind(last_advance);
	ASSERT_TRUE(addblk->length() == strlen(addthis) + 1 - last_advance);


//	string temps;

	head->addToEnd(addblk->duplicate());
//	cout << head->hexDump(temps,0) << endl;
	ASSERT_TRUE(head->total_length() == big_block_size - 1 + strlen(addthis) + 1 - last_advance);
	addblk->release();

	BufBlk<TESTAlloc> *contig = NULL;
	ASSERT_TRUE(head->getContigBlock(contig,0,head->total_length()));
//	cout << contig->hexDump(temps,0) << endl;

	ASSERT_EQ(0,memcmp(contig->rd_ptr(),big_block+1,big_block_size-1)) << "Walk through memory was not the same" << endl;
	ASSERT_EQ(0,memcmp(contig->rd_ptr()+big_block_size-1, addthis + last_advance, strlen(addthis) + 1 - last_advance)) << "Walk through memory was not the same" << endl;

	head->release();
	contig->release();
}




TEST_P(BasicTest, CopyConstructor) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	char *chaincopy = (char *) _alloc->malloc(head->total_length());

	BufBlk<TESTAlloc> *copy_head = new BufBlk<TESTAlloc>(*head);
	ASSERT_EQ(copy_head->total_length(), head->total_length());
	BufBlk<TESTAlloc> *look_copy = copy_head;
	BufBlk<TESTAlloc> *look = head;
	string temps;
//	cout << head->hexDump(temps,0) << endl;
//	cout << copy_head->hexDump(temps,0) << endl;
	while(look) {
		char *temp = (char *) _alloc->malloc(look->length());
		memcpy(temp, look->rd_ptr(), look->length());
//		cout << "temp:      " << dumpHexMem(temps,temp,look->length()) << endl;
//		cout << "look_copy: " << dumpHexMem(temps, look_copy->rd_ptr(),look_copy->length()) << endl;
		ASSERT_EQ(0,memcmp(temp, look_copy->rd_ptr(), look_copy->length()));
		_alloc->free(temp);
		ASSERT_TRUE(look != look_copy); // make sure its a real duplicate
		look = look->nexblk();
		look_copy = look_copy->nexblk();
	}

	copy_head->release();

	head->release();
}

/**
 * Test default constructor and operator=
 */
TEST_P(BasicTest, AssignmentTest) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	char *chaincopy = (char *) _alloc->malloc(head->total_length());

	BufBlk<TESTAlloc> *copy_head = new BufBlk<TESTAlloc>();
	*copy_head = *head;
	ASSERT_EQ(copy_head->total_length(), head->total_length());
	BufBlk<TESTAlloc> *look_copy = copy_head;
	BufBlk<TESTAlloc> *look = head;
	string temps;
//	cout << head->hexDump(temps,0) << endl;
//	cout << copy_head->hexDump(temps,0) << endl;
	while(look) {
		char *temp = (char *) _alloc->malloc(look->length());
		memcpy(temp, look->rd_ptr(), look->length());
//		cout << "temp:      " << dumpHexMem(temps,temp,look->length()) << endl;
//		cout << "look_copy: " << dumpHexMem(temps, look_copy->rd_ptr(),look_copy->length()) << endl;
		ASSERT_EQ(0,memcmp(temp, look_copy->rd_ptr(), look_copy->length()));
		_alloc->free(temp);
		ASSERT_TRUE(look != look_copy); // make sure its a real duplicate
		look = look->nexblk();
		look_copy = look_copy->nexblk();
	}
	copy_head->release();
	ASSERT_EQ(1,head->getRefCount());
	head->release();
}

/**
 * Tests assignment to a blank object
 */
TEST_P(BasicTest, AssignmentToEmpty) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	char *chaincopy = (char *) _alloc->malloc(head->total_length());

	BufBlk<TESTAlloc> *copy_head = new BufBlk<TESTAlloc>();
	*copy_head = *head;
	ASSERT_EQ(copy_head->total_length(), head->total_length());
	BufBlk<TESTAlloc> *look_copy = copy_head;
	BufBlk<TESTAlloc> *look = head;
	string temps;
//	cout << head->hexDump(temps,0) << endl;
//	cout << copy_head->hexDump(temps,0) << endl;
	while(look) {
		char *temp = (char *) _alloc->malloc(look->length());
		memcpy(temp, look->rd_ptr(), look->length());
//		cout << "temp:      " << dumpHexMem(temps,temp,look->length()) << endl;
//		cout << "look_copy: " << dumpHexMem(temps, look_copy->rd_ptr(),look_copy->length()) << endl;
		ASSERT_EQ(0,memcmp(temp, look_copy->rd_ptr(), look_copy->length()));
		_alloc->free(temp);
		ASSERT_TRUE(look != look_copy); // make sure its a real duplicate
		look = look->nexblk();
		look_copy = look_copy->nexblk();
	}

//	copy_head->release();
	BufBlk<TESTAlloc> stackbuf;
	ASSERT_TRUE(!stackbuf.isValid());
	ASSERT_EQ(2,head->getRefCount());
	*copy_head = stackbuf;
	ASSERT_EQ(1,head->getRefCount());
	copy_head->release();
	head->release();
}

TEST_P(BasicTest, StackDestructor) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	char *chaincopy = (char *) _alloc->malloc(head->total_length());

	BufBlk<TESTAlloc> *copy_head = new BufBlk<TESTAlloc>();
	*copy_head = *head;
	ASSERT_EQ(copy_head->total_length(), head->total_length());

//	copy_head->release();
	BufBlk<TESTAlloc> stackbuf(*copy_head);
	ASSERT_TRUE(stackbuf.isValid());
	ASSERT_EQ(3,head->getRefCount());
	copy_head->release();
	ASSERT_EQ(2,head->getRefCount());
	head->release();
	// Now stackbuf should fall off the stack, and release...
}

TEST_P(RepeatTest, SimpleCopyNextChunks) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

	char *chaincopy = (char *) _alloc->malloc(head->total_length());
	int copyleft = head->total_length();
	int out = 0;
	BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( *head );
	char *walkcopy = chaincopy;
	while(it->copyNextChunks( walkcopy,copyleft,out)) {
		copyleft -= out;
		walkcopy += out;
	}

	ASSERT_EQ(0,memcmp(chaincopy,big_block,big_block_size)) << "Walk through memory was not the same" << endl;

	head->release();

	delete it; // yes, you can do this. The iterator has a reference to the BufBlk

	_alloc->free(chaincopy);
}


TEST_P(RepeatTest, SimpleGetContigBlockOffset13) {

	TWMemBlkBufData<TESTAlloc> *l = const_cast<TWMemBlkBufData<TESTAlloc> *>(GetParam());

	BufBlk<TESTAlloc> *head = l->makeOneBigChain();
	if(head->total_length() > 13) {

	ASSERT_EQ(big_block_size,head->total_length()) << "Mismatch on total_length() versus control-group memory." << endl;

//	string temps;
//	cout << head->hexDump(temps,0) << endl;

	BufBlk<TESTAlloc> *contig = NULL;
	ASSERT_TRUE(head->getContigBlock(contig,13,head->total_length()-13));
//	cout << contig->hexDump(temps,0) << endl;

	ASSERT_EQ(0,memcmp(contig->rd_ptr(),big_block+13,big_block_size-13)) << "Walk through memory was not the same" << endl;

	contig->release();

	} else {
		const ::testing::TestInfo* const test_info =
		  ::testing::UnitTest::GetInstance()->current_test_info();
		printf("      >>> Skipping test %s of testcase %d. Control group size was too small for test\n",
		       test_info->name(), test_info->test_case_name());
	}

	head->release();

}


} // end namespace TWlibTests

using namespace TWlibTests;

TWMemBlkBufData<TESTAlloc> *t1, *t2, *t3, *t4;
TWMemBlkBufData<TESTAlloc> *big1, *big2;

INSTANTIATE_TEST_CASE_P(Strings,
		BasicTest,
		::testing::Values( t1 , t2 //, t3, t4
				));

INSTANTIATE_TEST_CASE_P(Strings,
		RepeatTest,
		::testing::Values( big1, big2 // , t2 //, t3, t4
				));


int main(int argc, char **argv) {
	t1 = MemBlkTestMaker<TESTAlloc>::makeStringData(8, "11111", "222222", "333333", "444444", "555555", "666666", "77777", "88888");
	t2 = MemBlkTestMaker<TESTAlloc>::makeStringData(8, "", "1", "22", "333", "4444", "666666", "77777", "88888");
	big1 = MemBlkTestMaker<TESTAlloc>::makeStringRepeatData(10, 8, "", "1", "22", "333", "4444", "666666", "77777", "88888"); // repeat this set of strings 10 times
	big2 = MemBlkTestMaker<TESTAlloc>::makeStringRepeatData(100, 9, "", "1", "22", "333", "4444", "666666", "77777", "88888", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"); // repeat this set of strings 100 times

//	t2 = MemBlkTestMaker::makeStringData(4, "hello", "hello2", "hello3", "hello4");
//	t3 = MemBlkTestMaker::makeStringData(4, "hello", "hello2", "hello3", "hello4");
//	t4 = MemBlkTestMaker::makeStringData(4, "hello", "hello2", "hello3", "hello4");
	::testing::InitGoogleTest(&argc, argv);
	return RUN_ALL_TESTS();
}

