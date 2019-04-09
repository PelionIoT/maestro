/*
 * tw_types.h
 *
 *  Created on: Jul 31, 2011
 *      Author: ed
 */

#ifndef TW_TYPES_H_
#define TW_TYPES_H_

typedef int tw_size;

#ifdef TW_OS_LACKS_SIZE_T
typedef int size_t;

#else
#include <stdlib.h>
#endif


//typedef int size_t;

#endif /* TW_TYPES_H_ */
