// WigWag LLC
// (c) 2010
// tw_assert.cpp
// Author: ed
// Oct 1, 2010/*
// Oct 1, 2010 * tw_assert.cpp
// Oct 1, 2010 *
// Oct 1, 2010 *  Created on: Oct 1, 2010
// Oct 1, 2010 *      Author: ed
// Oct 1, 2010 */

// $Id: Assert.cpp 80826 2008-03-04 14:51:23Z wotte $

#include <TW/tw_assert.h>
#include <TW/tw_log.h>

//ACE_RCSID(ace, Assert, "$Id: Assert.cpp 80826 2008-03-04 14:51:23Z wotte $")

//ACE_BEGIN_VERSIONED_NAMESPACE_DECL

// The following ASSERT macro is courtesy of Alexandre Karev
// <akg@na47sun05.cern.ch>.
void
__ace_assert(const char *file, int line, const ACE_TCHAR *expression)
{
  int error = ACE_Log_Msg::last_error_adapter ();
  ACE_Log_Msg *log = ACE_Log_Msg::instance ();

  log->set (file, line, -1, error, log->restart (),
            log->msg_ostream (), log->msg_callback ());

  log->log (LM_ERROR, ACE_TEXT ("ACE_ASSERT: file %N, line %l assertion failed for '%s'.%a\n"), expression, -1);
}

//ACE_END_VERSIONED_NAMESPACE_DECL
