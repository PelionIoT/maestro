// WigWag LLC
// (c) 2010
// test_fifo.cpp
// Author: ed

#include <stdio.h>
#include <pthread.h>
#include <poll.h>

#include <TW/tw_fifo.h>


#define __SLEEP_VAL (50*1000)   // sleep for 50 ms
#define __STOP_VAL 20

typedef Allocator<Alloc_Std> TESTAlloc;

// some function to sleep for a bit
int usleep( unsigned int usec)
{
	int subtotal = 0; /* microseconds */
	int msec; /* milliseconds */

	/* 'foo' is only here because some versions of 5.3 have
	 * a bug where the first argument to poll() is checked
	 * for a valid memory address even if the second argument is 0.
	 */
	struct pollfd foo;

	subtotal += usec;
	/* if less then 1 msec request, do nothing but remember it */
	if (subtotal < 1000)
		return (0);
	msec = subtotal / 1000;
	subtotal = subtotal % 1000;
	return poll(&foo, (unsigned long) 0, msec);
}


// tw_safeFIFO must operate on an object with
// self assignment: operator= (T &a, T &b)
// copy constructor
// default constructor
class testdat {
public:
	testdat() { x = 0; }
	testdat( int a ) { x = a; }; // default constructor (or param)
	testdat( testdat &o ) { x = o.x; };  // copy constructor
	testdat& operator=(const testdat &o) { this->x = o.x; return *this; }
	int x;
};

// thread functions:

void *producer_func( void *ptr ) { // pthread_requires this definition for the thread func
	// we will use void *ptr to pass a tw_safeFIFO
	tw_safeFIFO<testdat, TESTAlloc> *queue = (tw_safeFIFO<testdat, TESTAlloc> *) ptr;
	int n = 0;
	testdat fillme;

	// fill it up with 10 objs to make a better test...
	for(n=0;n<=10;n++) {
//		usleep(__SLEEP_VAL);
		fillme.x = 1;
		queue->add(fillme);
		printf("producer: added %d\n", fillme.x);
	}

	for(n=0;n<=__STOP_VAL;n++) {
		usleep(__SLEEP_VAL);
		fillme.x = n;
		queue->add(fillme);
		printf("producer: added %d\n", fillme.x);
	}
}

void *consumer_func( void *ptr ) {
	tw_safeFIFO<testdat, TESTAlloc> *queue = (tw_safeFIFO<testdat, TESTAlloc> *) ptr;
	testdat inbound;
	while(queue->removeOrBlock(inbound)) {
		printf("consumer: recv %d\n", inbound.x);
		if(inbound.x == __STOP_VAL)
			break;
	}
}

int main() {

	pthread_t consumer, producer;

	int ret1, ret2;

	tw_safeFIFO<testdat, TESTAlloc> queue;

	ret1 = pthread_create( &producer, NULL, producer_func, (void*) &queue);
    ret2 = pthread_create( &consumer, NULL, consumer_func, (void*) &queue);

	pthread_join( consumer, NULL );
	pthread_join( producer, NULL );

	printf("test done.\n");
}
