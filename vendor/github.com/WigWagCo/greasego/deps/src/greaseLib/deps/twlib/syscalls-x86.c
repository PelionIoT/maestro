/*
 * syscalls.c
 *
 *  Created on: Apr 9, 2010
 *      Author: ed
 */
//#include <linux/unistd.h>
//#include <linux/syscalls.h>
#include <sys/syscall.h>
#include <sys/types.h>
#include <unistd.h>

// arch independent
long _TW_getLWPnum() {
//	return (int) ::gettid();
	return (long int)syscall(SYS_gettid);
//	return sys_gettid(void);
}
