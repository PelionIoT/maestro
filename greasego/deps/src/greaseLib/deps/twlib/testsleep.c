// Framez LLC
// (c) 2010
// testsleep.c
// Author: ed
/** a helper program execute program X times, and sleep b/t each execution
 *
 */

#include <poll.h>
#include <stdio.h>

int usleep(usec)
	unsigned int usec; /* microseconds */
{
	static subtotal = 0; /* microseconds */
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


char *usage = "%s - sleep for N milliseconds\n"
		      "Usage:\n"
		      "%s {time-in-ms}\n";

int main(int argc, char *argv[]) {
	if(argc < 2) {
		printf(usage,argv[0],argv[0]);
		return -1;
	}

	int ms = atoi(argv[1]);

	if(ms < 0) {
		printf("invalid value for timeout!\n\n");
		return -1;
	} else
		usleep(ms*1000);

	return 0;
}
