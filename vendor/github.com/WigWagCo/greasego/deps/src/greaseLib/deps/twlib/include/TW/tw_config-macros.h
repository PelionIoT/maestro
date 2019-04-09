// -*- C++ -*-

//==========================================================================
/**
 *  @file   config-macros.h
 *
 *  $Id: config-macros.h 92474 2010-11-02 13:29:39Z johnnyw $
 *
 *  @author (Originally in OS.h)Doug Schmidt <schmidt@cs.wustl.edu>
 *  @author Jesper S. M|ller<stophph@diku.dk>
 *  @author and a cast of thousands...
 *
 *  This file contains the contents of the old config-lite.h header
 *  without C++ code (except for C++ code in macros).  Specifically,
 *  only macros or C language constructs are found in this header.
 *  Allows configuration values and macros to be used by some C
 *  language sources.
 */
//==========================================================================

#ifndef TW_CONFIG_MACROS_H
#define TW_CONFIG_MACROS_H

//#include "ace/config.h"

//#include "ace/Version.h"
//#include "ace/Versioned_Namespace.h"

#if !defined (TW_HAS_EXCEPTIONS)
#define TW_HAS_EXCEPTIONS
#endif /* !ACE_HAS_EXCEPTIONS */

// TW_HAS_TLI is used to decide whether to try any XTI/TLI functionality
// so if it isn't set, set it. Capabilities and differences between
// XTI and TLI favor XTI, but when deciding to do anything, as opposed to
// TW_NOTSUP_RETURN for example, TW_HAS_TLI is the deciding factor.
#if !defined (TW_HAS_TLI)
#  if defined (TW_HAS_XTI)
#    define TW_HAS_TLI
#  endif /* TW_HAS_XTI */
#endif /* TW_HAS_TLI */

#define TW_BITS_PER_ULONG (8 * sizeof (u_long))

#if !defined (TW_OSTREAM_TYPE)
# if defined (TW_LACKS_IOSTREAM_TOTALLY)
#   define TW_OSTREAM_TYPE FILE
# else  /* ! TW_LACKS_IOSTREAM_TOTALLY */
#   define TW_OSTREAM_TYPE ostream
# endif /* ! TW_LACKS_IOSTREAM_TOTALLY */
#endif /* ! TW_OSTREAM_TYPE */

#if !defined (TW_DEFAULT_LOG_STREAM)
# if defined (TW_LACKS_IOSTREAM_TOTALLY)
#   define TW_DEFAULT_LOG_STREAM 0
# else  /* ! TW_LACKS_IOSTREAM_TOTALLY */
#   define TW_DEFAULT_LOG_STREAM (&cerr)
# endif /* ! TW_LACKS_IOSTREAM_TOTALLY */
#endif /* ! TW_DEFAULT_LOG_STREAM */

// For Win32 compatibility...
# if !defined (TW_WSOCK_VERSION)
#   define TW_WSOCK_VERSION 0, 0
# endif /* TW_WSOCK_VERSION */

# if defined (TW_MT_SAFE) && (TW_MT_SAFE != 0)
#   define TW_MT(X) X
#   if !defined (_REENTRANT)
#     define _REENTRANT
#   endif /* _REENTRANT */
# else
#   define TW_MT(X)
# endif /* TW_MT_SAFE */

# if defined (TW_HAS_PURIFY)
#   define TW_INITIALIZE_MEMORY_BEFORE_USE
# endif /* TW_HAS_PURIFY */

# if defined (TW_HAS_VALGRIND)
#   define TW_INITIALIZE_MEMORY_BEFORE_USE
#   define TW_LACKS_DLCLOSE
# endif /* TW_HAS_VALGRIND */

// =========================================================================
// Perfect Multicast filting refers to RFC 3376, where a socket is only
// delivered dgrams for groups joined even if it didn't bind the group
// address.  We turn this option off by default, although most OS's
// except for Windows and Solaris probably lack perfect filtering.
// =========================================================================

# if !defined (TW_LACKS_PERFECT_MULTICAST_FILTERING)
#   define TW_LACKS_PERFECT_MULTICAST_FILTERING 0
# endif /* TW_LACKS_PERFECT_MULTICAST_FILTERING */

// =========================================================================
// Enable/Disable Features By Default
// =========================================================================

# if !defined (TW_HAS_POSITION_INDEPENDENT_POINTERS)
#   define TW_HAS_POSITION_INDEPENDENT_POINTERS 1
# endif /* TW_HAS_POSITION_INDEPENDENT_POINTERS */

# if !defined (TW_HAS_PROCESS_SPAWN)
#   if !defined (TW_LACKS_FORK) || \
       (defined (TW_WIN32) && !defined (TW_HAS_PHARLAP)) || \
       defined (TW_WINCE) || defined (TW_OPENVMS)
#     define TW_HAS_PROCESS_SPAWN 1
#   endif
# endif /* TW_HAS_PROCESS_SPAWN */

# if !defined (TW_HAS_DYNAMIC_LINKING)
#   if defined (TW_HAS_SVR4_DYNAMIC_LINKING) || defined (TW_WIN32) || defined (TW_VXWORKS) || defined (__hpux)
#     define TW_HAS_DYNAMIC_LINKING 1
#   endif
# endif /* TW_HAS_DYNAMIC_LINKING */

# if defined (TW_USES_FIFO_SEM)
#   if defined (TW_HAS_POSIX_SEM) || defined (TW_LACKS_MKFIFO) || defined (TW_LACKS_FCNTL)
#     undef TW_USES_FIFO_SEM
#   endif
# endif /* TW_USES_FIFO_SEM */

// =========================================================================
// INLINE macros
//
// These macros handle all the inlining of code via the .i or .inl files
// =========================================================================

#if defined (TW_LACKS_INLINE_FUNCTIONS) && !defined (TW_NO_INLINE)
#  define TW_NO_INLINE
#endif /* defined (TW_LACKS_INLINE_FUNCTIONS) && !defined (TW_NO_INLINE) */

// ACE inlining has been explicitly disabled.  Implement
// internally within ACE by undefining __TW_INLINE__.
#if defined (TW_NO_INLINE)
#  undef __TW_INLINE__
#endif /* ! TW_NO_INLINE */

#if defined (__TW_INLINE__)
#  define TW_INLINE inline
#  if !defined (TW_HAS_INLINED_OSCALLS)
#    define TW_HAS_INLINED_OSCALLS
#  endif /* !ACE_HAS_INLINED_OSCALLS */
#else
#  define TW_INLINE
#endif /* __TW_INLINE__ */

// ============================================================================
// EXPORT macros
//
// Since Win32 DLL's do not export all symbols by default, they must be
// explicitly exported (which is done by *_Export macros).
// ============================================================================

// Win32 should have already defined the macros in config-win32-common.h
#if !defined (TW_HAS_CUSTOM_EXPORT_MACROS)
#  define TW_Proper_Export_Flag
#  define TW_Proper_Import_Flag
#  define TW_EXPORT_SINGLETON_DECLARATION(T)
#  define TW_IMPORT_SINGLETON_DECLARATION(T)
#  define TW_EXPORT_SINGLETON_DECLARE(SINGLETON_TYPE, CLASS, LOCK)
#  define TW_IMPORT_SINGLETON_DECLARE(SINGLETON_TYPE, CLASS, LOCK)
#else
// An export macro should at the very least have been defined.

#  ifndef TW_Proper_Import_Flag
#    define TW_Proper_Import_Flag
#  endif  /* !ACE_Proper_Import_Flag */

#  ifndef TW_EXPORT_SINGLETON_DECLARATION
#    define TW_EXPORT_SINGLETON_DECLARATION(T)
#  endif  /* !ACE_EXPORT_SINGLETON_DECLARATION */

#  ifndef TW_IMPORT_SINGLETON_DECLARATION
#    define TW_IMPORT_SINGLETON_DECLARATION(T)
#  endif  /* !ACE_IMPORT_SINGLETON_DECLARATION */

#  ifndef TW_EXPORT_SINGLETON_DECLARE
#    define TW_EXPORT_SINGLETON_DECLARE(SINGLETON_TYPE, CLASS, LOCK)
#  endif  /* !ACE_EXPORT_SINGLETON_DECLARE */

#  ifndef TW_IMPORT_SINGLETON_DECLARE
#    define TW_IMPORT_SINGLETON_DECLARE(SINGLETON_TYPE, CLASS, LOCK)
#  endif  /* !ACE_IMPORT_SINGLETON_DECLARE */

#endif /* !ACE_HAS_CUSTOM_EXPORT_MACROS */

// This is a whim of mine -- that instead of annotating a class with
// TW_Export in its declaration, we make the declaration near the TOP
// of the file with TW_DECLARE_EXPORT.
// TS = type specifier (e.g., class, struct, int, etc.)
// ID = identifier
// So, how do you use it?  Most of the time, just use ...
// TW_DECLARE_EXPORT(class, someobject);
// If there are global functions to be exported, then use ...
// TW_DECLARE_EXPORT(void, globalfunction) (int, ...);
// Someday, when template libraries are supported, we made need ...
// TW_DECLARE_EXPORT(template class, sometemplate) <class TYPE, class LOCK>;
# define TW_DECLARE_EXPORT(TS,ID) TS TW_Export ID

// ============================================================================
// Cast macros
//
// These macros are used to choose between the old cast style and the new
// *_cast<> operators
// ============================================================================

#   define TW_sap_any_cast(TYPE)                                      reinterpret_cast<TYPE> (const_cast<TW_Addr &> (TW_Addr::sap_any))

# if !defined (TW_CAST_CONST)
    // Sun CC 4.2, for example, requires const in reinterpret casts of
    // data members in const member functions.  But, other compilers
    // complain about the useless const.  This keeps everyone happy.
#   if defined (__SUNPRO_CC)
#     define TW_CAST_CONST const
#   else  /* ! __SUNPRO_CC */
#     define TW_CAST_CONST
#   endif /* ! __SUNPRO_CC */
# endif /* ! TW_CAST_CONST */

// ============================================================================
// Compiler Silencing macros
//
// Some compilers complain about parameters that are not used.  This macro
// should keep them quiet.
// ============================================================================

#if !defined (TW_UNUSED_ARG)
# if defined (__GNUC__) && ((__GNUC__ > 4 || (__GNUC__ == 4 && __GNUC_MINOR__ >= 2)))
#   define TW_UNUSED_ARG(a) (void) (a)
# elif defined (__GNUC__) || defined (ghs) || defined (__hpux) || defined (__DECCXX) || defined (__rational__) || defined (__USLC__) || defined (TW_RM544) || defined (__DCC__) || defined (__PGI) || defined (__TANDEM)
// Some compilers complain about "statement with no effect" with (a).
// This eliminates the warnings, and no code is generated for the null
// conditional statement.  @note that may only be true if -O is enabled,
// such as with GreenHills (ghs) 1.8.8.
#  define TW_UNUSED_ARG(a) do {/* null */} while (&a == 0)
# elif defined (__DMC__)
   #define TW_UNUSED_ID(identifier)
   template <class T>
   inline void TW_UNUSED_ARG(const T& TW_UNUSED_ID(t)) { }
# else /* ghs || __GNUC__ || ..... */
#  define TW_UNUSED_ARG(a) (a)
# endif /* ghs || __GNUC__ || ..... */
#endif /* !ACE_UNUSED_ARG */

#if defined (_MSC_VER) || defined (ghs) || defined (__DECCXX) || defined(__BORLANDC__) || defined (TW_RM544) || defined (__USLC__) || defined (__DCC__) || defined (__PGI) || defined (__TANDEM) || (defined (__HP_aCC) && (__HP_aCC < 40000 || __HP_aCC >= 60500))
# define TW_NOTREACHED(a)
#else  /* ghs || ..... */
# define TW_NOTREACHED(a) a
#endif /* ghs || ..... */

// ============================================================================
// TW_ALLOC_HOOK* macros
//
// Macros to declare and define class-specific allocation operators.
// ============================================================================

# if defined (TW_HAS_ALLOC_HOOKS)
#   define TW_ALLOC_HOOK_DECLARE \
  void *operator new (size_t bytes); \
  void operator delete (void *ptr);

  // Note that these are just place holders for now.  Some day they
  // may be be replaced by <TW_Malloc>.
#   define TW_ALLOC_HOOK_DEFINE(CLASS) \
  void *CLASS::operator new (size_t bytes) { return ::new char[bytes]; } \
  void CLASS::operator delete (void *ptr) { delete [] ((char *) ptr); }
# else
#   define TW_ALLOC_HOOK_DECLARE struct __Ace {} /* Just need a dummy... */
#   define TW_ALLOC_HOOK_DEFINE(CLASS)
# endif /* TW_HAS_ALLOC_HOOKS */

// ============================================================================
/**
 * TW_OSCALL* macros
 *
 * @deprecated TW_OSCALL_RETURN and TW_OSCALL should not be used.
 *             Please restart system calls in your application code.
 *             See the @c sigaction(2) man page for documentation
 *             regarding enabling restartable system calls across
 *             signals via the @c SA_RESTART flag.
 *
 * The following two macros used ensure that system calls are properly
 * restarted (if necessary) when interrupts occur.  However, that
 * capability was never enabled by any of our supported platforms.
 * In fact, some parts of ACE would not function properly when that
 * ability was enabled.  Furthermore, they assumed that ability to
 * restart system calls was determined statically.  That assumption
 * does not hold for modern platforms, where that ability is
 * determined dynamically at run-time.
 */
// ============================================================================

#define TW_OSCALL_RETURN(X,TYPE,FAILVALUE) \
  do \
    return (TYPE) (X); \
  while (0)
#define TW_OSCALL(X,TYPE,FAILVALUE,RESULT) \
  do \
    RESULT = (TYPE) (X); \
  while (0)

#if defined (TW_WIN32)
# define TW_WIN32CALL_RETURN(X,TYPE,FAILVALUE) \
  do { \
    TYPE ace_result_; \
    ace_result_ = (TYPE) X; \
    if (ace_result_ == FAILVALUE) \
      TW_OS::set_errno_to_last_error (); \
    return ace_result_; \
  } while (0)
# define TW_WIN32CALL(X,TYPE,FAILVALUE,RESULT) \
  do { \
    RESULT = (TYPE) X; \
    if (RESULT == FAILVALUE) \
      TW_OS::set_errno_to_last_error (); \
  } while (0)
#endif  /* TW_WIN32 */

// The C99 security-improved run-time returns an error value on failure;
// 0 on success.
#if defined (TW_HAS_TR24731_2005_CRT)
#  define TW_SECURECRTCALL(X,TYPE,FAILVALUE,RESULT) \
  do { \
    errno_t ___ = X; \
    if (___ != 0) { errno = ___; RESULT = FAILVALUE; } \
  } while (0)
#endif /* TW_HAS_TR24731_2005_CRT */

// ============================================================================
// Fundamental types
// ============================================================================

#if defined (TW_WIN32)

typedef HANDLE TW_HANDLE;
typedef SOCKET TW_SOCKET;
# define TW_INVALID_HANDLE INVALID_HANDLE_VALUE

#else /* ! TW_WIN32 */

typedef int TW_HANDLE;
typedef TW_HANDLE TW_SOCKET;
# define TW_INVALID_HANDLE -1

#endif /* TW_WIN32 */

// Define the type that's returned from the platform's native thread
// functions. TW_THR_FUNC_RETURN is the type defined as the thread
// function's return type, except when the thread function doesn't return
// anything (pSoS). The TW_THR_FUNC_NO_RETURN_VAL macro is used to
// indicate that the actual thread function doesn't return anything. The
// rest of ACE uses a real type so there's no a ton of conditional code
// everywhere to deal with the possibility of no return type.
# if defined (TW_VXWORKS) && !defined (TW_HAS_PTHREADS)
# include /**/ <taskLib.h>
typedef int TW_THR_FUNC_RETURN;
#define TW_HAS_INTEGRAL_TYPE_THR_FUNC_RETURN
# elif defined (TW_WIN32)
typedef DWORD TW_THR_FUNC_RETURN;
#define TW_HAS_INTEGRAL_TYPE_THR_FUNC_RETURN
# else
typedef void* TW_THR_FUNC_RETURN;
# endif /* TW_VXWORKS */
typedef TW_THR_FUNC_RETURN (*TW_THR_FUNC)(void *);

#ifdef __cplusplus
extern "C" {
#endif  /* __cplusplus */
typedef void (*TW_THR_C_DEST)(void *);
#ifdef __cplusplus
}
#endif  /* __cplusplus */
typedef void (*TW_THR_DEST)(void *);

// Now some platforms have special requirements...
# if defined (TW_VXWORKS) && !defined (TW_HAS_PTHREADS)
typedef FUNCPTR TW_THR_FUNC_INTERNAL;  // where typedef int (*FUNCPTR) (...)
# else
typedef TW_THR_FUNC TW_THR_FUNC_INTERNAL;
# endif /* TW_VXWORKS */

# ifdef __cplusplus
extern "C"
{
# endif  /* __cplusplus */
# if defined (TW_VXWORKS) && !defined (TW_HAS_PTHREADS)
typedef FUNCPTR TW_THR_C_FUNC;  // where typedef int (*FUNCPTR) (...)
# else
typedef TW_THR_FUNC_RETURN (*TW_THR_C_FUNC)(void *);
# endif /* TW_VXWORKS */
# ifdef __cplusplus
}
# endif  /* __cplusplus */

// ============================================================================
// Macros for controlling the lifetimes of dlls loaded by TW_DLL--including
// all dlls loaded via the ACE Service Config framework.
//
// Please don't change these values or add new ones wantonly, since we use
// the TW_BIT_ENABLED, etc..., macros to test them.
// ============================================================================

// Per-process policy that unloads dlls eagerly.
#define TW_DLL_UNLOAD_POLICY_PER_PROCESS 0
// Apply policy on a per-dll basis.  If the dll doesn't use one of the macros
// below, the current per-process policy will be used.
#define TW_DLL_UNLOAD_POLICY_PER_DLL 1
// Don't unload dll when refcount reaches zero, i.e., wait for either an
// explicit unload request or program exit.
#define TW_DLL_UNLOAD_POLICY_LAZY 2
// Default policy allows dlls to control their own destinies, but will
// unload those that don't make a choice eagerly.
#define TW_DLL_UNLOAD_POLICY_DEFAULT TW_DLL_UNLOAD_POLICY_PER_DLL

// Add this macro you one of your cpp file in your dll.  X should
// be either TW_DLL_UNLOAD_POLICY_DEFAULT or TW_DLL_UNLOAD_POLICY_LAZY.
#define TW_DLL_UNLOAD_POLICY(CLS,X) \
extern "C" u_long CLS##_Export _get_dll_unload_policy (void) \
  { return X;}

// ============================================================================
// TW_USES_CLASSIC_SVC_CONF macro
// ============================================================================

// For now, default is to use the classic svc.conf format.
#if !defined (TW_USES_CLASSIC_SVC_CONF)
# if defined (TW_HAS_CLASSIC_SVC_CONF) && defined (TW_HAS_XML_SVC_CONF)
#   error You can only use either CLASSIC or XML svc.conf, not both.
# endif
// Change the TW_HAS_XML_SVC_CONF to TW_HAS_CLASSIC_SVC_CONF when
// we switch ACE to use XML svc.conf as default format.
# if defined (TW_HAS_XML_SVC_CONF)
#   define TW_USES_CLASSIC_SVC_CONF 0
# else
#   define TW_USES_CLASSIC_SVC_CONF 1
# endif /* TW_HAS_XML_SVC_CONF */
#endif /* TW_USES_CLASSIC_SVC_CONF */

// ============================================================================
// Default svc.conf file extension.
// ============================================================================
#if defined (TW_USES_CLASSIC_SVC_CONF) && (TW_USES_CLASSIC_SVC_CONF == 1)
# define TW_DEFAULT_SVC_CONF_EXT   ".conf"
#else
# define TW_DEFAULT_SVC_CONF_EXT   ".conf.xml"
#endif /* TW_USES_CLASSIC_SVC_CONF && TW_USES_CLASSIC_SVC_CONF == 1 */

// ============================================================================
// Miscellaneous macros
// ============================================================================

#if defined (TW_USES_EXPLICIT_STD_NAMESPACE)
#  define TW_STD_NAMESPACE std
#else
#  define TW_STD_NAMESPACE
#endif

#if !defined (TW_OS_String)
#  define TW_OS_String TW_OS
#endif /* TW_OS_String */
#if !defined (TW_OS_Memory)
#  define TW_OS_Memory TW_OS
#endif /* TW_OS_Memory */
#if !defined (TW_OS_Dirent)
#  define TW_OS_Dirent TW_OS
#endif /* TW_OS_Dirent */
#if !defined (TW_OS_TLI)
#  define TW_OS_TLI TW_OS
#endif /* TW_OS_TLI */

// -------------------------------------------------------------------
// Preprocessor symbols will not be expanded if they are
// concatenated.  Force the preprocessor to expand them during the
// argument prescan by calling a macro that itself calls another that
// performs the actual concatenation.
#define TW_PREPROC_CONCATENATE_IMPL(A,B) A ## B
#define TW_PREPROC_CONCATENATE(A,B) TW_PREPROC_CONCATENATE_IMPL(A,B)
// -------------------------------------------------------------------

/// If MPC is using a lib modifier this define will be set and this then
/// is used by the service configurator framework
#if defined MPC_LIB_MODIFIER && !defined (TW_LD_DECORATOR_STR)
#define TW_LD_DECORATOR_STR TW_TEXT( MPC_LIB_MODIFIER )
#endif /* MPC_LIB_MODIFIER */

#ifndef TW_GCC_CONSTRUCTOR_ATTRIBUTE
# define TW_GCC_CONSTRUCTOR_ATTRIBUTE
#endif

#ifndef TW_GCC_DESTRUCTOR_ATTRIBUTE
# define TW_GCC_DESTRUCTOR_ATTRIBUTE
#endif

#ifndef TW_HAS_TEMPLATE_TYPEDEFS
#define TW_HAS_TEMPLATE_TYPEDEFS
#endif

#ifndef TW_DEPRECATED
# define TW_DEPRECATED
#endif

#endif /* TW_CONFIG_MACROS_H */
