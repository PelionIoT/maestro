/*
 * tw_defs.h
 *
 *  Created on: Jul 31, 2011
 *      Author: ed
 */

#ifndef TW_DEFS_H_
#define TW_DEFS_H_

// CONFIG
//#include <tw_config-macros.h>
//#include <tw_config-lite.h>

#define TW_Export

#define TW_TCHAR char

#include <stdint.h>

#define TW_UINT32 uint32_t

// how to represent char strings
#define TW_TEXT(s) s


#define TW_INLINE

// legacy ACE stuff we may be able to kill
#define TW_BEGIN_VERSIONED_NAMESPACE_DECL
#define TW_END_VERSIONED_NAMESPACE_DECL
#define TW_ALLOC_HOOK_DEFINE(x)

//#define ACE_BEGIN_VERSIONED_NAMESPACE_DECL
//#define ACE_END_VERSIONED_NAMESPACE_DECL
//#define ACE_ALLOC_HOOK_DEFINE(x)

#endif /* TW_DEFS_H_ */
