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
#include <errno.h>
#include <string.h>
#include <assert.h>

#include <TW/tw_utils.h>
#include <TW/tw_alloc.h>
#include <TW/tw_sema.h>
#include <TW/tw_circular.h>

#include <sys/syscall.h>

static long gettid__test() {
	return syscall(SYS_gettid);
}

#define TPRINTF(s,...) fprintf(stderr, "(tid:%d)  " s, gettid__twcircular(), ##__VA_ARGS__ )

void *print_message_function( void *ptr );

#define QUEUE_SIZE 20
#define RUN_SIZE 200
#define UNIQUE_VALUES (RUN_SIZE/PRODUCER_THREADS)

#define CONSUMER_THREADS 4
#define PRODUCER_THREADS 4

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


int verify[UNIQUE_VALUES];

class data {
public:
	int x;
	data() : x(-1000) {}
	data(data &o) : x(o.x) {}

};

void *producer( void *ptr ) {
	threadinfo *inf = reinterpret_cast<threadinfo *>(ptr);
	tw_safeCircular<data, TESTAlloc > *Q = reinterpret_cast<tw_safeCircular<data, TESTAlloc > *>(inf->p);
	int x = UNIQUE_VALUES;
	int val = START_VAL;
	data D;
	TPRINTF("+++ Producer %d started\n\n", inf->threadnum);
	while(x > 0) {
		D.x = val;
		TPRINTF(">>> Producer %d: adding %d\n", inf->threadnum, val);
		Q->add(D);
		TPRINTF(">>> Producer %d: added %d\n\n", inf->threadnum, val);
		val++;
		x--;
	}
	TPRINTF("--- Producer %d done!!!\n\n", inf->threadnum);
}

void *consumer( void *ptr ) {
	threadinfo *inf = reinterpret_cast<threadinfo *>(ptr);
	tw_safeCircular<data, TESTAlloc > *Q = reinterpret_cast<tw_safeCircular<data, TESTAlloc > *>(inf->p);
	int x = RUN_SIZE / CONSUMER_THREADS;
	int cnt = 0;
	int tc = 0;
	data D;
	TPRINTF("+++ Consumer %d started\n\n", inf->threadnum);

	while(x > 0) {
		// this also works...
//		totalMutex->acquire();
//		printf("HERE\n");
//		if(TOTAL < 1) {
//			totalMutex->release();
//			break;
//		}
		do {
			if(Q->removeOrBlock(D)) {
			cnt++;
			TPRINTF("<<< Consumer %d: removed %d\n\n",inf->threadnum, D.x);
			totalMutex->acquire();
			assert(D.x >= 0);
			verify[D.x] = verify[D.x] + 1;
			totalMutex->release();
			TOTAL--;
			x--;
			break;
			} else {
				TPRINTF("<<< Consumer %d: failed remove\n\n", inf->threadnum);
			}
		} while(1);
//		totalMutex->release();
	}
	TPRINTF("--- Consumer %d done!!!\n\n", inf->threadnum);
	OUTPUT[inf->threadnum] = cnt;
}



int main()
{
     //pthread_t thread1, thread2, thread3;
	pthread_t consumert[CONSUMER_THREADS];
	pthread_t producert[PRODUCER_THREADS];


	memset(verify,0,sizeof(verify));

	for (int x=0;x<CONSUMER_THREADS;x++)
		OUTPUT[x] = 0;

     int  iret1, iret2, iret3;
     totalMutex = new TW_Mutex();


     tw_safeCircular<data, TESTAlloc > theQ( QUEUE_SIZE, true );

    /* Create independent threads each of which will execute function */
	 threadinfo *inf;

     for (int x=0;x<CONSUMER_THREADS;x++) {
    	 inf = new threadinfo;
    	 inf->p = reinterpret_cast<void *>(&theQ);
    	 inf->threadnum = x;
    	 pthread_create( &consumert[x], NULL, consumer, reinterpret_cast<void *>(inf));
     }
     printf("---- consumers started --- sleep 1 - then producers ----\n\n");
     sleep(1);

     for (int x=0;x<PRODUCER_THREADS;x++) {
    	 inf = new threadinfo;
    	 inf->p = reinterpret_cast<void *>(&theQ);
    	 inf->threadnum = x;
    	 pthread_create( &producert[x], NULL, producer, reinterpret_cast<void *>(inf));
     }


//     iret1 = pthread_create( &thread1, NULL, creator, reinterpret_cast<void *>(&theQ));
//     iret2 = pthread_create( &thread2, NULL, consumer, reinterpret_cast<void *>(&theQ));

  //   iret3 = pthread_create( &thread3, NULL, print_and_look, reinterpret_cast<void *>(&test_sema));

     /* Wait till threads are complete before main continues. Unless we  */
     /* wait we run the risk of executing an exit which will terminate   */
     /* the process and all threads before the threads have completed.   */


     for (int x=0;x<CONSUMER_THREADS;x++) {
    	 pthread_join( consumert[x], NULL);
     }
     for (int x=0;x<PRODUCER_THREADS;x++) {
    	 pthread_join( producert[x], NULL);
     }

//     pthread_join( thread2, NULL);
//     pthread_join( thread3, NULL);

 //    printf("Thread 1 returns: %d\n",iret1);
  //   printf("Thread 2 returns: %d\n",iret2);

 	for (int x=0;x<CONSUMER_THREADS;x++) {
 		printf("Thread num %d -> output %d\n", x, OUTPUT[x]);
 	}

 	for(int n=0;n<UNIQUE_VALUES;n++) {
 		printf("verify[%d] == %d\n", n,verify[n]);
 		assert(verify[n] == PRODUCER_THREADS);
 	}


     exit(0);
}

