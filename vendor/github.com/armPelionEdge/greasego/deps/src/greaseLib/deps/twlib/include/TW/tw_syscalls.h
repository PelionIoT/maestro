/*
 * zdb-syscalls.h
 *
 *  Created on: Apr 9, 2010
 *      Author: ed
 */

#ifndef TW_SYSCALLS_H_
#define TW_SYSCALLS_H_
#ifdef __cplusplus
extern "C" {
#endif
long _TW_getLWPnum(); // get LWP number using gettid() if available on the arch
#ifdef __cplusplus
}
#endif
#endif /* TW_SYSCALLS_H_ */
