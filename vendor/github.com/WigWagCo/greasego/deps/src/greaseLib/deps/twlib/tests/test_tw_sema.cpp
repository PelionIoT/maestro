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

#include <TW/tw_utils.h>
#include <TW/tw_sema.h>

void *print_message_function( void *ptr );

#define RELEASE_CNT 15
#define ACQUIRE_CNT 7
#define QUEUE_CNT 0

#define TIMEOUT_SEC 0
#define TIMEOUT_USEC 1000000

using namespace TWlib;

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


void *print_and_look( void *ptr ) {
	TW_Sema *sem = reinterpret_cast<TW_Sema *>(ptr);
	int x = ACQUIRE_CNT;
	while(x > 0) {
		sem->acquire();
		x--;
		printf("Look acquired semaphore %d times\n",ACQUIRE_CNT-x);
	}
}

void *print_and_look_timeout( void *ptr ) {
	TW_Sema *sem = reinterpret_cast<TW_Sema *>(ptr);
	int x = ACQUIRE_CNT;
	struct timeval ts;
	struct timespec ts2;

	int tc = 0;
	while(x > 0) {
		gettimeofday(&ts, NULL);

		add_usec_to_timeval(TIMEOUT_USEC, &ts);
		timeval_to_timespec(&ts,&ts2);
//		ts2.tv_sec = ts.tv_sec;
//		ts2.tv_nsec = ts.tv_usec * 1000;
//		ts2.tv_sec += ts2.tv_nsec / 1000000000L;
//		ts2.tv_nsec %= 1000000000L;
		int res = 0;
		if((res = sem->acquire(&ts2)) != 0) {
			 if(res == ETIMEDOUT) {
				 tc++;
				 printf("timed out (%d)...\n", tc);
			 } else
				 printf("Other error: %s\n", strerror(res));
		} else {
			tc = 0;
			x--;
			printf("Look acquired semaphore %d times (no timeout)...\n",ACQUIRE_CNT-x);
		}
	}
}

void *print_and_look_timeout2( void *ptr ) {
	TW_Sema *sem = reinterpret_cast<TW_Sema *>(ptr);
	int x = ACQUIRE_CNT;
	struct timeval ts;
	struct timespec ts2;

	int tc = 0;
	while(x > 0) {
		int res = 0;
		if((res = sem->acquire(TIMEOUT_USEC)) != 0) {
			 if(res == ETIMEDOUT) {
				 tc++;
				 printf("timed out (%d)...\n", tc);
			 } else
				 printf("Other error: %s\n", strerror(res));
		} else {
			tc = 0;
			x--;
			printf("Look acquired semaphore %d times (no timeout) (#2)...\n",ACQUIRE_CNT-x);
		}
	}
}
void *print_and_count( void *ptr ) {
	TW_Sema *sem = reinterpret_cast<TW_Sema *>(ptr);
	int x = RELEASE_CNT;
	while(x > 0) {
		sleep(2);
		printf("Slept two seconds & releasing: X is %d\n",x);
		sem->release();
		x--;
	}

}

/*
void *print_release_countdown( void *ptr ) {
	TW_Sema *sem = reinterpret_cast<TW_Sema *>(ptr);
	int x = 5;
	while(x > 0) {
		sleep(2);
		printf("Slept two seconds & releasing: X is %d\n",x);
		sem->release();
		x--;
	}
}

void *print_acquire_delete( void *ptr ) {
	TW_Sema *sem = reinterpret_cast<TW_Sema *>(ptr);
	int x = 15;
	while(x > 0) {
		if(x==1) {
			sem->flagDeleteAtZero();
			printf("Flagging delete at zero\n");
		}
		sleep(2);
		printf("Slept two seconds & acquiring: X is %d\n",x);
		sem->acquire();
		x--;
	}
}
*/

int main()
{
     pthread_t thread1, thread2, thread3, thread4, thread5;
     char *message1 = "Thread 1";
     char *message2 = "Thread 2";
     int  iret1, iret2, iret3;


     TW_Sema test_sema(QUEUE_CNT);


    /* Create independent threads each of which will execute function */

     iret1 = pthread_create( &thread1, NULL, print_and_look_timeout2, reinterpret_cast<void *>(&test_sema));
     iret2 = pthread_create( &thread2, NULL, print_and_count, reinterpret_cast<void *>(&test_sema));
     iret3 = pthread_create( &thread3, NULL, print_and_look, reinterpret_cast<void *>(&test_sema));

     /* Wait till threads are complete before main continues. Unless we  */
     /* wait we run the risk of executing an exit which will terminate   */
     /* the process and all threads before the threads have completed.   */

     pthread_join( thread1, NULL);
     pthread_join( thread2, NULL);
     pthread_join( thread3, NULL);

     printf("Thread 1 returns: %d\n",iret1);
     printf("Thread 2 returns: %d\n",iret2);

//     TW_Sema countdown(10); // uses the semaphore as a countdown system, one thread flags it to self-delete at 0
//
// /    iret1 = pthread_create( &thread4, NULL, print_release_countdown, reinterpret_cast<void *>(&countdown));
//     iret2 = pthread_create( &thread5, NULL, print_acquire_delete, reinterpret_cast<void *>(&countdown));

//     pthread_join( thread4, NULL);
//     pthread_join( thread5, NULL);

     exit(0);
}

void *print_message_function( void *ptr )
{
     char *message;
     message = (char *) ptr;
     printf("%s \n", message);
}


