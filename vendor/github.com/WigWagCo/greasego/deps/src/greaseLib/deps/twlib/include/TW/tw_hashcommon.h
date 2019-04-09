// WigWag LLC
// (c) 2011
// tw_hashcommon.h
// Author: ed
// Jan 4, 2012

/*
 * tw_hashcommon.h
 *
 *  Created on: Jan 4, 2012
 *      Author: ed
 */

#ifndef TW_HASHCOMMON_H_
#define TW_HASHCOMMON_H_

#include <TW/tw_log.h>

// Extra debugging can be achived by defining TW_HASH_DEBUG

#ifdef TW_HASH_DEBUG
#ifndef __TW_HASH_DEBUG
#define __TW_HASH_DEBUG( s, ... ) TW_DEBUG( s, __VA_ARGS__ )
#endif
#else
#ifndef __TW_HASH_DEBUG
#define __TW_HASH_DEBUG( s, ... )  { }
#endif
#endif

#ifdef TW_HASH_DEBUG
#ifndef __TW_HASH_DEBUGL
#define __TW_HASH_DEBUGL( s, ... ) TW_DEBUG_L( s, __VA_ARGS__ )
#endif
#else
#ifndef __TW_HASH_DEBUGL
#define __TW_HASH_DEBUGL( s, ... )  { }
#endif
#endif

#ifdef TW_HASH_DEBUG
#ifndef __TW_HASH_DEBUGLT
#define __TW_HASH_DEBUGLT( s, ... ) TW_DEBUG_LT( s, __VA_ARGS__ )
#endif
#else
#ifndef __TW_HASH_DEBUGLT
#define __TW_HASH_DEBUGLT( s, ... )  { }
#endif
#endif



#endif /* TW_HASHCOMMON_H_ */
