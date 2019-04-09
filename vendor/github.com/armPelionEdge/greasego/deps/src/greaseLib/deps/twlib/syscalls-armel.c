/*
 * syscalls-armel.c
 *
 *  Created on: Apr 9, 2010
 *      Author: ed
 */

/*
 * syscalls.c
 *
 *  Created on: Apr 9, 2010
 *      Author: ed
 */
//#include <linux/unistd.h>
//#include <linux/syscalls.h>
#include <sys/types.h>
#include <unistd.h>

// arch independent

long _zdb_getLWPnum() {
//	return (int) ::gettid();
	// TODO: need syscall number for ARMel
//	return (long int)syscall(224);
	return 101010;
//	return sys_gettid(void);
}
