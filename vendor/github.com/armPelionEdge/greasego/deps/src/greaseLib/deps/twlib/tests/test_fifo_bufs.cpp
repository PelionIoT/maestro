// WigWag LLC
// (c) 2010
// test_fifo.cpp
// Author: ed

#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <poll.h>

#include <TW/tw_log.h>
#include <TW/tw_fifo.h>
#include <TW/tw_task.h>
#include <TW/tw_bufblk.h>


#define __SLEEP_VAL (50*1000)   // sleep for 50 ms
#define __STOP_VAL 25

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


const char *teststr1 = "------- %d --------";
const char *teststr2 = "+++++++ %d ++++++++";

typedef Allocator<Alloc_Std> TESTAlloc;

// custom queue built with tw_safeFIFO
class test_str_queue : public tw_safeFIFO<BufBlk<TESTAlloc>, Allocator<Alloc_Std> > {};


// producer thread object
class producer_task : public Task<test_str_queue> {
public:
	test_str_queue *worktask( test_str_queue *queue ) {
		char buffer[25];
		//		tw_safeFIFO<testdat> *queue = (tw_safeFIFO<testdat> *) ptr;
		int n = 0;
		BufBlk<TESTAlloc> *blk = NULL;
		BufBlk<TESTAlloc> *blk2 = NULL;


		// fill it up with 10 objs to make a better test...
		for(n=0;n<=10;n++) {
			blk = new BufBlk<TESTAlloc>( 30 );
			int c = sprintf(buffer, teststr1, n);
			blk->copyFrom( buffer, c + 1 );
			queue->add(*blk);
			printf("producer: added %d\n", n);
			blk->release();
		}

		for(n=0;n<=5;n++) {
			blk = new BufBlk<TESTAlloc>( 30 );
			int c = sprintf(buffer, teststr1, n);
			blk->copyFrom( buffer, c ); // don't copy the null - we are making concat
			blk2 = new BufBlk<TESTAlloc>( 30 );
			c = sprintf(buffer, teststr2, n);
			blk2->copyFrom( buffer, c + 1 );
			blk->setNexblk(blk2);
			queue->add(*blk);
			printf("producer: added linked %d\n", n);
			blk->release();
		}

		for(n=0;n<=__STOP_VAL;n++) {
			usleep(__SLEEP_VAL);
			blk = new BufBlk<TESTAlloc>( 30 );
			int c = sprintf(buffer, teststr1, n);
			blk->copyFrom( buffer, c + 1 );
			queue->add(*blk);
			printf("producer: added %d\n", n);
			blk->release();
		}



		return NULL;
	}
};

// consumer thread object
class consumer_task : public Task<test_str_queue> {
public:
	test_str_queue *worktask( test_str_queue *queue ) {
		BufBlk<TESTAlloc> inbound;
		char buffer[100];
		char buf2[10];
		int walk = 0;
		int x = 0;
		int sz = 0;
		while(queue->removeOrBlock(inbound)) {
			if(x < 11) {
			printf("consumer: recv %s\n", inbound.rd_ptr());
			} else {
				BufBlkIter<TESTAlloc> *it = new BufBlkIter<TESTAlloc>( inbound );
				printf("consumer: ");
				if(x < 13) { // two different tests here...
					walk = 0;
					int out = 0;
					while(it->copyNextChunks( (char *)(buffer + walk),100-walk,out)) {
						walk += out;
						printf("[chnk] %d ",out);
					}
					printf(" recv [copyNextChunks] %s\n", buffer);
				} else {
					walk = 0;
					sz = 0;
					char *look;
					while(it->getNextChunk(look,sz,10)) {
						::memcpy(buffer+walk,look,sz);
						walk += sz;
						printf("[cp%d]",sz);
					}
					printf(" recv [getNextChunk] %s\n", buffer);
				}
				delete it;
			}
			x++;
			if(x > __STOP_VAL + 16)
				break;
		}
//		inbound.release(); // note - you can't release a stack allocated elem - it will release when it drops off the stack
		return NULL;
	}
};

namespace TWlib {
template<>
test_str_queue *TWlib::Task<test_str_queue>::worktask( test_str_queue *queue ) {
	return NULL;
}
}

int main() {

//	pthread_t consumer, producer;

	consumer_task consumer;
	producer_task producer;

	int ret1, ret2;

	test_str_queue queue;

	consumer.startTask(&queue);
	producer.startTask(&queue);

	if(producer.isRunning())
		printf("producer is running\n");
	if(consumer.isRunning())
		printf("consumer is running\n");

	consumer.waitForTask();
	producer.waitForTask();

	if(!producer.isRunning())
		printf("producer is completed\n");
	if(!consumer.isRunning())
		printf("consumer is completed\n");

	printf("test done.\n");
	return 0;
}
