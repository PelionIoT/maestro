/*
    MIT License

    Copyright (c) 2019, Arm Limited and affiliates.

    SPDX-License-Identifier: MIT
    
    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.
*/

#ifndef GREASE_COMMON_TAGS_H
#define GREASE_COMMON_TAGS_H

#include <stdint.h>

#define GREASE_RESERVED_TAGS_START 0xF000
// well known tags

// client code should use these.
#define GREASE_ECHO_TAG     0xFFF1 /* "echo" depicts something from grease_echo */
#define GREASE_CONSOLE_TAG  0xFFF2 /* "console" depicts something which would come from the console */
#define GREASE_NATIVE_TAG   0xFFF3 /* "native"  depicts native code */


#define GREASE_RESERVED_ORIGINS_START 0xFFF0

#define GREASE_GREASEECHO_ORIGIN 0xFFF1

#define GREASE__CONST_TAG( name ) extern const uint32_t name

GREASE__CONST_TAG( GREASE_RESERVED_TAGS_ECHO ); //     0xFFF01   /* "echo" depicts something from grease_echo */
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_CONSOLE ); //     0xFFF01   /* "echo" depicts something from grease_echo */
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_NATIVE ); //     0xFFF01   /* "echo" depicts something from grease_echo */
//#define GREASE_TAG_CONSOLE  0xFFF02   /* "console" depicts something which would come from the console */
//#define GREASE_TAG_NATIVE   0xFFF03   /* "native"  depicts native code */

// implemented in grease_lib.cc
// client code should never use these
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_AUTH );
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_AUTHPRIV);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_CRON);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_DAEMON);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_FTP);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_KERN);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LPR);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_MAIL);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_MARK);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_NEWS);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_SECURITY);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_SYSLOG);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_USER);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_UUCP);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LOCAL0);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LOCAL1);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LOCAL2);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LOCAL3);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LOCAL4);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LOCAL5);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LOCAL6);
GREASE__CONST_TAG( GREASE_RESERVED_TAGS_SYS_LOCAL7);


#endif
