// Framez LLC
// (c) 2011
// testutils.h
// Author: ed
// Apr 23, 2011

/*
 * testutils.h
 *
 *  Created on: Apr 23, 2011
 *      Author: ed
 */

#ifndef TESTUTILS_H_
#define TESTUTILS_H_

#include <string>

using namespace std;

string &dumpHexMem(string &out, char *area, int size);
/*
void disconnectBlocks(ACE_Message_Block *in, ACE_Message_Block **parts, int numblks );
void equalBlocks(ACE_Message_Block *in, ACE_Message_Block **parts, int blksize );
void equalBlocksConnect(ACE_Message_Block *in, ACE_Message_Block *&newhead, int blksize );
ACE_Message_Block *combineBlockInMiddle( ACE_Message_Block *first, ACE_Message_Block *second );
void dumpACE_Message_Block(ACE_Message_Block *head, string &out);
void dumpACE_Message_Block2(ACE_Message_Block *head, string &out);
int countLinksInChain(ACE_Message_Block *head);
void splitInTwo(ACE_Message_Block *chain, ACE_Message_Block *&outleft,ACE_Message_Block *&outright );
*/

#endif /* TESTUTILS_H_ */
