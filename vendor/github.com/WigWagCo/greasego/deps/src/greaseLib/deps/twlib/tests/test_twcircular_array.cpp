// WigWag LLC
// (c) 2011
// test_tw_sema.cpp
// Author: ed
// Mar 22, 2011/*
// Mar 22, 2011 * test_tw_sema.cpp
// Mar 22, 2011 *
// Mar 22, 2011 *  Created on: Mar 22, 2011
// Mar 22, 2011 *      Author: ed
// Mar 22, 2011 */

#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <time.h>
#include <sys/time.h>
#include <assert.h>
#include <errno.h>
#include <string.h>

#include <TW/tw_utils.h>
#include <TW/tw_alloc.h>
#include <TW/tw_sema.h>
#include <TW/tw_circular.h>

void *print_message_function( void *ptr );

#define QUEUE_SIZE 20
#define RUN_SIZE 200

#define CONSUMER_THREADS 4
#define PRODUCER_THREADS 2


#define START_VAL 0

int TOTAL = RUN_SIZE;
TW_Mutex *totalMutex;

/*
struct timeval* usec_to_timeval( int64_t usec, struct timeval* tv )
{
	tv->tv_sec = usec / 1000000 ;
	tv->tv_usec = usec % 1000000 ;
	return tv ;
}

struct timeval* add_usec_to_timeval( int64_t usec, struct timeval* tv ) {
    tv->tv_sec += usec / 1000000 ;
    tv->tv_usec += usec % 1000000 ;

//    tv.tv_sec = tv1.tv_sec + tv2.tv_sec ;  // add seconds
 //   tv.usec = tv1.tv_usec + tv2.tv_usec ; // add microseconds
//    tv->tv_sec += tv.tv_usec / 1000000 ;  // add microsecond overflow to seconds
//   tv->tv_usec %= 1000000 ; // subtract the overflow from microseconds
    return tv;
}

*/

using namespace TWlib;

typedef Allocator<Alloc_Std> TESTAlloc;

int OUTPUT[PRODUCER_THREADS];

class threadinfo {
public:
	int threadnum;
	void *p; // some data
};

class data {
public:
	int x;
	data() : x(0) {}
	data(data &o) : x(o.x) {}

};


int main()
{

     tw_safeCircular<data, TESTAlloc > theQ( QUEUE_SIZE, true );

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 d.x = n;
         theQ.add(d);
    	 printf("add [%d] .remaining() = %d\n", n, theQ.remaining());
     }

     int n = 0;
     data d;
     while(theQ.remove(d)) {
    	 printf("remove [%d]  = %d\n", n, d.x);
    	 n++;
     }

     printf("--- again...\n");


     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 d.x = n;
         theQ.add(d);
    	 printf("add [%d] .remaining() = %d\n", n, theQ.remaining());
     }

     n = 0;
     while(theQ.remove(d)) {
    	 printf("remove [%d]  = %d\n", n, d.x);
    	 n++;
     }

     printf("--- reverse ---\n");

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 d.x = n;
         theQ.add(d);
    	 printf("add [%d] .remaining() = %d\n", n, theQ.remaining());
     }

     theQ.reverse();                                                  // reverse. now backwards

     n = 0;
     while(theQ.remove(d)) {
    	 printf("remove [%d]  = %d\n", n, d.x);
    	 n++;
     }

     theQ.reverse();                                                   // reverse. now forward
     printf("--- reverse (again) ---\n");

//     printf("-- clear() --- \n");
//     for(int n=0;n<QUEUE_SIZE;n++) {
//    	 data d;
//    	 d.x = n;
//         theQ.add(d);
//     }
//     theQ.clear();
     printf("--- sanity check ---\n");
     assert(theQ.remaining() == 0);
     d.x=101;
     theQ.add(d);
     assert(theQ.remaining() == 1);
     d.x=0;
     assert(theQ.remove(d));
     assert(d.x == 101);
     assert(theQ.remaining() == 0);

     printf("--- get()/set() ---\n");


     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 d.x = n;
    	 theQ.add(d);
     }

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 assert(theQ.get(n,d));
    	 assert(d.x == n);
    	 printf("get(%d) = %d\n", n, d.x);
    	 d.x = d.x * 10;
    	 assert(theQ.set(n,d));
     }

     theQ.reverse();                                                   // reverse. now backwards
     printf("get() after reverse() [should be the same but * 10]\n");

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 assert(theQ.get(n,d));
    	 printf("get(%d) = %d\n", n, d.x);
     }

     theQ.reverse();                                                   // reverse. now forwards
     printf("sanity check....\n");

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 assert(theQ.get(n,d));
    	 printf("get(%d) = %d\n", n, d.x);
     }
     theQ.reverse();                                                   // reverse. now backwards

     data removedD;
     assert(theQ.remove(removedD));  // the last item should be 190 - so now its the first...
     printf("removed x=%d\n", removedD.x);
     assert(removedD.x == 190);

     theQ.reverse();                                                  // reverse. now forwards
     printf("reverse()...\n");


     tw_safeCircular<data, TESTAlloc >::iter iter = theQ.getIter();

     n = 0;
     while(!iter.atEnd()) {
    	 data d;
    	 iter.data(d);
    	 printf("iter %d = %d\n", n, d.x);
    	 n++;
    	 if(!iter.next()) {
    		 printf("iter at end.\n\n");
    	 }
     }
     iter.release();

     theQ.remove(removedD);
     printf("removed: x=%d\n\n", removedD.x);

     tw_safeCircular<data, TESTAlloc >::iter iter2 = theQ.getIter();

     n = 0;
     while(!iter2.atEnd()) {
    	 data d;
    	 iter2.data(d);
    	 printf("iter %d = %d\n", n, d.x);
    	 n++;
    	 if(!iter2.next()) {
    		 printf("iter at end.\n");
    	 }
     }
     iter2.release();

     theQ.clear();
     printf("clear()\n");
     assert(theQ.remaining() == 0);

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 d.x = n;
    	 theQ.add(d);
     }
     assert(theQ.remaining() == QUEUE_SIZE);

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 assert(theQ.get(n,d));
    	 assert(d.x == n);
//    	 printf("get(%d) = %d\n", n, d.x);
    	 d.x = d.x * 10;
    	 assert(theQ.set(n,d));
     }

     exit(0);
}

