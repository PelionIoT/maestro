// -*- C++ -*-

//==========================================================================
/**
 *  @file   config-lite.h
 *
 *  $Id: config-lite.h 92580 2010-11-15 09:48:02Z johnnyw $
 *
 *  @author (Originally in OS.h)Doug Schmidt <schmidt@cs.wustl.edu>
 *  @author Jesper S. M|ller<stophph@diku.dk>
 *  @author and a cast of thousands...
 *
 *  This file contains the contents of the old config-all.h in order to
 *  avoid a circular dependency problem caused by some of the new
 *  includes added to config-all.h, e.g., OS_main.h.
 */
//==========================================================================

#ifndef TW_CONFIG_LITE_H
#define TW_CONFIG_LITE_H

//#include /**/ "ace/pre.h"

#include <tw_pre.h>
#include <tw_defs.h>

//#include "ace/config-macros.h"

#if !defined (TW_LACKS_PRAGMA_ONCE)
# pragma once
#endif /* TW_LACKS_PRAGMA_ONCE */

// ============================================================================
// UNICODE macros (to be added later)
// ============================================================================

// Get the unicode (i.e. TW_TCHAR) defines
//# include "ace/ace_wchar.h"

// ============================================================================
// at_exit declarations
// ============================================================================

TW_BEGIN_VERSIONED_NAMESPACE_DECL

// Marker for cleanup, used by TW_Exit_Info.
extern int ace_exit_hook_marker;

TW_END_VERSIONED_NAMESPACE_DECL

// For use by <TW_OS::exit>.
extern "C"
{
  typedef void (*TW_EXIT_HOOK) (void);
}

// Signature for registering a cleanup function that is used by the
// TW_Object_Manager and the TW_Thread_Manager.
# if defined (TW_HAS_SIG_C_FUNC)
extern "C" {
# endif /* TW_HAS_SIG_C_FUNC */
typedef void (*TW_CLEANUP_FUNC)(void *object, void *param) /* throw () */;
# if defined (TW_HAS_SIG_C_FUNC)
}
# endif /* TW_HAS_SIG_C_FUNC */

// ============================================================================
// log_msg declarations
// ============================================================================

TW_BEGIN_VERSIONED_NAMESPACE_DECL

# if defined (TW_HAS_WIN32_STRUCTURAL_EXCEPTIONS)
typedef int (*TW_SEH_EXCEPT_HANDLER)(void *);
// Prototype of win32 structured exception handler functions.
// They are used to get the exception handling expression or
// as exception handlers.
# endif /* TW_HAS_WIN32_STRUCTURAL_EXCEPTIONS */

class TW_OS_Thread_Descriptor;
class TW_OS_Log_Msg_Attributes;
typedef void (*TW_INIT_LOG_MSG_HOOK) (TW_OS_Log_Msg_Attributes &attr
# if defined (TW_HAS_WIN32_STRUCTURAL_EXCEPTIONS)
                                       , TW_SEH_EXCEPT_HANDLER selector
                                       , TW_SEH_EXCEPT_HANDLER handler
# endif /* TW_HAS_WIN32_STRUCTURAL_EXCEPTIONS */
                                       );
typedef void (*TW_INHERIT_LOG_MSG_HOOK) (TW_OS_Thread_Descriptor*,
                                          TW_OS_Log_Msg_Attributes &);

typedef void (*TW_CLOSE_LOG_MSG_HOOK) (void);

typedef void (*TW_SYNC_LOG_MSG_HOOK) (const TW_TCHAR *prog_name);

typedef TW_OS_Thread_Descriptor *(*TW_THR_DESC_LOG_MSG_HOOK) (void);

TW_END_VERSIONED_NAMESPACE_DECL

/**
 * @deprecated TW_DECLARE_STL_REVERSE_ITERATORS is a crutch to be
 *             used until all C++ compiler supported by ACE support
 *             the standard reverse_iterator adapters.
 * @internal   TW_DECLARE_STL_REVERSE_ITERATORS is not meant for use
 *             outside of ACE.
 */
// STL reverse_iterator declaration generator
// Make sure you include <iterator> in the file you're using this
// generator, and that the following traits are available:
//
//   iterator
//   const_iterator
//   value_type
//   reference
//   pointer
//   const_reference
//   const_pointer
//   difference_type
//
// Once all C++ compilers support the standard reverse_iterator
// adapters, we can drop this generator macro or at least drop the
// MSVC++ or Sun Studio preprocessor conditional blocks.
#if defined (__SUNPRO_CC) && __SUNPRO_CC <= 0x5100 \
      && !defined (_STLPORT_VERSION)
  // If we're not using the stlport4 C++ library (which has standard
  // iterators), we need to ensure this is included in order to test
  // the _RWSTD_NO_CLASS_PARTIAL_SPEC feature test macro below.
# include <Cstd/stdcomp.h>
#endif /* __SUNPRO_CC <= 0x5100 */
#if (defined (_MSC_VER) && (_MSC_VER <= 1310) && defined (_WIN64)) \
    || defined (TW_HAS_BROKEN_STD_REVERSE_ITERATOR)
  // VC 7.1 and the latest 64-bit platform SDK still don't define a standard
  // compliant reverse_iterator adapter.
# define TW_DECLARE_STL_REVERSE_ITERATORS \
  typedef std::reverse_iterator<iterator, value_type> reverse_iterator; \
  typedef std::reverse_iterator<const_iterator, \
                                value_type const> const_reverse_iterator;
#elif defined (__SUNPRO_CC) && __SUNPRO_CC <= 0x5100 \
      && defined (_RWSTD_NO_CLASS_PARTIAL_SPEC)
# define TW_DECLARE_STL_REVERSE_ITERATORS \
  typedef std::reverse_iterator<iterator, \
                                std::input_iterator_tag, \
                                value_type, \
                                reference, \
                                pointer, \
                                difference_type> reverse_iterator; \
  typedef std::reverse_iterator<const_iterator, \
                                std::input_iterator_tag, \
                                value_type const, \
                                const_reference, \
                                const_pointer, \
                                difference_type> const_reverse_iterator;
#else
# define TW_DECLARE_STL_REVERSE_ITERATORS \
  typedef std::reverse_iterator<iterator>       reverse_iterator; \
  typedef std::reverse_iterator<const_iterator> const_reverse_iterator;
#endif  /* _MSC_VER && _WIN64 */


//#include /**/ "ace/post.h"
#include <tw_post.h>

#endif /* TW_CONFIG_LITE_H */
