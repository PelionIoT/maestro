// WigWag LLC
// (c) 2010
// tw_macros.h
// Author: ed
// Sep 30, 2010

/*
 * These are global macros - mostly for memory allocation
 *
 * Many of these macros were lifted entirely from the ACE project.
 * http://www.cs.wustl.edu/~schmidt/ACE-copying.html
 * SEE licenses.txt for more information.
 */

#ifndef TW_MACROS_H_
#define TW_MACROS_H_






#if !defined (TW_MALLOC_FUNC)
#  define TW_MALLOC_FUNC ::malloc
#endif
#if !defined (TW_CALLOC_FUNC)
#  define TW_CALLOC_FUNC ::calloc
#endif
#if !defined (TW_FREE_FUNC)
#  define TW_FREE_FUNC ::free
#endif
#if !defined (TW_REALLOC_FUNC)
#  define TW_REALLOC_FUNC ::realloc
#endif

//
//#if defined (TW_HAS_OLD_MALLOC)
//typedef char * TW_MALLOC_T;
//#else
//typedef void * TW_MALLOC_T;
//#endif /* TW_HAS_OLD_MALLOC */


// ============================================================================
// TW_NEW macros
//
// A useful abstraction for expressions involving operator new since
// we can change memory allocation error handling policies (e.g.,
// depending on whether ANSI/ISO exception handling semantics are
// being used).
// ============================================================================

// If new(std::nothrow) is defined then, by definition, new throws exceptions.
#if defined (TW_HAS_NEW_NOTHROW)
#  if !defined (TW_NEW_THROWS_EXCEPTIONS)
#    define TW_NEW_THROWS_EXCEPTIONS
#  endif
#endif

// The Windows MFC exception mechanism requires that a caught CException
// (including the CMemoryException in use here) be freed using its Delete()
// method. Thus, when MFC is in use and we're catching exceptions as a result
// of new(), the exception's Delete() method has to be called. No other
// platform imposes this sort of restriction/requirement. The Windows
// config stuff (at least for MSVC/MFC) defines a TW_del_bad_alloc macro
// that works with its TW_bad_alloc macro to implement this cleanup
// requirement. Since no other platform requires this, define it as
// empty here.
#if !defined (TW_del_bad_alloc)
#  define TW_del_bad_alloc
#endif

#if defined (TW_NEW_THROWS_EXCEPTIONS)

// Since new() throws exceptions, we need a way to avoid passing
// exceptions past the call to new because TW counts on having a 0
// return value for a failed allocation. Some compilers offer the
// new (nothrow) version, which does exactly what we want. Others
// do not. For those that do not, this sets up what exception is thrown,
// and then below we'll do a try/catch around the new to catch it and
// return a 0 pointer instead.

// ED-CHANGE: removing this section...
#ifdef _NOT_DEF_

#  if defined (__HP_aCC)
      // I know this works for HP aC++... if <stdexcept> is used, it
      // introduces other stuff that breaks things, like <memory>, which
      // screws up auto_ptr.
#    include /**/ <new>
    // _HP_aCC was first defined at aC++ 03.13 on HP-UX 11. Prior to that
    // (03.10 and before) a failed new threw bad_alloc. After that (03.13
    // and above) the exception thrown is dependent on the below settings.
#    if (HPUX_VERS >= 1100)
#      if ((__HP_aCC < 32500 && !defined (RWSTD_NO_NAMESPACE)) || \
           defined (TW_USES_STD_NAMESPACE_FOR_STDCPP_LIB))
#        define TW_bad_alloc ::std::bad_alloc
#        define TW_nothrow   ::std::nothrow
#        define TW_nothrow_t ::std::nothrow_t
#      else
#        define TW_bad_alloc bad_alloc
#        define TW_nothrow   nothrow
#        define TW_nothrow_t nothrow_t
#      endif /* __HP_aCC */
#    elif ((__HP_aCC <  12500 && !defined (RWSTD_NO_NAMESPACE)) || \
           defined (TW_USES_STD_NAMESPACE_FOR_STDCPP_LIB))
#      define TW_bad_alloc ::std::bad_alloc
#      define TW_nothrow   ::std::nothrow
#      define TW_nothrow_t ::std::nothrow_t
#    else
#      define TW_bad_alloc bad_alloc
#      define TW_nothrow   nothrow
#      define TW_nothrow_t nothrow_t
#    endif /* HPUX_VERS < 1100 */
#    define TW_throw_bad_alloc throw TW_bad_alloc ()
#  elif defined (__SUNPRO_CC)
#      if (__SUNPRO_CC < 0x500) || (__SUNPRO_CC_COMPAT == 4)
#        include /**/ <exception.h>
         // Note: we catch ::xalloc rather than just xalloc because of
         // a name clash with unsafe_ios::xalloc()
#        define TW_bad_alloc ::xalloc
#        define TW_throw_bad_alloc throw TW_bad_alloc ("no more memory")
#      else
#        include /**/ <new>
#        define TW_bad_alloc ::std::bad_alloc
#        if defined (TW_HAS_NEW_NOTHROW)
#          if defined (TW_USES_STD_NAMESPACE_FOR_STDCPP_LIB)
#            define TW_nothrow   ::std::nothrow
#            define TW_nothrow_t ::std::nothrow_t
#          else
#            define TW_nothrow   nothrow
#            define TW_nothrow_t nothrow_t
#          endif /* TW_USES_STD_NAMESPACE_FOR_STDCPP_LIB */
#        endif /* TW_HAS_NEW_NOTHROW */
#        define TW_throw_bad_alloc throw TW_bad_alloc ()
#      endif /* __SUNPRO_CC < 0x500 */
#  elif defined (TW_USES_STD_NAMESPACE_FOR_STDCPP_LIB)
#    include /**/ <new>
#    if !defined (TW_bad_alloc)
#      define TW_bad_alloc ::std::bad_alloc
#    endif
#    define TW_nothrow   ::std::nothrow
#    define TW_nothrow_t ::std::nothrow_t
     // MFC changes the behavior of operator new at all MSVC versions from 6 up.
#    if defined (TW_HAS_MFC) && (TW_HAS_MFC == 1)
#      define TW_throw_bad_alloc AfxThrowMemoryException ()
#    else
#      define TW_throw_bad_alloc throw TW_bad_alloc ()
#    endif
#  else
#    include /**/ <new>
#    if !defined (TW_bad_alloc)
#      define TW_bad_alloc bad_alloc
#    endif
#    define TW_nothrow   nothrow
#    define TW_nothrow_t nothrow_t
     // MFC changes the behavior of operator new at all MSVC versions from 6 up.
#    if defined (TW_HAS_MFC) && (TW_HAS_MFC == 1)
#      define TW_throw_bad_alloc AfxThrowMemoryException ()
#    else
#      define TW_throw_bad_alloc throw TW_bad_alloc ()
#    endif
#  endif /* __HP_aCC */


#endif // _NOT_DEF_

// ED-CHANGE: just define these here for simplicity
#include /**/ <new>
#define TW_nothrow nothrow
#define TW_bad_alloc bad_alloc
#define TW_throw_bad_alloc throw TW_bad_alloc ()


#  if defined (TW_HAS_NEW_NOTHROW)
#    define TW_NEW_RETURN(POINTER,CONSTRUCTOR,RET_VAL) \
   do { POINTER = new (TW_nothrow) CONSTRUCTOR; \
     if (POINTER == 0) { errno = ENOMEM; return RET_VAL; } \
   } while (0)
#    define TW_NEW(POINTER,CONSTRUCTOR) \
   do { POINTER = new(TW_nothrow) CONSTRUCTOR; \
     if (POINTER == 0) { errno = ENOMEM; return; } \
   } while (0)
#    define TW_NEW_NORETURN(POINTER,CONSTRUCTOR) \
   do { POINTER = new(TW_nothrow) CONSTRUCTOR; \
     if (POINTER == 0) { errno = ENOMEM; } \
   } while (0)

#  else

#    define TW_NEW_RETURN(POINTER,CONSTRUCTOR,RET_VAL) \
   do { try { POINTER = new CONSTRUCTOR; } \
     catch (TW_bad_alloc) { TW_del_bad_alloc errno = ENOMEM; POINTER = 0; return RET_VAL; } \
   } while (0)

#    define TW_NEW(POINTER,CONSTRUCTOR) \
   do { try { POINTER = new CONSTRUCTOR; } \
     catch (TW_bad_alloc) { TW_del_bad_alloc errno = ENOMEM; POINTER = 0; return; } \
   } while (0)

#    define TW_NEW_NORETURN(POINTER,CONSTRUCTOR) \
   do { try { POINTER = new CONSTRUCTOR; } \
     catch (TW_bad_alloc) { TW_del_bad_alloc errno = ENOMEM; POINTER = 0; } \
   } while (0)
#  endif /* TW_HAS_NEW_NOTHROW */

#else /* TW_NEW_THROWS_EXCEPTIONS */

# define TW_NEW_RETURN(POINTER,CONSTRUCTOR,RET_VAL) \
   do { POINTER = new CONSTRUCTOR; \
     if (POINTER == 0) { errno = ENOMEM; return RET_VAL; } \
   } while (0)
# define TW_NEW(POINTER,CONSTRUCTOR) \
   do { POINTER = new CONSTRUCTOR; \
     if (POINTER == 0) { errno = ENOMEM; return; } \
   } while (0)
# define TW_NEW_NORETURN(POINTER,CONSTRUCTOR) \
   do { POINTER = new CONSTRUCTOR; \
     if (POINTER == 0) { errno = ENOMEM; } \
   } while (0)

# if !defined (TW_bad_alloc)
    class TW_bad_alloc_class {};
#   define TW_bad_alloc  TW_bad_alloc_class
# endif
# if defined (TW_HAS_MFC) && (TW_HAS_MFC == 1)
#   define TW_throw_bad_alloc  AfxThrowMemoryException ()
# else
#   define TW_throw_bad_alloc  throw TW_bad_alloc ()
# endif

#endif /* TW_NEW_THROWS_EXCEPTIONS */













# define TW_ALLOCATOR_RETURN(POINTER,ALLOCATOR,RET_VAL) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; return RET_VAL; } \
   } while (0)
# define TW_ALLOCATOR(POINTER,ALLOCATOR) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; return; } \
   } while (0)
# define TW_ALLOCATOR_NORETURN(POINTER,ALLOCATOR) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; } \
   } while (0)


// this is called 'placement new'
// http://www.glenmccl.com/nd_cmp.htm
//
// the idea is you can allocate without new, but still call the destructor or constructor.

# define TW_NEW_MALLOC_RETURN(POINTER,ALLOCATOR,CONSTRUCTOR,RET_VAL) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; return RET_VAL;} \
     else { (void) new (POINTER) CONSTRUCTOR; } \
   } while (0)
# define TW_NEW_MALLOC(POINTER,ALLOCATOR,CONSTRUCTOR) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; return;} \
     else { (void) new (POINTER) CONSTRUCTOR; } \
   } while (0)
# define TW_NEW_MALLOC_NORETURN(POINTER,ALLOCATOR,CONSTRUCTOR) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM;} \
     else { (void) new (POINTER) CONSTRUCTOR; } \
   } while (0)

/* TW_Metrics */
#if defined TW_LACKS_ARRAY_PLACEMENT_NEW
# define TW_NEW_MALLOC_ARRAY_RETURN(POINTER,ALLOCATOR,CONSTRUCTOR,COUNT,RET_VAL) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; return RET_VAL;} \
     else { for (u_int i = 0; i < COUNT; ++i) \
              {(void) new (POINTER) CONSTRUCTOR; ++POINTER;} \
            POINTER -= COUNT;} \
   } while (0)
# define TW_NEW_MALLOC_ARRAY(POINTER,ALLOCATOR,CONSTRUCTOR,COUNT) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; return;} \
     else { for (u_int i = 0; i < COUNT; ++i) \
              {(void) new (POINTER) CONSTRUCTOR; ++POINTER;} \
            POINTER -= COUNT;} \
   } while (0)
#else /* ! defined TW_LACKS_ARRAY_PLACEMENT_NEW */
# define TW_NEW_MALLOC_ARRAY_RETURN(POINTER,ALLOCATOR,CONSTRUCTOR,COUNT,RET_VAL) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; return RET_VAL;} \
     else { (void) new (POINTER) CONSTRUCTOR [COUNT]; } \
   } while (0)
# define TW_NEW_MALLOC_ARRAY(POINTER,ALLOCATOR,CONSTRUCTOR,COUNT) \
   do { POINTER = ALLOCATOR; \
     if (POINTER == 0) { TW_errno = ENOMEM; return;} \
     else { (void) new (POINTER) CONSTRUCTOR [COUNT]; } \
   } while (0)
#endif /* defined TW_LACKS_ARRAY_PLACEMENT_NEW */


/////////// Free / Destructor

# define TW_DES_NOFREE(POINTER,CLASS) \
   do { \
        if (POINTER) \
          { \
            (POINTER)->~CLASS (); \
          } \
      } \
   while (0)

# define TW_DES_ARRAY_NOFREE(POINTER,SIZE,CLASS) \
   do { \
        if (POINTER) \
          { \
            for (size_t i = 0; \
                 i < SIZE; \
                 ++i) \
            { \
              (&(POINTER)[i])->~CLASS (); \
            } \
          } \
      } \
   while (0)

# define TW_DES_FREE(POINTER,DEALLOCATOR,CLASS) \
   do { \
        if (POINTER) \
          { \
            (POINTER)->~CLASS (); \
            DEALLOCATOR (POINTER); \
          } \
      } \
   while (0)

# define TW_DES_ARRAY_FREE(POINTER,SIZE,DEALLOCATOR,CLASS) \
   do { \
        if (POINTER) \
          { \
            for (size_t i = 0; \
                 i < SIZE; \
                 ++i) \
            { \
              (&(POINTER)[i])->~CLASS (); \
            } \
            DEALLOCATOR (POINTER); \
          } \
      } \
   while (0)

# if defined (TW_HAS_WORKING_EXPLICIT_TEMPLATE_DESTRUCTOR)
#   define TW_DES_NOFREE_TEMPLATE(POINTER,T_CLASS,T_PARAMETER) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS (); \
            } \
        } \
     while (0)
#   define TW_DES_ARRAY_NOFREE_TEMPLATE(POINTER,SIZE,T_CLASS,T_PARAMETER) \
     do { \
          if (POINTER) \
            { \
              for (size_t i = 0; \
                   i < SIZE; \
                   ++i) \
              { \
                (&(POINTER)[i])->~T_CLASS (); \
              } \
            } \
        } \
     while (0)

#if defined (TW_EXPLICIT_TEMPLATE_DESTRUCTOR_TAKES_ARGS)
#   define TW_DES_FREE_TEMPLATE(POINTER,DEALLOCATOR,T_CLASS,T_PARAMETER) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS T_PARAMETER (); \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#else
#   define TW_DES_FREE_TEMPLATE(POINTER,DEALLOCATOR,T_CLASS,T_PARAMETER) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS (); \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#endif /* defined(TW_EXPLICIT_TEMPLATE_DESTRUCTOR_TAKES_ARGS) */
#   define TW_DES_ARRAY_FREE_TEMPLATE(POINTER,SIZE,DEALLOCATOR,T_CLASS,T_PARAMETER) \
     do { \
          if (POINTER) \
            { \
              for (size_t i = 0; \
                   i < SIZE; \
                   ++i) \
              { \
                (&(POINTER)[i])->~T_CLASS (); \
              } \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#if defined(TW_EXPLICIT_TEMPLATE_DESTRUCTOR_TAKES_ARGS)
#   define TW_DES_FREE_TEMPLATE2(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS <T_PARAM1, T_PARAM2> (); \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#else
#   define TW_DES_FREE_TEMPLATE2(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS (); \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#endif /* defined(TW_EXPLICIT_TEMPLATE_DESTRUCTOR_TAKES_ARGS) */
#if defined(TW_EXPLICIT_TEMPLATE_DESTRUCTOR_TAKES_ARGS)
#   define TW_DES_FREE_TEMPLATE3(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2,T_PARAM3) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS <T_PARAM1, T_PARAM2, T_PARAM3> (); \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#else
#   define TW_DES_FREE_TEMPLATE3(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2,T_PARAM3) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS (); \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#endif /* defined(TW_EXPLICIT_TEMPLATE_DESTRUCTOR_TAKES_ARGS) */
#if defined(TW_EXPLICIT_TEMPLATE_DESTRUCTOR_TAKES_ARGS)
#   define TW_DES_FREE_TEMPLATE4(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2,T_PARAM3, T_PARAM4) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS <T_PARAM1, T_PARAM2, T_PARAM3, T_PARAM4> (); \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#else
#   define TW_DES_FREE_TEMPLATE4(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2,T_PARAM3, T_PARAM4) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->~T_CLASS (); \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
#endif /* defined(TW_EXPLICIT_TEMPLATE_DESTRUCTOR_TAKES_ARGS) */
#   define TW_DES_ARRAY_FREE_TEMPLATE2(POINTER,SIZE,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2) \
     do { \
          if (POINTER) \
            { \
              for (size_t i = 0; \
                   i < SIZE; \
                   ++i) \
              { \
                (&(POINTER)[i])->~T_CLASS (); \
              } \
              DEALLOCATOR (POINTER); \
            } \
        } \
     while (0)
# else /* ! TW_HAS_WORKING_EXPLICIT_TEMPLATE_DESTRUCTOR */
#   define TW_DES_NOFREE_TEMPLATE(POINTER,T_CLASS,T_PARAMETER) \
     do { \
          if (POINTER) \
            { \
              (POINTER)->T_CLASS T_PARAMETER::~T_CLASS (); \
            } \
        } \
     while (0)
#   define TW_DES_ARRAY_NOFREE_TEMPLATE(POINTER,SIZE,T_CLASS,T_PARAMETER) \
     do { \
          if (POINTER) \
            { \
              for (size_t i = 0; \
                   i < SIZE; \
                   ++i) \
              { \
                (POINTER)[i].T_CLASS T_PARAMETER::~T_CLASS (); \
              } \
            } \
        } \
     while (0)
#     define TW_DES_FREE_TEMPLATE(POINTER,DEALLOCATOR,T_CLASS,T_PARAMETER) \
       do { \
            if (POINTER) \
              { \
                POINTER->T_CLASS T_PARAMETER::~T_CLASS (); \
                DEALLOCATOR (POINTER); \
              } \
          } \
       while (0)
#     define TW_DES_ARRAY_FREE_TEMPLATE(POINTER,SIZE,DEALLOCATOR,T_CLASS,T_PARAMETER) \
       do { \
            if (POINTER) \
              { \
                for (size_t i = 0; \
                     i < SIZE; \
                     ++i) \
                { \
                  POINTER[i].T_CLASS T_PARAMETER::~T_CLASS (); \
                } \
                DEALLOCATOR (POINTER); \
              } \
          } \
       while (0)
#     define TW_DES_FREE_TEMPLATE2(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2) \
       do { \
            if (POINTER) \
              { \
                POINTER->T_CLASS <T_PARAM1, T_PARAM2>::~T_CLASS (); \
                DEALLOCATOR (POINTER); \
              } \
          } \
       while (0)
#     define TW_DES_FREE_TEMPLATE3(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2,T_PARAM3) \
       do { \
            if (POINTER) \
              { \
                POINTER->T_CLASS <T_PARAM1, T_PARAM2, T_PARAM3>::~T_CLASS (); \
                DEALLOCATOR (POINTER); \
              } \
          } \
       while (0)
#     define TW_DES_FREE_TEMPLATE4(POINTER,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2,T_PARAM3,T_PARAM4) \
       do { \
            if (POINTER) \
              { \
                POINTER->T_CLASS <T_PARAM1, T_PARAM2, T_PARAM3, T_PARAM4>::~T_CLASS (); \
                DEALLOCATOR (POINTER); \
              } \
          } \
       while (0)
#     define TW_DES_ARRAY_FREE_TEMPLATE2(POINTER,SIZE,DEALLOCATOR,T_CLASS,T_PARAM1,T_PARAM2) \
       do { \
            if (POINTER) \
              { \
                for (size_t i = 0; \
                     i < SIZE; \
                     ++i) \
                { \
                  POINTER[i].T_CLASS <T_PARAM1, T_PARAM2>::~T_CLASS (); \
                } \
                DEALLOCATOR (POINTER); \
              } \
          } \
       while (0)
# endif /* defined ! TW_HAS_WORKING_EXPLICIT_TEMPLATE_DESTRUCTOR */





#endif /* TW_MACROS_H_ */
