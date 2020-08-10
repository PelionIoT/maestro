// WigWag LLC
// (c) 2011
// testutils.cpp
// Author: ed
// Apr 23, 2011/*
// Apr 23, 2011 * testutils.cpp
// Apr 23, 2011 *
// Apr 23, 2011 *  Created on: Apr 23, 2011
// Apr 23, 2011 *      Author: ed
// Apr 23, 2011 */

#include <gtest/gtest.h>

#include <TW/tw_alloc.h>
#include <TW/tw_bufblk.h>

#include <string>
#include <iostream>

#include "testutils.h"

using namespace std;

string &dumpHexMem(string &out, char *area, int size) {
	char *walk = area;
	ostringstream outs;
	out.clear();
	outs.str(out);
	outs.setf(ios_base::hex,ios_base::basefield);
	outs << "dumpHexMem[";
	bool first = true;

	for(int x=0;x<size;x++) {
		if (!first) outs << ",";
		outs << ((int)*walk);
		walk++;
		first = false;
	}
	outs << "]";
	out = outs.str();
	return out;
}

///**
// * disconnects a chain into individual blocks. more like what might appear from a network connection.
// * @param in
// * @param parts
// * @param numblks
// */
//void disconnectBlocks(ACE_Message_Block *in, ACE_Message_Block **parts, int numblks ) {
//	for (int x=0;x<numblks;x++) {
//		parts[x] = NULL;
//	}
//
//	ACE_Message_Block *look = in;
//	int x = 0;
//	while(look) {
//		parts[x] = look;
//		look = look->cont();
//		parts[x]->cont(NULL); // disconnect them
//		x++;
//		if(x > numblks)
//			ADD_FAILURE() << "overrun in disconnectBlocks()" << endl;
//	}
//
//}
//
///**
// * slices up 'in' into a bunch of smaller block of blksize
// * @param in
// * @param parts
// * @param blksize
// */
//void equalBlocks(ACE_Message_Block *in, ACE_Message_Block **parts, int blksize ) {
//	int blknum = 0;
//	int blkspc = 0;
//	ACE_Message_Block *look = in;
//
//	while(look) {
//		if(parts[blknum] == NULL) {
////			cout << "HELLLLLLLLLLLLLLLLLLO" << endl;
////			cout << " ** ";
//			parts[blknum] = new ACE_Message_Block((size_t) blksize, ACE_Message_Block::MB_DATA, 0,0,ACE_Allocator::instance());
//		}
////		cout << "LOOK: " << look << " length: " << look->length();
////		cout << " num: " << blknum << " block space: " << parts[blknum]->space() << endl;
//		blkspc = parts[blknum]->space();
//		if(look->length() > blkspc) {
//			parts[blknum]->copy(look->rd_ptr(), blkspc);
//			look->rd_ptr(blkspc);
//			blknum++;
//		} else {
//			parts[blknum]->copy(look->rd_ptr(), look->length());
//			look = look->cont();
//		}
//	}
//}
//
///**
// * slices up 'in' into a bunch of smaller block of blksize
// * @param in
// * @param parts
// * @param blksize
// */
//void equalBlocksConnect(ACE_Message_Block *in, ACE_Message_Block *&newhead, int blksize ) {
//	int blknum = 0;
//	int blkspc = 0;
//	ACE_Message_Block *look = in;
//	ACE_Message_Block *parts = NULL;
//	ACE_Message_Block *lastpart = NULL;
//	if(look) {
//		newhead = new ACE_Message_Block((size_t) blksize, ACE_Message_Block::MB_DATA, 0,0,ACE_Allocator::instance());
//		cout << " ** ";
//		parts = newhead;
//	}
//	while(look) {
//		if(parts == NULL) {
////			cout << "HELLLLLLLLLLLLLLLLLLO" << endl;
//			cout << " ** ";
//			parts = new ACE_Message_Block((size_t) blksize, ACE_Message_Block::MB_DATA, 0,0,ACE_Allocator::instance());
//		}
////		cout << "LOOK: " << look << " length: " << look->length();
////		cout << " num: " << blknum << " block space: " << parts->space() << endl;
//		blkspc = parts->space();
//		if(look->length() > blkspc) { // if inbound block is not big enough to handle all..
//			parts->copy(look->rd_ptr(), blkspc);
//			look->rd_ptr(blkspc);
//			if(lastpart) {
//				lastpart->cont(parts);
//			}
//			lastpart = parts;
//			parts = NULL;
////			blknum++;
//		} else { // inbound block can handle all of this block..
//			parts->copy(look->rd_ptr(), look->length());
//			look = look->cont();
//			if((parts->space() < 1) || !look) {
//				if(lastpart) {
//					lastpart->cont(parts);
//				}
//				lastpart = parts;
//				parts = NULL;
//			}
//		}
//	}
//}
//
//
///**
// * this takes two ACE_Message_Blocks - find their last chain in the block of the first one, and combines it and
// * the first block of the second into one large block. The entire group is chained together.
// * @param first This chain should not be release()d or used after the call.
// * @param seccond This chain should not be release()d or used after the call.
// * @return The combined chain. This if not NULL is now the only valid chain.
// */
//ACE_Message_Block *combineBlockInMiddle( ACE_Message_Block *first, ACE_Message_Block *second ) {
//	if(!first || !second) {
//		cout << "combineBlockInMiddle got NULLs" << endl;
//		return NULL;
//	}
//	ACE_Message_Block *look=first;
//	ACE_Message_Block *ret=first;
//	ACE_Message_Block *frontend = first;
//	while(look->cont()) {
//		frontend = look; // one back from last chain
//		look = look->cont();
//	}
//
//	int s = look->length();
//	ACE_Message_Block *middle = new ACE_Message_Block(second->length()+s, ACE_Message_Block::MB_DATA, 0,0,ACE_Allocator::instance());
//	middle->copy(look->rd_ptr(),s);
//	middle->copy(second->rd_ptr(),second->length());
//	if(look == first)
//		ret = middle; // if there was only one block in 'first' then middle is now the front
//	else
//		frontend->cont(middle); // otherwise, make the one before middle now point to middle
//
//	if(second->cont()) { // if the second chain has a cont() chain...
//		middle->cont(second->cont()); // attach it to the new middle block
//		second->cont(NULL); // disconnect the original first block of second from its remaining chains
//	}
//	look->cont(NULL);
//	look->release();   // the two blocks which made middle are history... bye.
//	second->release();
//	return ret;
//}
//
//
//void dumpACE_Message_Block(ACE_Message_Block *head, string &out) {
//	ostringstream outs;
//	out.clear();
//	outs.str(out);
//	outs.setf(ios_base::hex,ios_base::basefield);
//	ACE_Message_Block *walk = head;
//	char *walkp = NULL;
//	while(walk) {
//		outs << "[";
//		walkp=walk->rd_ptr();
//		bool first = true;
//		while(walkp < walk->wr_ptr()) {
//			if (!first) outs << ",";
//			outs << ((int)*walkp);
//			first = false;
//			walkp++;
//		}
//		outs << "]->";
//		walk = walk->cont();
//	}
//	outs << "END";
//	out = outs.str();
//}
//
//
//
//void dumpACE_Message_Block2(ACE_Message_Block *head, string &out) {
//	ostringstream outs;
//	out.clear();
//	outs.str(out);
//	outs.setf(ios_base::hex,ios_base::basefield);
//	ACE_Message_Block *walk = head;
//	char *walkp = NULL;
//	while(walk) {
//		walkp=walk->rd_ptr();
//		outs << ((void *)walkp) << "[";
//		bool first = true;
//		while(walkp < walk->wr_ptr()) {
//			if (!first) outs << ",";
//			outs << ((int)*walkp);
//			first = false;
//			walkp++;
//		}
//		outs << "]->";
//		walk = walk->cont();
//	}
//	outs << "END";
//	out = outs.str();
//}
//
//int countLinksInChain(ACE_Message_Block *head) {
//	int ret = 0;
//	ACE_Message_Block *l = head;
//	while(l) {
//		ret++;
//		l = l->cont();
//	}
//	return ret;
//}
//
//void splitInTwo(ACE_Message_Block *chain, ACE_Message_Block *&outleft,ACE_Message_Block *&outright ) {
//	ACE_Message_Block *walk = chain;
//	int cnt = countLinksInChain(chain);
//	outleft = chain;
//	if(cnt > 3) {
//		for(int x = 0;x<cnt/2;x++) {
//			walk = walk->cont();
//		}
//		outright = walk->cont();
//		walk->cont(NULL); // break apart
//	}
//}
