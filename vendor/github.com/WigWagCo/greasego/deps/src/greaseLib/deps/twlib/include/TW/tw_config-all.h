// -*- C++ -*-

//==========================================================================
/**
 *  @file   config-all.h
 *
 *  $Id: config-all.h 84216 2009-01-22 18:34:40Z johnnyw $
 *
 *  @author (Originally in OS.h)Doug Schmidt <schmidt@cs.wustl.edu>
 *  @author Jesper S. M|ller<stophph@diku.dk>
 *  @author and a cast of thousands...
 */
//==========================================================================

#ifndef TW_CONFIG_ALL_H
#define TW_CONFIG_ALL_H

//#include /**/ "ace/pre.h"
#include <tw_pre.h>
#include <tw_defs.h>
#include <tw_config-lite.h>

//#include "ace/config-lite.h"

#if !defined (TW_LACKS_PRAGMA_ONCE)
# pragma once
#endif /* TW_LACKS_PRAGMA_ONCE */

// This is used to indicate that a platform doesn't support a
// particular feature.
#if defined TW_HAS_VERBOSE_NOTSUP
  // Print a console message with the file and line number of the
  // unsupported function.
# include "OS/OS_NS_stdio.h"
# define TW_NOTSUP_RETURN(FAILVALUE) do { errno = ENOTSUP; TW_OS::fprintf (stderr, TW_TEXT ("TW_NOTSUP: %s, line %d\n"), __FILE__, __LINE__); return FAILVALUE; } while (0)
# define TW_NOTSUP do { errno = ENOTSUP; TW_OS::fprintf (stderr, TW_TEXT ("TW_NOTSUP: %s, line %d\n"), __FILE__, __LINE__); return; } while (0)
#else /* ! TW_HAS_VERBOSE_NOTSUP */
# define TW_NOTSUP_RETURN(FAILVALUE) do { errno = ENOTSUP ; return FAILVALUE; } while (0)
# define TW_NOTSUP do { errno = ENOTSUP; return; } while (0)
#endif /* ! TW_HAS_VERBOSE_NOTSUP */

// ----------------------------------------------------------------

# define TW_TRACE_IMPL(X) TW_Trace ____ (TW_TEXT (X), __LINE__, TW_TEXT (__FILE__))

// By default tracing is turned off.
#if !defined (TW_NTRACE)
#  define TW_NTRACE 1
#endif /* TW_NTRACE */

#if (TW_NTRACE == 1)
#  define TW_TRACE(X)
#else
#  if !defined (TW_HAS_TRACE)
#    define TW_HAS_TRACE
#  endif /* TW_HAS_TRACE */
#  define TW_TRACE(X) TW_TRACE_IMPL(X)
#  include "ace/Trace.h"
#endif /* TW_NTRACE */

// By default we perform no tracing on the OS layer, otherwise the
// coupling between the OS layer and Log_Msg is too tight.  But the
// application can override the default if they wish to.
#if !defined (TW_OS_NTRACE)
#  define TW_OS_NTRACE 1
#endif /* TW_OS_NTRACE */

#if (TW_OS_NTRACE == 1)
#  define TW_OS_TRACE(X)
#else
#  if !defined (TW_HAS_TRACE)
#    define TW_HAS_TRACE
#  endif /* TW_HAS_TRACE */
#  define TW_OS_TRACE(X) TW_TRACE_IMPL(X)
#  include "ace/Trace.h"
#endif /* TW_OS_NTRACE */

#if !defined (TW_HAS_MONITOR_FRAMEWORK)
# define TW_HAS_MONITOR_FRAMEWORK 1
#endif

#if !defined (TW_HAS_SENDFILE)
# define TW_HAS_SENDFILE 0
#endif

#if !defined (TW_HAS_MONITOR_POINTS)
# define TW_HAS_MONITOR_POINTS 0
#endif

// These includes are here to avoid circular dependencies.
// Keep this at the bottom of the file.  It contains the main macros.
#include "ace/OS_main.h"

//#include /**/ "ace/post.h"
#include <tw_post.h>

#endif /* TW_CONFIG_ALL_H */
