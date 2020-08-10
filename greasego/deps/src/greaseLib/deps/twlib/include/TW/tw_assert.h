// WigWag LLC
// (c) 2010
// tw_assert.h
// Author: ed
// Oct 1, 2010

/*
 * tw_assert.h
 *
 *  Created on: Oct 1, 2010
 *      Author: ed
 */

#ifndef TW_ASSERT_H_
#define TW_ASSERT_H_

// from ACE

namespace TWlib {
// The following ASSERT macro is courtesy of Alexandre Karev
// <akg@na47sun05.cern.ch>.
void
__ace_assert(const char *file, int line, const char *expression)
{
  int error = ACE_Log_Msg::last_error_adapter ();
  ACE_Log_Msg *log = ACE_Log_Msg::instance ();

  log->set (file, line, -1, error, log->restart (),
            log->msg_ostream (), log->msg_callback ());

  log->log (LM_ERROR, ACE_TEXT ("ACE_ASSERT: file %N, line %l assertion failed for '%s'.%a\n"), expression, -1);
}

}
#endif /* TW_ASSERT_H_ */
