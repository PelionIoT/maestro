// $Id: Trace.cpp 91286 2010-08-05 09:04:31Z johnnyw $

//#include "ace/Trace.h"
#include <TW/tw_trace.h>
// Turn off tracing for the duration of this file.
#if defined (ACE_NTRACE)
#undef ACE_NTRACE
#endif /* ACE_NTRACE */
#define ACE_NTRACE 1

//#include "ace/Log_Msg.h"
//#include "ace/Object_Manager_Base.h"

#include <TW/tw_log.h>

ACE_BEGIN_VERSIONED_NAMESPACE_DECL

// = Static initialization.

// Keeps track of how far to indent per trace call.
int TW_Trace::nesting_indent_ = TW_Trace::DEFAULT_INDENT;

// Is tracing enabled?
bool TW_Trace::enable_tracing_ = TW_Trace::DEFAULT_TRACING;

ACE_ALLOC_HOOK_DEFINE(TW_Trace)

void
TW_Trace::dump (void) const
{
#if defined (ACE_HAS_DUMP)
#endif /* ACE_HAS_DUMP */
}

// Determine whether or not tracing is enabled

bool
TW_Trace::is_tracing (void)
{
  return TW_Trace::enable_tracing_;
}

// Enable the tracing facility.

void
TW_Trace::start_tracing (void)
{
  TW_Trace::enable_tracing_ = true;
}

// Disable the tracing facility.

void
TW_Trace::stop_tracing (void)
{
  TW_Trace::enable_tracing_ = false;
}

// Change the nesting indentation level.

void
TW_Trace::set_nesting_indent (int indent)
{
  TW_Trace::nesting_indent_ = indent;
}

// Get the nesting indentation level.

int
TW_Trace::get_nesting_indent (void)
{
  return TW_Trace::nesting_indent_;
}

// Perform the first part of the trace, which prints out the string N,
// the LINE, and the ACE_FILE as the function is entered.

TW_Trace::TW_Trace (const ACE_TCHAR *n,
                      int line,
                      const ACE_TCHAR *file)
{
#if defined (ACE_NLOGGING)
  ACE_UNUSED_ARG (line);
  ACE_UNUSED_ARG (file);
#endif /* ACE_NLOGGING */

  this->name_ = n;

  // If ACE has not yet been initialized, don't try to trace... there's
  // too much stuff not yet initialized.
  if (TW_Trace::enable_tracing_ && !ACE_OS_Object_Manager::starting_up ())
    {
      ACE_Log_Msg *lm = ACE_LOG_MSG;
      if (lm->tracing_enabled ()
          && lm->trace_active () == 0)
        {
          lm->trace_active (1);
          ACE_DEBUG ((LM_TRACE,
                      ACE_TEXT ("%*s(%t) calling %s in file `%s' on line %d\n"),
                      TW_Trace::nesting_indent_ * lm->inc (),
                      ACE_TEXT (""),
                      this->name_,
                      file,
                      line));
          lm->trace_active (0);
        }
    }
}

// Perform the second part of the trace, which prints out the NAME as
// the function is exited.

TW_Trace::~TW_Trace (void)
{
  // If ACE has not yet been initialized, don't try to trace... there's
  // too much stuff not yet initialized.
  if (TW_Trace::enable_tracing_ && !ACE_OS_Object_Manager::starting_up ())
    {
      ACE_Log_Msg *lm = ACE_LOG_MSG;
      if (lm->tracing_enabled ()
          && lm->trace_active () == 0)
        {
          lm->trace_active (1);
          ACE_DEBUG ((LM_TRACE,
                      ACE_TEXT ("%*s(%t) leaving %s\n"),
                      TW_Trace::nesting_indent_ * lm->dec (),
                      ACE_TEXT (""),
                      this->name_));
          lm->trace_active (0);
        }
    }
}

ACE_END_VERSIONED_NAMESPACE_DECL
