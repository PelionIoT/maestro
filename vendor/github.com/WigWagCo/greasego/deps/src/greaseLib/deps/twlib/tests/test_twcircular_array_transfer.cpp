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

#define QUEUE_SIZE 5
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
	data() : x(-1000) {}
	data(data &o) : x(o.x) {}
	data& operator=(const data& o) {
		x = o.x;
		return *this;
	}
	data(data &&o) : x(o.x) { o.x = -1000; }
};


int main()
{

	 int verify[QUEUE_SIZE];
	 memset(verify,0,sizeof(QUEUE_SIZE));

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
    	 assert(n==d.x);
    	 n++;
     }

     assert(n == QUEUE_SIZE);

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
    	 assert(n==d.x);
    	 n++;
     }




     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 d.x = n;
         theQ.add(d);
    	 printf("add [%d] .remaining() = %d\n", n, theQ.remaining());
     }


     tw_safeCircular<data, TESTAlloc > theQ2( QUEUE_SIZE, true );

     assert(theQ.remaining() == QUEUE_SIZE);
     printf("transfer to theQ2\n");
     theQ2.transferFrom(theQ);

     assert(theQ.remaining() == 0);
     assert(theQ2.remaining() == QUEUE_SIZE);


     theQ.transferFrom(theQ2);
     assert(theQ2.remaining() == 0);
     assert(theQ.remaining() == QUEUE_SIZE);

     theQ2.transferFrom(theQ);

     assert(theQ.remaining() == 0);
     assert(theQ2.remaining() == QUEUE_SIZE);



     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 assert(theQ2.get(n,d));
    	 assert(d.x == n);
     }
	 assert(!theQ.get(0,d)); // and assure the original Q got reinitialized

     tw_safeCircular<data, TESTAlloc > theQ3( QUEUE_SIZE, true );

     theQ3.cloneFrom(theQ2);

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 assert(theQ2.get(n,d));
    	 assert(d.x == n);
    	 assert(theQ3.get(n,d));
    	 assert(d.x == n);
     }


     tw_safeCircular<data, TESTAlloc > theQ4( QUEUE_SIZE, true );

     n = 0;
     while(theQ2.remove(d)) {
    	 printf("Q2: remove [%d]  = %d\n", n, d.x);
    	 theQ4.add(d);
    	 n++;
     }
     printf("---\n");
     assert(n == QUEUE_SIZE);
     assert(theQ4.remaining() == QUEUE_SIZE);

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 assert(theQ4.get(n,d));
    	 assert(d.x == n);
     }

     theQ2.transferFrom(theQ4);

     assert(theQ4.remaining() == 0);
     assert(theQ2.remaining() == QUEUE_SIZE);

     for(int n=0;n<QUEUE_SIZE;n++) {
    	 data d;
    	 assert(theQ2.get(n,d));
    	 assert(d.x == n);
     }

     int z = 0;
     while(theQ2.remove(d)) {
    	 printf("Q2: remove [%d]  = %d\n", z, d.x);
//    	 assert(d.x == z);
    	 z++;
     }
     printf("---\n");

     assert(z == QUEUE_SIZE);


     // the logger.h test...


	 tw_safeCircular<data, TESTAlloc > nextQ( QUEUE_SIZE, true );

	 for(int p=0;p<QUEUE_SIZE;p++) {
		 data d;
		 d.x = p;
		 nextQ.add(d);
	 }
	 assert(nextQ.remaining() == QUEUE_SIZE);

	 tw_safeCircular<data, TESTAlloc > nextQ2( QUEUE_SIZE, true );

	 for(int z=0;z<100;z++) {
		 printf("<iterate %d > ",z);

		 data outd;
		 while(nextQ.remove(outd)) {
			 nextQ2.add(outd);
		 }

		 assert(nextQ.remaining() == 0);
//		 printf(" Q2.remain = %d ", nextQ2.remaining());
		 assert(nextQ2.remaining() == QUEUE_SIZE);


		 for(int n=0;n<QUEUE_SIZE;n++) {
			 data d;
			 assert(nextQ2.get(n,d));
			 assert(d.x == n);
		 }

		 nextQ.transferFrom(nextQ2);

		 assert(nextQ2.remaining() == 0);
		 assert(nextQ.remaining() == QUEUE_SIZE);


//		 printf("reverse()\n");


	 }

	 printf("\nhere...\n");

	 for (int z=0;z<100;z++) {

		 tw_safeCircular<data, TESTAlloc > nextQ3( QUEUE_SIZE, true );

		 nextQ.reverse();
		 printf("z:%d nextQ.remain = %d\n", z,nextQ.remaining());
		 assert(nextQ.remaining() == QUEUE_SIZE);

		 if(z== 1 || z%2 > 0) {
			 printf("not reversed\n");
			 for(int p=0;p<QUEUE_SIZE;p++) {
				 data d;
				 nextQ.remove(d);
				 printf("rev: d.x = %d\n",d.x);
				 assert(d.x == p);
				 nextQ3.add(d);
			 }
		 } else {
			 printf("reversed\n");
			 for(int p=QUEUE_SIZE-1;p>=0;p--) {
				 data d;
				 nextQ.remove(d);
				 printf("rev: d.x = %d\n",d.x);
				 assert(d.x == p);
				 nextQ3.add(d);
			 }

		 }

		 nextQ.transferFrom(nextQ3);

	 }

//
//
//
//     theQ.reverse();                                                  // reverse. now backwards
//
//     n = 0;
//     while(theQ.remove(d)) {
//    	 printf("remove [%d]  = %d\n", n, d.x);
//    	 n++;
//     }
//
//     theQ.reverse();                                                   // reverse. now forward
//     printf("--- reverse (again) ---\n");
//
////     printf("-- clear() --- \n");
////     for(int n=0;n<QUEUE_SIZE;n++) {
////    	 data d;
////    	 d.x = n;
////         theQ.add(d);
////     }
////     theQ.clear();
//     printf("--- sanity check ---\n");
//     assert(theQ.remaining() == 0);
//     d.x=101;
//     theQ.add(d);
//     assert(theQ.remaining() == 1);
//     d.x=0;
//     assert(theQ.remove(d));
//     assert(d.x == 101);
//     assert(theQ.remaining() == 0);
//
//     printf("--- get()/set() ---\n");
//
//
//     for(int n=0;n<QUEUE_SIZE;n++) {
//    	 data d;
//    	 d.x = n;
//    	 theQ.add(d);
//     }
//
//     for(int n=0;n<QUEUE_SIZE;n++) {
//    	 data d;
//    	 assert(theQ.get(n,d));
//    	 assert(d.x == n);
//    	 printf("get(%d) = %d\n", n, d.x);
//    	 d.x = d.x * 10;
//    	 assert(theQ.set(n,d));
//     }
//
//     theQ.reverse();                                                   // reverse. now backwards
//     printf("get() after reverse() [should be the same but * 10]\n");
//
//     for(int n=0;n<QUEUE_SIZE;n++) {
//    	 data d;
//    	 assert(theQ.get(n,d));
//    	 printf("get(%d) = %d\n", n, d.x);
//     }
//
//     theQ.reverse();                                                   // reverse. now forwards
//     printf("sanity check....\n");
//
//     for(int n=0;n<QUEUE_SIZE;n++) {
//    	 data d;
//    	 assert(theQ.get(n,d));
//    	 printf("get(%d) = %d\n", n, d.x);
//     }
//     theQ.reverse();                                                   // reverse. now backwards
//
//     data removedD;
//     assert(theQ.remove(removedD));  // the last item should be 190 - so now its the first...
//     printf("removed x=%d\n", removedD.x);
//     assert(removedD.x == 190);
//
//     theQ.reverse();                                                  // reverse. now forwards
//     printf("reverse()...\n");
//
//
//     tw_safeCircular<data, TESTAlloc >::iter iter = theQ.getIter();
//
//     n = 0;
//     while(!iter.atEnd()) {
//    	 data d;
//    	 iter.data(d);
//    	 printf("iter %d = %d\n", n, d.x);
//    	 n++;
//    	 if(!iter.next()) {
//    		 printf("iter at end.\n\n");
//    	 }
//     }
//     iter.release();
//
//     theQ.remove(removedD);
//     printf("removed: x=%d\n\n", removedD.x);
//
//     tw_safeCircular<data, TESTAlloc >::iter iter2 = theQ.getIter();
//
//     n = 0;
//     while(!iter2.atEnd()) {
//    	 data d;
//    	 iter2.data(d);
//    	 printf("iter %d = %d\n", n, d.x);
//    	 n++;
//    	 if(!iter2.next()) {
//    		 printf("iter at end.\n");
//    	 }
//     }
//     iter2.release();
//
//     theQ.clear();
//     printf("clear()\n");
//     assert(theQ.remaining() == 0);
//
//     for(int n=0;n<QUEUE_SIZE;n++) {
//    	 data d;
//    	 d.x = n;
//    	 theQ.add(d);
//     }
//     assert(theQ.remaining() == QUEUE_SIZE);
//
//     for(int n=0;n<QUEUE_SIZE;n++) {
//    	 data d;
//    	 assert(theQ.get(n,d));
//    	 assert(d.x == n);
////    	 printf("get(%d) = %d\n", n, d.x);
//    	 d.x = d.x * 10;
//    	 assert(theQ.set(n,d));
//     }

     exit(0);
}

