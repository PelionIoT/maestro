// WigWag LLC
// (c) 2010
// tw_alloc.h
// Author: ed
// Sep 30, 2010

/*
 * tw_alloc.h
 *
 *  Created on: Sep 30, 2010
 *      Author: ed
 */

#ifndef TW_ALLOC_H_
#define TW_ALLOC_H_

#include <TW/tw_types.h>
#ifdef __APPLE__
#include <stdlib.h>
#else
#include <malloc.h>
#endif
#include <stdio.h>
#include <string.h>

//#define MS_SYNC 1

namespace TWlib {

/**
 * The allocator class needs a struct T as:
 * struct T {
 * 		void *malloc(tw_size n);
 * 		free(void *p);
 * 		sync(void *addr, tw_size len, int flags);
 * }
 */

/**
 * plain-old standard allocator
 */
struct Alloc_Std {
	static void *malloc (tw_size nbytes) {  return ::malloc((int) nbytes); }
	static void *calloc (tw_size nelem, tw_size elemsize) { return ::calloc((size_t) nelem, (size_t) elemsize); }
	static void *realloc(void *d, tw_size s) { return ::realloc(d,(size_t) s); }
	static void free(void *p) { ::free(p); }
	static void sync(void *addr, tw_size len, int flags = 0) { } // does nothing - not shared memory
	static void *memcpy(void *d, const void *s, size_t n) { return ::memcpy(d,s,n); };
	static void *memmove(void *d, const void *s, size_t n) { return ::memmove(d,s,n); };
	static int memcmp(void *l, void *r, size_t n) { return ::memcmp(l, r, n); }
	static void *memset(void *d, int c, size_t n) { return ::memset(d,c,n); }
	static const char *ALLOC_NOMEM_ERROR_MESSAGE;
};

template <class T>
class Allocator {
//protected:
//	static Allocator<Alloc_Std> _default_alloc;
	static Allocator<T> *_instance;
public:
//	static Allocator *instance() { return &_default_alloc; }
	Allocator<T>() {}

// These are instance specific function. For Alloc_Std, these are exactly the same.
// But if it was a file based allocator, for instance, they would be different.
	void *i_malloc (tw_size nbytes) { return T::malloc(nbytes); }
	void *i_calloc (tw_size nume, tw_size esize) { return T::calloc(nume, esize); }
	void *i_realloc (void *d, tw_size nbytes) { return T::realloc(d,nbytes); }
	void i_free(void *p) { T::free(p); }
	void i_sync(void *addr, tw_size len, int flags = 0) { T::sync(addr,len,flags); }
	const char *i_error_nomemmsg() { return T::ALLOC_NOMEM_ERROR_MESSAGE; }

// These are all the same as getInstance()->malloc/free/sync()
	static void *malloc (tw_size nbytes) { return T::malloc(nbytes); }
	static void *calloc (tw_size nume, tw_size esize) { return T::calloc(nume, esize); }
	static void *realloc (void *d, tw_size nbytes) { return T::realloc(d,nbytes); }
	static void free(void *p) { T::free(p); }
	static void sync(void *addr, tw_size len, int flags = 0) { T::sync(addr,len,flags); }

// these functions are always static
	static void *memcpy(void *d, const void *s, size_t n) { return T::memcpy(d,s,n); };
	static void *memmove(void *d, const void *s, size_t n) { return T::memmove(d,s,n); };
	static int memcmp(void *l, void *r, size_t n) { return T::memcmp(l, r, n); }
	static void *memset(void *d, int c, size_t n) { return T::memset(d,c,n); }
	static Allocator<T> *getInstance() { if(!_instance) _instance = new Allocator<T>(); return _instance; }
	static const char *error_nomemmsg() { return T::ALLOC_NOMEM_ERROR_MESSAGE; }
};
template <class T> Allocator<T> *Allocator<T>::_instance = NULL;


/*

ACE_NEW( datb, ACE_Message_Block( len, ACE_Message_Block::MB_DATA, 0, 0, _alloc ) );

* do { datb = new(::std::nothrow) ACE_Message_Block( len, ACE_Message_Block::MB_DATA, 0, 0, _alloc ); \
if (datb == 0) { (*__errno_location ()) = 12; return; } \
} while (0)
*
*/


}

#define TW_ALLOC_ERROR_MSG_NO_MEM "*** MEMORY TWlib::Allocator FAILURE using ALLOC: %s:%d ***\n"

// this is a macro which prints an error, but does without using the tw_log system.
#define TW_PALLOC_ERROR( s, ... ) fprintf(stderr, s, __VA_ARGS__ )

// usage: TW_NEW( Allocator<Std_Alloc>, String, mystring, String() );
// uses placement to use Allocator to allocate, then calls constructor.
#define TW_NEW( newvar, type, constr, ALLOC )  do {  newvar = (type *) ALLOC::malloc(sizeof(type)); \
	if(!newvar) TW_PALLOC_ERROR( ALLOC::error_nomemmsg(), __FILE__, __LINE__ ); \
	else newvar = new (newvar) constr; } while(0)

// usage: TW_NEW_WALLOC( Allocator *a, String, mystring, String() );
// uses placement to use Allocator to allocate, then calls constructor.
#define TW_NEW_WALLOC( newvar, type, constr, ALLOCI )  do {  newvar = (type *) ALLOCI->i_malloc(sizeof(type)); \
	if(!newvar) TW_PALLOC_ERROR( TW_ALLOC_ERROR_MSG_NO_MEM, __FILE__, __LINE__ ); \
	else newvar = new (newvar) constr; } while(0)

// explicity calls destructor on 'oldvar' then frees memory using Allocator
#define TW_DELETE( oldvar, type, ALLOC )  do {  oldvar->~type(); ALLOC::free(oldvar); } while(0)

// explicity calls destructor on 'oldvar' then frees memory using an Allocator instance var
#define TW_DELETE_WALLOC( oldvar, type, ALLOCI )  do {  oldvar->~type(); ALLOCI->i_free(oldvar); } while(0)



// We aren't using all this crap. Too complicated. too much ACE. sucks.

#ifdef _DONT_DEFINE_ME

class TW_Allocator
{
public:

  /// Unsigned integer type used for specifying memory block lengths.
  typedef size_t size_type;

  // = Memory Management

  /// Get pointer to a default TW_Allocator.
  static TW_Allocator *instance (void);

  /// Set pointer to a process-wide TW_Allocator and return existing
  /// pointer.
  static TW_Allocator *instance (TW_Allocator *);

  /// Delete the dynamically allocated Singleton
  static void close_singleton (void);

  /// "No-op" constructor (needed to make certain compilers happy).
  TW_Allocator (void);

  /// Virtual destructor
  virtual ~TW_Allocator (void);

  /// Allocate @a nbytes, but don't give them any initial value.
  virtual void *malloc (size_type nbytes) = 0;

  /// Allocate @a nbytes, giving them @a initial_value.
  virtual void *calloc (size_type nbytes, char initial_value = '\0') = 0;

  /// Allocate <n_elem> each of size @a elem_size, giving them
  /// @a initial_value.
  virtual void *calloc (size_type n_elem,
                        size_type elem_size,
                        char initial_value = '\0') = 0;

  /// Free <ptr> (must have been allocated by <TW_Allocator::malloc>).
  virtual void free (void *ptr) = 0;

  /// Remove any resources associated with this memory manager.
  virtual int remove (void) = 0;

  // = Map manager like functions

  /**
   * Associate @a name with @a pointer.  If @a duplicates == 0 then do
   * not allow duplicate @a name/@a pointer associations, else if
   * @a duplicates != 0 then allow duplicate @a name/@a pointer
   * assocations.  Returns 0 if successfully binds (1) a previously
   * unbound @a name or (2) @a duplicates != 0, returns 1 if trying to
   * bind a previously bound @a name and @a duplicates == 0, else
   * returns -1 if a resource failure occurs.
   */
  virtual int bind (const char *name, void *pointer, int duplicates = 0) = 0;

  /**
   * Associate @a name with @a pointer.  Does not allow duplicate
   * @a name/@a pointer associations.  Returns 0 if successfully binds
   * (1) a previously unbound @a name, 1 if trying to bind a previously
   * bound @a name, or returns -1 if a resource failure occurs.  When
   * this call returns @a pointer's value will always reference the
   * void * that @a name is associated with.  Thus, if the caller needs
   * to use @a pointer (e.g., to free it) a copy must be maintained by
   * the caller.
   */
  virtual int trybind (const char *name, void *&pointer) = 0;

  /// Locate @a name and pass out parameter via pointer.  If found,
  /// return 0, returns -1 if failure occurs.
  virtual int find (const char *name, void *&pointer) = 0;

  /// Returns 0 if the name is in the mapping. -1, otherwise.
  virtual int find (const char *name) = 0;

  /// Unbind (remove) the name from the map.  Don't return the pointer
  /// to the caller
  virtual int unbind (const char *name) = 0;

  /// Break any association of name.  Returns the value of pointer in
  /// case the caller needs to deallocate memory.
  virtual int unbind (const char *name, void *&pointer) = 0;

  // = Protection and "sync" (i.e., flushing memory to persistent
  // backing store).

  /**
   * Sync @a len bytes of the memory region to the backing store
   * starting at @c this->base_addr_.  If @a len == -1 then sync the
   * whole region.
   */
  virtual int sync (ssize_t len = -1, int flags = MS_SYNC) = 0;

  /// Sync @a len bytes of the memory region to the backing store
  /// starting at @a addr.
  virtual int sync (void *addr, size_type len, int flags = MS_SYNC) = 0;

  /**
   * Change the protection of the pages of the mapped region to @a prot
   * starting at <this->base_addr_> up to @a len bytes.  If @a len == -1
   * then change protection of all pages in the mapped region.
   */
 // TODO
//  virtual int protect (ssize_t len = -1, int prot = PROT_RDWR) = 0;

  /// Change the protection of the pages of the mapped region to @a prot
  /// starting at @a addr up to @a len bytes.
// TODO
  //  virtual int protect (void *addr, size_type len, int prot = PROT_RDWR) = 0;

#if defined (TW_HAS_MALLOC_STATS)
  /// Dump statistics of how malloc is behaving.
  virtual void print_stats (void) const = 0;
#endif /* TW_HAS_MALLOC_STATS */

  /// Dump the state of the object.
  virtual void dump (void) const = 0;
private:
  // DO NOT ADD ANY STATE (DATA MEMBERS) TO THIS CLASS!!!!  See the
  // <TW_Allocator::instance> implementation for explanation.

  /// Pointer to a process-wide TW_Allocator instance.
  static TW_Allocator *allocator_;

  /// Must delete the <allocator_> if non-0.
  static int delete_allocator_;
};

#endif

#endif /* TW_ALLOC_H_ */
