// -*- C++ -*-

//=============================================================================
/**
 *  @file    Trace.h
 *
 *  $Id: Trace.h 87823 2009-11-30 12:38:34Z johnnyw $
 *
 *  @author Douglas C. Schmidt <schmidt@cs.wustl.edu>
 */
//=============================================================================

#ifndef TW_Trace_H
#define TW_Trace_H

#include <tw_defs.h>

#include <tw_pre.h>

//#include /**/ "ace/ACE_export.h"

#if !defined (ACE_LACKS_PRAGMA_ONCE)
# pragma once
#endif /* ACE_LACKS_PRAGMA_ONCE */

ACE_BEGIN_VERSIONED_NAMESPACE_DECL

/**
 * @class TW_Trace
 *
 * @brief A C++ trace facility that keeps track of which methods are
 * entered and exited.
 *
 * This class uses C++ constructors and destructors to automate
 * the TW_Trace nesting.  In addition, thread-specific storage
 * is used to enable multiple threads to work correctly.
 */
class TW_Export TW_Trace
{
public:
  // = Initialization and termination methods.

  /// Perform the first part of the trace, which prints out the string
  /// N, the LINE, and the ACE_FILE as the function is entered.
  TW_Trace (const TW_TCHAR *n,
             int line = 0,
             const TW_TCHAR *file = TW_TEXT (""));

  /// Perform the second part of the trace, which prints out the NAME
  /// as the function is exited.
  ~TW_Trace (void);

  // = Control the tracing level.
  /// Determine if tracing is enabled or not
  static bool is_tracing(void);

  /// Enable the tracing facility.
  static void start_tracing (void);

  /// Disable the tracing facility.
  static void stop_tracing (void);

  /// Change the nesting indentation level.
  static void set_nesting_indent (int indent);

  /// Get the nesting indentation level.
  static int get_nesting_indent (void);

  /// Dump the state of an object.
  void dump (void) const;

private:
  // Keeps track of how deeply the call stack is nested (this is
  // maintained in thread-specific storage to ensure correctness in
  // multiple threads of control.

  /// Name of the method we are in.
  const TW_TCHAR *name_;

  /// Keeps track of how far to indent per trace call.
  static int nesting_indent_;

  /// Is tracing enabled?
  static bool enable_tracing_;

  /// Default values.
  enum
  {
    DEFAULT_INDENT  = 3,
    DEFAULT_TRACING = 1
  };
};

ACE_END_VERSIONED_NAMESPACE_DECL

#include <tw_post.h>

#endif /* TW_Trace_H */
