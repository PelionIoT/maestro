// WigWag LLC
// (c) 2010
// test_fifo.cpp
// Author: ed

#include <stdio.h>
#include <pthread.h>
#include <poll.h>

//#define _TW_FIFO_DEBUG_ON
#include <TW/tw_fifo.h>

#include <TW/tw_task.h>


#define __SLEEP_VAL (50*1000)   // sleep for 50 ms
#define __STOP_VAL 20

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

typedef Allocator<Alloc_Std> TESTAlloc;

// custom queue built with tw_safeFIFO
class test_dat_queue : public tw_safeFIFO<testdat, TESTAlloc> {};

// producer thread object
class producer_task : public Task<test_dat_queue> {
public:
	test_dat_queue *worktask( test_dat_queue *queue ) {
//		tw_safeFIFO<testdat> *queue = (tw_safeFIFO<testdat> *) ptr;
		int n = 0;
		testdat fillme;
		printf("producer task.\n");

		fillme.x = 98;
		queue->addToHead(fillme);
		printf("producer: addToHead %d\n", fillme.x);

		// fill it up with 10 objs to make a better test...
		for(n=0;n<=10;n++) {
	//		usleep(__SLEEP_VAL);
			fillme.x = 1;
//			printf("Add. (queue %x)\n", queue);
			queue->add(fillme);
			printf("producer: added %d\n", fillme.x);
		}

		fillme.x = 99;
		queue->addToHead(fillme);
		printf("producer: addToHead %d\n", fillme.x);

		for(n=0;n<=__STOP_VAL;n++) {
			usleep(__SLEEP_VAL);
			fillme.x = n;
			queue->add(fillme);
			printf("producer: added %d\n", fillme.x);
		}
		return NULL;
	}
};

// consumer thread object
class consumer_task : public Task<test_dat_queue> {
public:
/*	test_dat_queue *worktask( test_dat_queue *queue ) {
		testdat inbound;
		printf("overloaded template func.\n");
		while(queue->removeOrBlock(inbound)) {
			printf("consumer: recv %d\n", inbound.x);
			if(inbound.x == __STOP_VAL)
				break;
		}
		return NULL;
	}
	*/
};

namespace TWlib {
template<>
test_dat_queue *TWlib::Task<test_dat_queue>::worktask( test_dat_queue *queue ) {
	testdat inbound;
	printf("template func\n");
	while(queue->removeOrBlock(inbound)) {
		printf("consumer: recv %d\n", inbound.x);
		if(inbound.x == __STOP_VAL)
			break;
	}
	return NULL;
}
}

int main() {

//	pthread_t consumer, producer;

	consumer_task consumer;
	producer_task producer;

	int ret1, ret2;

	test_dat_queue queue;

	consumer.startTask(&queue);
	producer.startTask(&queue);

	if(producer.isRunning())
		printf("producer is running (LWP %d)\n", producer.getLWP());
	if(consumer.isRunning())
		printf("consumer is running (LWP %d)\n", consumer.getLWP());

	consumer.waitForTask();
	producer.waitForTask();

	if(producer.isRunning())
		printf("producer is completed\n");
	if(consumer.isRunning())
		printf("producer is completed\n");

	printf("test done.\n");
}
