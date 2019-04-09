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

/*
 * error-common.cc
 *
 *  Created on: Sep 3, 2014
 *      Author: ed
 * (c) 2014, WigWag Inc.
 */



#include "error-common.h"



#include <string.h>
//extern char *strdup(const char *s);

extern "C" char *local_strdup_safe(const char *s);

#ifndef GREASE_LIB

namespace _errcmn {


void DefineConstants(v8::Handle<v8::Object> target) {

#ifdef _ERRCMN_ADD_CONSTS
#include "errcmn_inl.h"
#endif


#ifdef E2BIG
_ERRCMN_DEFINE_CONSTANT_WREV(target, E2BIG);
#endif

#ifdef EACCES
_ERRCMN_DEFINE_CONSTANT_WREV(target, EACCES);
#endif

#ifdef EADDRINUSE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EADDRINUSE);
#endif

#ifdef EADDRNOTAVAIL
_ERRCMN_DEFINE_CONSTANT_WREV(target, EADDRNOTAVAIL);
#endif

#ifdef EAFNOSUPPORT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EAFNOSUPPORT);
#endif

#ifdef EAGAIN
_ERRCMN_DEFINE_CONSTANT_WREV(target, EAGAIN);
#endif

#ifdef EALREADY
_ERRCMN_DEFINE_CONSTANT_WREV(target, EALREADY);
#endif

#ifdef EBADF
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBADF);
#endif

#ifdef EBADMSG
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBADMSG);
#endif

#ifdef EBUSY
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBUSY);
#endif

#ifdef ECANCELED
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECANCELED);
#endif

#ifdef ECHILD
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECHILD);
#endif

#ifdef ECONNABORTED
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECONNABORTED);
#endif

#ifdef ECONNREFUSED
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECONNREFUSED);
#endif

#ifdef ECONNRESET
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECONNRESET);
#endif

#ifdef EDEADLK
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDEADLK);
#endif

#ifdef EDESTADDRREQ
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDESTADDRREQ);
#endif

#ifdef EDOM
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDOM);
#endif

#ifdef EDQUOT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDQUOT);
#endif

#ifdef EEXIST
_ERRCMN_DEFINE_CONSTANT_WREV(target, EEXIST);
#endif

#ifdef EFAULT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EFAULT);
#endif

#ifdef EFBIG
_ERRCMN_DEFINE_CONSTANT_WREV(target, EFBIG);
#endif

#ifdef EHOSTUNREACH
_ERRCMN_DEFINE_CONSTANT_WREV(target, EHOSTUNREACH);
#endif

#ifdef EIDRM
_ERRCMN_DEFINE_CONSTANT_WREV(target, EIDRM);
#endif

#ifdef EILSEQ
_ERRCMN_DEFINE_CONSTANT_WREV(target, EILSEQ);
#endif

#ifdef EINPROGRESS
_ERRCMN_DEFINE_CONSTANT_WREV(target, EINPROGRESS);
#endif

#ifdef EINTR
_ERRCMN_DEFINE_CONSTANT_WREV(target, EINTR);
#endif

#ifdef EINVAL
_ERRCMN_DEFINE_CONSTANT_WREV(target, EINVAL);
#endif

#ifdef EIO
_ERRCMN_DEFINE_CONSTANT_WREV(target, EIO);
#endif

#ifdef EISCONN
_ERRCMN_DEFINE_CONSTANT_WREV(target, EISCONN);
#endif

#ifdef EISDIR
_ERRCMN_DEFINE_CONSTANT_WREV(target, EISDIR);
#endif

#ifdef ELOOP
_ERRCMN_DEFINE_CONSTANT_WREV(target, ELOOP);
#endif

#ifdef EMFILE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMFILE);
#endif

#ifdef EMLINK
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMLINK);
#endif

#ifdef EMSGSIZE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMSGSIZE);
#endif

#ifdef EMULTIHOP
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMULTIHOP);
#endif

#ifdef ENAMETOOLONG
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENAMETOOLONG);
#endif

#ifdef ENETDOWN
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENETDOWN);
#endif

#ifdef ENETRESET
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENETRESET);
#endif

#ifdef ENETUNREACH
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENETUNREACH);
#endif

#ifdef ENFILE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENFILE);
#endif

#ifdef ENOBUFS
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOBUFS);
#endif

#ifdef ENODATA
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENODATA);
#endif

#ifdef ENODEV
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENODEV);
#endif

#ifdef ENOENT
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOENT);
#endif

#ifdef ENOEXEC
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOEXEC);
#endif

#ifdef ENOLCK
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOLCK);
#endif

#ifdef ENOLINK
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOLINK);
#endif

#ifdef ENOMEM
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOMEM);
#endif

#ifdef ENOMSG
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOMSG);
#endif

#ifdef ENOPROTOOPT
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOPROTOOPT);
#endif

#ifdef ENOSPC
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOSPC);
#endif

#ifdef ENOSR
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOSR);
#endif

#ifdef ENOSTR
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOSTR);
#endif

#ifdef ENOSYS
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOSYS);
#endif

#ifdef ENOTCONN
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTCONN);
#endif

#ifdef ENOTDIR
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTDIR);
#endif

#ifdef ENOTEMPTY
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTEMPTY);
#endif

#ifdef ENOTSOCK
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTSOCK);
#endif

#ifdef ENOTSUP
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTSUP);
#endif

#ifdef ENOTTY
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTTY);
#endif

#ifdef ENXIO
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENXIO);
#endif

#ifdef EOPNOTSUPP
_ERRCMN_DEFINE_CONSTANT_WREV(target, EOPNOTSUPP);
#endif

#ifdef EOVERFLOW
_ERRCMN_DEFINE_CONSTANT_WREV(target, EOVERFLOW);
#endif

#ifdef EPERM
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPERM);
#endif

#ifdef EPIPE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPIPE);
#endif

#ifdef EPROTO
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPROTO);
#endif

#ifdef EPROTONOSUPPORT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPROTONOSUPPORT);
#endif

#ifdef EPROTOTYPE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPROTOTYPE);
#endif

#ifdef ERANGE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ERANGE);
#endif

#ifdef EROFS
_ERRCMN_DEFINE_CONSTANT_WREV(target, EROFS);
#endif

#ifdef ESPIPE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESPIPE);
#endif

#ifdef ESRCH
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESRCH);
#endif

#ifdef ESTALE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESTALE);
#endif

#ifdef ETIME
_ERRCMN_DEFINE_CONSTANT_WREV(target, ETIME);
#endif

#ifdef ETIMEDOUT
_ERRCMN_DEFINE_CONSTANT_WREV(target, ETIMEDOUT);
#endif

#ifdef ETXTBSY
_ERRCMN_DEFINE_CONSTANT_WREV(target, ETXTBSY);
#endif

#ifdef EWOULDBLOCK
_ERRCMN_DEFINE_CONSTANT_WREV(target, EWOULDBLOCK);
#endif

#ifdef EXDEV
_ERRCMN_DEFINE_CONSTANT_WREV(target, EXDEV);
#endif

//#define	ENOBUFS		105	/* No buffer space available */
#ifdef ENOBUFS
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOBUFS);
#endif

//#define	ENODEV		19	/* No such device */
#ifdef ENODEV
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENODEV);
#endif

//#define	ENOTDIR		20	/* Not a directory */
#ifdef ENOTDIR
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTDIR);
#endif

//------------------------------------------------------

//#define	EISDIR		21	/* Is a directory */
#ifdef EISDIR
_ERRCMN_DEFINE_CONSTANT_WREV(target, EISDIR);
#endif

//#define	EINVAL		22	/* Invalid argument */
#ifdef EINVAL
_ERRCMN_DEFINE_CONSTANT_WREV(target, EINVAL);
#endif

//#define	ENFILE		23	/* File table overflow */
#ifdef ENFILE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENFILE);
#endif

//#define	EMFILE		24	/* Too many open files */
#ifdef EMFILE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMFILE);
#endif

//#define	ENOTTY		25	/* Not a typewriter */
#ifdef ENOTTY
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTTY);
#endif

//#define	ETXTBSY		26	/* Text file busy */
#ifdef ETXTBSY
_ERRCMN_DEFINE_CONSTANT_WREV(target, ETXTBSY);
#endif

//#define	EFBIG		27	/* File too large */
#ifdef EFBIG
_ERRCMN_DEFINE_CONSTANT_WREV(target, EFBIG);
#endif

//#define	ENOSPC		28	/* No space left on device */
#ifdef ENOSPC
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOSPC);
#endif

//#define	ESPIPE		29	/* Illegal seek */
#ifdef ESPIPE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESPIPE);
#endif

//#define	EROFS		30	/* Read-only file system */
#ifdef EROFS
_ERRCMN_DEFINE_CONSTANT_WREV(target, EROFS);
#endif

//#define	EMLINK		31	/* Too many links */
#ifdef EMLINK
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMLINK);
#endif

//#define	EPIPE		32	/* Broken pipe */
#ifdef EPIPE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPIPE);
#endif

//#define	EDOM		33	/* Math argument out of domain of func */
#ifdef EDOM
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDOM);
#endif

//#define	ERANGE		34	/* Math result not representable */
#ifdef ERANGE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ERANGE);
#endif

//#define	EDEADLK		35	/* Resource deadlock would occur */
#ifdef EDEADLK
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDEADLK);
#endif

//#define	ENAMETOOLONG	36	/* File name too long */
#ifdef ENAMETOOLONG
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENAMETOOLONG);
#endif

//#define	ENOLCK		37	/* No record locks available */
#ifdef ENOLCK
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOLCK);
#endif

//#define	ENOSYS		38	/* Function not implemented */
#ifdef ENOSYS
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOSYS);
#endif

//#define	ENOTEMPTY	39	/* Directory not empty */
#ifdef ENOTEMPTY
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTEMPTY);
#endif

//#define	ELOOP		40	/* Too many symbolic links encountered */
#ifdef ELOOP
_ERRCMN_DEFINE_CONSTANT_WREV(target, ELOOP);
#endif

//#define	EWOULDBLOCK	EAGAIN	/* Operation would block */

//#define	ENOMSG		42	/* No message of desired type */
#ifdef ENOMSG
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOMSG);
#endif

//#define	EIDRM		43	/* Identifier removed */
#ifdef EIDRM
_ERRCMN_DEFINE_CONSTANT_WREV(target, EIDRM);
#endif


//#define	ECHRNG		44	/* Channel number out of range */
#ifdef ECHRNG
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECHRNG);
#endif

//#define	EL2NSYNC	45	/* Level 2 not synchronized */
#ifdef EL2NSYNC
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECHRNG);
#endif

//#define	EL3HLT		46	/* Level 3 halted */
#ifdef EL3HLT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EL3HLT);
#endif

//#define	EL3RST		47	/* Level 3 reset */
#ifdef EL3RST
_ERRCMN_DEFINE_CONSTANT_WREV(target, EL3RST);
#endif

//#define	ELNRNG		48	/* Link number out of range */
#ifdef ELNRNG
_ERRCMN_DEFINE_CONSTANT_WREV(target, ELNRNG);
#endif

//#define	EUNATCH		49	/* Protocol driver not attached */
#ifdef EUNATCH
_ERRCMN_DEFINE_CONSTANT_WREV(target, EUNATCH);
#endif

//#define	ENOCSI		50	/* No CSI structure available */
#ifdef ENOCSI
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOCSI);
#endif

//#define	EL2HLT		51	/* Level 2 halted */
#ifdef EL2HLT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EL2HLT);
#endif

//#define	EBADE		52	/* Invalid exchange */
#ifdef EBADE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBADE);
#endif

//#define	EBADR		53	/* Invalid request descriptor */
#ifdef EBADR
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBADR);
#endif

//#define	EXFULL		54	/* Exchange full */
#ifdef EXFULL
_ERRCMN_DEFINE_CONSTANT_WREV(target, EXFULL);
#endif

//#define	ENOANO		55	/* No anode */
#ifdef ENOANO
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOANO);
#endif

//#define	EBADRQC		56	/* Invalid request code */
#ifdef EBADRQC
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBADRQC);
#endif

//#define	EBADSLT		57	/* Invalid slot */
#ifdef EBADSLT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBADSLT);
#endif

//

//#define	EDEADLOCK	EDEADLK
#ifdef EDEADLOCK
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDEADLOCK);
#endif

//

//#define	EBFONT		59	/* Bad font file format */
#ifdef EBFONT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBFONT);
#endif

//#define	ENOSTR		60	/* Device not a stream */
#ifdef ENOSTR
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOSTR);
#endif

//#define	ENODATA		61	/* No data available */
#ifdef ENODATA
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENODATA);
#endif

//#define	ETIME		62	/* Timer expired */
#ifdef ETIME
_ERRCMN_DEFINE_CONSTANT_WREV(target, ETIME);
#endif

//#define	ENOSR		63	/* Out of streams resources */
#ifdef ENOSR
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOSR);
#endif

//#define	ENONET		64	/* Machine is not on the network */
#ifdef ENONET
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENONET);
#endif

//#define	ENOPKG		65	/* Package not installed */
#ifdef ENOPKG
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOPKG);
#endif

//#define	EREMOTE		66	/* Object is remote */
#ifdef EREMOTE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EREMOTE);
#endif

//#define	ENOLINK		67	/* Link has been severed */
#ifdef ENOLINK
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOLINK);
#endif

//#define	EADV		68	/* Advertise error */
#ifdef EADV
_ERRCMN_DEFINE_CONSTANT_WREV(target, EADV);
#endif

//#define	ESRMNT		69	/* Srmount error */
#ifdef ESRMNT
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESRMNT);
#endif

//#define	ECOMM		70	/* Communication error on send */
#ifdef ECOMM
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECOMM);
#endif

//#define	EPROTO		71	/* Protocol error */
#ifdef EPROTO
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPROTO);
#endif

//#define	EMULTIHOP	72	/* Multihop attempted */
#ifdef EMULTIHOP
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMULTIHOP);
#endif

//#define	EDOTDOT		73	/* RFS specific error */
#ifdef EDOTDOT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDOTDOT);
#endif

//#define	EBADMSG		74	/* Not a data message */
#ifdef EBADMSG
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBADMSG);
#endif

//#define	EOVERFLOW	75	/* Value too large for defined data type */
#ifdef EOVERFLOW
_ERRCMN_DEFINE_CONSTANT_WREV(target, EOVERFLOW);
#endif

//#define	ENOTUNIQ	76	/* Name not unique on network */
#ifdef ENOTUNIQ
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTUNIQ);
#endif

//#define	EBADFD		77	/* File descriptor in bad state */
#ifdef EBADFD
_ERRCMN_DEFINE_CONSTANT_WREV(target, EBADFD);
#endif

//#define	EREMCHG		78	/* Remote address changed */
#ifdef EREMCHG
_ERRCMN_DEFINE_CONSTANT_WREV(target, EREMCHG);
#endif

//#define	ELIBACC		79	/* Can not access a needed shared library */
#ifdef ELIBACC
_ERRCMN_DEFINE_CONSTANT_WREV(target, ELIBACC);
#endif

//#define	ELIBBAD		80	/* Accessing a corrupted shared library */
#ifdef ELIBBAD
_ERRCMN_DEFINE_CONSTANT_WREV(target, ELIBBAD);
#endif

//#define	ELIBSCN		81	/* .lib section in a.out corrupted */
#ifdef ELIBSCN
_ERRCMN_DEFINE_CONSTANT_WREV(target, ELIBSCN);
#endif

//#define	ELIBMAX		82	/* Attempting to link in too many shared libraries */
#ifdef ELIBMAX
_ERRCMN_DEFINE_CONSTANT_WREV(target, ELIBMAX);
#endif

//#define	ELIBEXEC	83	/* Cannot exec a shared library directly */
#ifdef ELIBEXEC
_ERRCMN_DEFINE_CONSTANT_WREV(target, ELIBEXEC);
#endif

//#define	EILSEQ		84	/* Illegal byte sequence */
#ifdef EILSEQ
_ERRCMN_DEFINE_CONSTANT_WREV(target, EILSEQ);
#endif

//#define	ERESTART	85	/* Interrupted system call should be restarted */
#ifdef ERESTART
_ERRCMN_DEFINE_CONSTANT_WREV(target, ERESTART);
#endif

//#define	ESTRPIPE	86	/* Streams pipe error */
#ifdef ESTRPIPE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESTRPIPE);
#endif

//#define	EUSERS		87	/* Too many users */
#ifdef EUSERS
_ERRCMN_DEFINE_CONSTANT_WREV(target, EUSERS);
#endif

//#define	ENOTSOCK	88	/* Socket operation on non-socket */
#ifdef ENOTSOCK
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTSOCK);
#endif

//#define	EDESTADDRREQ	89	/* Destination address required */
#ifdef EDESTADDRREQ
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDESTADDRREQ);
#endif

//#define	EMSGSIZE	90	/* Message too long */
#ifdef EMSGSIZE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMSGSIZE);
#endif

//#define	EPROTOTYPE	91	/* Protocol wrong type for socket */
#ifdef EPROTOTYPE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPROTOTYPE);
#endif

//#define	ENOPROTOOPT	92	/* Protocol not available */
#ifdef ENOPROTOOPT
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOPROTOOPT);
#endif

//#define	EPROTONOSUPPORT	93	/* Protocol not supported */
#ifdef EPROTONOSUPPORT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPROTONOSUPPORT);
#endif

//#define	ESOCKTNOSUPPORT	94	/* Socket type not supported */
#ifdef ESOCKTNOSUPPORT
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESOCKTNOSUPPORT);
#endif

//#define	EOPNOTSUPP	95	/* Operation not supported on transport endpoint */
#ifdef EOPNOTSUPP
_ERRCMN_DEFINE_CONSTANT_WREV(target, EOPNOTSUPP);
#endif

//#define	EPFNOSUPPORT	96	/* Protocol family not supported */
#ifdef EPFNOSUPPORT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EPFNOSUPPORT);
#endif

//#define	EAFNOSUPPORT	97	/* Address family not supported by protocol */
#ifdef EAFNOSUPPORT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EAFNOSUPPORT);
#endif

//#define	EADDRINUSE	98	/* Address already in use */
#ifdef EADDRINUSE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EADDRINUSE);
#endif

//#define	EADDRNOTAVAIL	99	/* Cannot assign requested address */
#ifdef EADDRNOTAVAIL
_ERRCMN_DEFINE_CONSTANT_WREV(target, EADDRNOTAVAIL);
#endif

//#define	ENETDOWN	100	/* Network is down */
#ifdef ENETDOWN
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENETDOWN);
#endif

//#define	ENETUNREACH	101	/* Network is unreachable */
#ifdef ENETUNREACH
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENETUNREACH);
#endif

//#define	ENETRESET	102	/* Network dropped connection because of reset */
#ifdef ENETRESET
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENETRESET);
#endif

//#define	ECONNABORTED	103	/* Software caused connection abort */
#ifdef ECONNABORTED
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECONNABORTED);
#endif

//#define	ECONNRESET	104	/* Connection reset by peer */
#ifdef ECONNRESET
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECONNRESET);
#endif





//#define	EISCONN		106	/* Transport endpoint is already connected */
#ifdef EISCONN
_ERRCMN_DEFINE_CONSTANT_WREV(target, EISCONN);
#endif

//#define	ENOTCONN	107	/* Transport endpoint is not connected */
#ifdef ENOTCONN
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTCONN);
#endif

//#define	ESHUTDOWN	108	/* Cannot send after transport endpoint shutdown */
#ifdef ESHUTDOWN
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESHUTDOWN);
#endif

//#define	ETOOMANYREFS	109	/* Too many references: cannot splice */
#ifdef ETOOMANYREFS
_ERRCMN_DEFINE_CONSTANT_WREV(target, ETOOMANYREFS);
#endif

//#define	ETIMEDOUT	110	/* Connection timed out */
#ifdef ETIMEDOUT
_ERRCMN_DEFINE_CONSTANT_WREV(target, ETIMEDOUT);
#endif

//#define	ECONNREFUSED	111	/* Connection refused */
#ifdef ECONNREFUSED
_ERRCMN_DEFINE_CONSTANT_WREV(target, ECONNREFUSED);
#endif

//#define	EHOSTDOWN	112	/* Host is down */
#ifdef EHOSTDOWN
_ERRCMN_DEFINE_CONSTANT_WREV(target, EHOSTDOWN);
#endif

//#define	EHOSTUNREACH	113	/* No route to host */
#ifdef EHOSTUNREACH
_ERRCMN_DEFINE_CONSTANT_WREV(target, EHOSTUNREACH);
#endif

//#define	EALREADY	114	/* Operation already in progress */
#ifdef EALREADY
_ERRCMN_DEFINE_CONSTANT_WREV(target, EALREADY);
#endif

//#define	EINPROGRESS	115	/* Operation now in progress */
#ifdef EINPROGRESS
_ERRCMN_DEFINE_CONSTANT_WREV(target, EINPROGRESS);
#endif

//#define	ESTALE		116	/* Stale NFS file handle */
#ifdef ESTALE
_ERRCMN_DEFINE_CONSTANT_WREV(target, ESTALE);
#endif

//#define	EUCLEAN		117	/* Structure needs cleaning */
#ifdef EUCLEAN
_ERRCMN_DEFINE_CONSTANT_WREV(target, EUCLEAN);
#endif

//#define	ENOTNAM		118	/* Not a XENIX named type file */
#ifdef ENOTNAM
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOTNAM);
#endif

//#define	ENAVAIL		119	/* No XENIX semaphores available */
#ifdef ENAVAIL
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENAVAIL);
#endif

//#define	EISNAM		120	/* Is a named type file */
#ifdef EISNAM
_ERRCMN_DEFINE_CONSTANT_WREV(target, EISNAM);
#endif

//#define	EREMOTEIO	121	/* Remote I/O error */
#ifdef EREMOTEIO
_ERRCMN_DEFINE_CONSTANT_WREV(target, EREMOTEIO);
#endif

//#define	EDQUOT		122	/* Quota exceeded */
#ifdef EDQUOT
_ERRCMN_DEFINE_CONSTANT_WREV(target, EDQUOT);
#endif

//


//#define	ENOMEDIUM	123	/* No medium found */
#ifdef ENOMEDIUM
_ERRCMN_DEFINE_CONSTANT_WREV(target, ENOMEDIUM);
#endif

//#define	EMEDIUMTYPE	124	/* Wrong medium type */
#ifdef EMEDIUMTYPE
_ERRCMN_DEFINE_CONSTANT_WREV(target, EMEDIUMTYPE);
#endif

//--------------------------------------------------------

}

#endif

typedef struct {
	const char *label;
	const int code;
} custom_errno;



custom_errno custom_errs[] = {
//		{ "Port is unresponsive.", SIXLBR_PORT_UNRESPONSIVE }
		{ "Unknown TTY.", GREASE_UNKNOWN_TTY },
		{ "Unknown failure.", GREASE_UNKNOWN_FAILURE },
		{ "Call needs path.", GREASE_UNKNOWN_NO_PATH }
};


	char *get_custom_err_str(int _errno) {
		int n = sizeof(custom_errs);
		while(n > 0) {
			if(custom_errs[n-1].code == _errno) {
				return (char *) custom_errs[n-1].label;
			}
			n--;
		}
		return NULL;
	}


	const int max_error_buf = 255;

	char *_errcmn::get_error_str(int _errno) {
		char *ret = (char *) malloc(max_error_buf);
		int r = ERR_STRERROR_R(_errno,ret,max_error_buf);
		if ( r != 0 ) DBG_OUT("strerror_r bad return: %d\n",r);
		return ret;
	}


	void _errcmn::free_error_str(char *b) {
		free(b);
	}

	void _errcmn::err_ev::setError(int e,const char *m)
	{
		_errno = e;
		if(errstr) free(errstr);
		if(!m) {
			if(_errno < _ERRCMN_CUSTOM_ERROR_CUTOFF) {
				errstr = get_error_str(_errno);
			} else {
				char *custom = get_custom_err_str(_errno);
				if(custom)
					errstr = ::local_strdup_safe(custom);
				else
				    errstr = ::local_strdup_safe("Unknown Error.");
			}
		} else
			errstr = ::local_strdup_safe(m);
	}


#ifndef GREASE_LIB
	v8::Local<v8::Value> errno_to_JS(int _errno, const char *prefix) {
		v8::Local<v8::Object> retobj = Nan::New<v8::Object>();

		if(_errno) {
			char *temp = NULL;
			if(_errno < _ERRCMN_CUSTOM_ERROR_CUTOFF) {
				char *errstr = get_error_str(_errno);
				if(errstr) {
					if(prefix) {
						int len = strlen(prefix)+strlen(errstr)+2;
						temp = (char *) malloc(len);
						memset(temp,0,len);
						strcpy(temp, prefix);
						strcat(temp, errstr);
					} else {
						temp = errstr;
					}
					Nan::Set(retobj,Nan::New("message").ToLocalChecked(), Nan::New(temp).ToLocalChecked());
					free_error_str(errstr);
					if(prefix) {
						free(temp);
					}
				}
			}
			Nan::Set(retobj,Nan::New("errno").ToLocalChecked(), Nan::New((uint32_t)_errno));
		}
		return retobj;
	}

	v8::Local<v8::Value> errno_to_JSError(int _errno, const char *prefix) {
		v8::Local<v8::Object> retobj;
		bool custom = false;
		if(_errno) {
			char *temp = NULL;
			char *errstr = NULL;
			if(_errno < _ERRCMN_CUSTOM_ERROR_CUTOFF) {
				errstr = get_error_str(_errno);
			} else {
				errstr = get_custom_err_str(_errno);
				custom = true;
			}
			if(errstr) {
					if(prefix) {
						int len = strlen(prefix)+strlen(errstr)+2;
						temp = (char *) malloc(len);
						memset(temp,0,len);
						strcpy(temp, prefix);
						strcat(temp, errstr);
					} else {
						temp = errstr;
					}
					retobj = Nan::Error(Nan::New(temp).ToLocalChecked())->ToObject();

					if(!custom) {
						free_error_str(errstr);
					}
					if(prefix) {
						free(temp);
					}

			} else
				retobj = Nan::Error(Nan::New("Error").ToLocalChecked())->ToObject();
//				retobj = v8::Exception::Error(v8::String::New("Error"));

			Nan::Set(retobj->ToObject(),Nan::New("errno").ToLocalChecked(), Nan::New((uint32_t)_errno));
		}
		return retobj;
	}

	v8::Handle<v8::Value> err_ev_to_JS(err_ev &e, const char *prefix) {
		Nan::EscapableHandleScope scope;
	//		v8::Local<v8::Object> retobj = v8::Object::New();
		v8::Local<v8::Value> retobj = Nan::Undefined();

		if(e.hasErr()) {
			char *temp = NULL;
			if(prefix && e.errstr) {
				int len = strlen(prefix)+strlen(e.errstr)+2;
				temp = (char *) malloc(len);
				memset(temp,0,len);
				strcpy(temp, prefix);
				strcat(temp, e.errstr);
	//				retobj->Set(v8::String::New("message"), v8::String::New(temp));
				retobj = Nan::Error(temp);
				free(temp);
			}
			else retobj = Nan::Error("Error");
			Nan::Set(retobj->ToObject(),Nan::New("errno").ToLocalChecked(), Nan::New((uint32_t)e._errno));
		}
		return scope.Escape(retobj);
	}
#endif


