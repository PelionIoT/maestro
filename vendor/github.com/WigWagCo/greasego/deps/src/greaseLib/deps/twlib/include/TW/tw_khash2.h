/*
 * tw_khash2.h
 *
 *  Created on: Mar 13, 2012
 *      Author: ed
 * (c) 2012, WigWag LLC
 */

#ifndef TW_KHASH2_H_
#define TW_KHASH2_H_


/* The MIT License

   Copyright (c) 2008, by Attractive Chaos <attractivechaos@aol.co.uk>

   Permission is hereby granted, free of charge, to any person obtaining
   a copy of this software and associated documentation files (the
   "Software"), to deal in the Software without restriction, including
   without limitation the rights to use, copy, modify, merge, publish,
   distribute, sublicense, and/or sell copies of the Software, and to
   permit persons to whom the Software is furnished to do so, subject to
   the following conditions:

   The above copyright notice and this permission notice shall be
   included in all copies or substantial portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
   EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
   MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
   NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
   BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
   ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
   SOFTWARE.
*/
/**
   Standalone hash library is a free C++ template library for hash tables.
   It implements open-address hashing with the "double hashing" technique,
   which makes this library different from most of other implementations,
   such as STL and TR1 hash based classes, which are all based on chained
   hashing. However, fewer implementations do not mean open-address hashing
   is less efficient. As a matter of fact, this hash library is at least as
   efficient, in terms of both time and space, as STL and TR1 libraries, and
   in some cases outperforms others.
 */

/*
  2008-09-08, 2.0.5:

    * fixed a compiling error in erase(iterator &)

  2008-08-31, 2.0.4:

    * fixed a compiling error
    * fixed a bug in erase()

  2008-08-30, 2.0.3:

    * fixed a compiling error in htmap_t::erase

  2008-08-29, 2.0.2:

    * fixed a compiling error in htset_t::insert

  2008-08-29, 2.0.1:

    * renamed class names
    * fixed a bug and reduced number of comparisons as well

  2008-08-29, 2.0.0:

    * redesign interface

  2008-08-28, 1.9.7:

    * use AC_HASH_INT by default. This speeds random integer input.

  2008-08-28, 1.9.6:

    * add linear probing, though not by default
    * fix a flaw in #define. It should not cause any problem.

  2008-08-26, 1.9.5: change function names
  2008-03-03, 1.9.4: add "const" to ::clone(), although the compiler
                     complains nothing
  2008-03-03, 1.9.3: reduce unnecessary memory consumption in ::flags
                     ::clone() method for hash_set_misc and hash_map_misc
  2008-02-07, 1.9.2: change license to the MIT license
  2007-07-18, 1.9.2: fixed a bug in hash_*_char when an element is erased.
 */

#ifndef AC_KHASH_HH_
#define AC_KHASH_HH_

#include <string.h>
#include <stdlib.h>
#include <stdint.h>

#define AC_KHASH_VERSION "2.0.5"

// even on 64-bit systems, khashint_t should be uint32_t
typedef uint32_t khashint_t;

/*
 * Hash functions
 */
// Do a web search "g_str_hash X31_HASH" for more information.
/** hash function for strings (char*) */
inline khashint_t __ac_X31_hash_string(const char *s)
{
    khashint_t h = *s;
    if (h) for (++s ; *s; ++s) h = (h << 5) - h + *s;
    return h;
}
/** Jenkins' hash function for 32-bit integers. Not used in this library. */
inline uint32_t __ac_Jenkins_hash_int(uint32_t key)
{
    key += (key << 12);
    key ^= (key >> 22);
    key += (key << 4);
    key ^= (key >> 9);
    key += (key << 10);
    key ^= (key >> 2);
    key += (key << 7);
    key ^= (key >> 12);
    return key;
}
/** Jenkins' hash function for 64-bit integers. Used when AC_HASH_INT macro is set. */
inline uint64_t __ac_Jenkins_hash_64(uint64_t key)
{
    key += ~(key << 32);
    key ^= (key >> 22);
    key += ~(key << 13);
    key ^= (key >> 8);
    key += (key << 3);
    key ^= (key >> 15);
    key += ~(key << 27);
    key ^= (key >> 31);
    return key;
}
/** Wang's hash function for 32-bit inegers. Used when AC_HASH_INT macro is set. */
inline uint32_t __ac_Wang_hash_int(uint32_t key)
{
    key += ~(key << 15);
    key ^=  (key >> 10);
    key +=  (key << 3);
    key ^=  (key >> 6);
    key += ~(key << 11);
    key ^=  (key >> 16);
    return key;
}

/** common hash functions */
struct khashf_t {
    inline khashint_t operator () (uint16_t key) const { return key; }
    inline khashint_t operator () (uint32_t key) const { return key; }
    inline khashint_t operator () (int key) const { return key; }
    inline khashint_t operator () (uint64_t key) const { return uint32_t(key>>19) ^ uint32_t(key); }
    inline khashint_t operator () (const char *p) const { return __ac_X31_hash_string(p); }
};

/** alternative common hash functions */
struct khashf2_t {
    inline khashint_t operator () (uint16_t key) const { return __ac_Wang_hash_int(uint32_t(key)); }
    inline khashint_t operator () (uint32_t key) const { return __ac_Wang_hash_int(key); }
    inline khashint_t operator () (int key) const { return __ac_Wang_hash_int(key); }
    inline khashint_t operator () (uint64_t key) const { return (khashint_t)__ac_Jenkins_hash_64(key); }
    inline khashint_t operator () (const char *p) const { return __ac_X31_hash_string(p); }
};

/** common key comparisons */
struct khasheq_t {
    inline bool operator () (uint16_t a, uint16_t b) const { return a == b; }
    inline bool operator () (uint32_t a, uint32_t b) const { return a == b; }
    inline bool operator () (int a, int b) const { return a == b; }
    inline bool operator () (uint64_t a, uint64_t b) const { return a == b; }
    inline bool operator () (const char *a, const char *b) const { return strcmp(a, b) == 0; }
};

template<class T1, class T2>
struct kpair_t {
    T1 first;
    T2 second;
    kpair_t() {};
    kpair_t(const T1 &a, const T2 &b) : first(a), second(b) {};
};

/*
 * Table for primes
 */
const int __ac_HASH_PRIME_SIZE = 32;
static const uint32_t __ac_prime_list[__ac_HASH_PRIME_SIZE] =
{
  0ul,          3ul,          11ul,         23ul,         53ul,
  97ul,         193ul,        389ul,        769ul,        1543ul,
  3079ul,       6151ul,       12289ul,      24593ul,      49157ul,
  98317ul,      196613ul,     393241ul,     786433ul,     1572869ul,
  3145739ul,    6291469ul,    12582917ul,   25165843ul,   50331653ul,
  100663319ul,  201326611ul,  402653189ul,  805306457ul,  1610612741ul,
  3221225473ul, 4294967291ul
};

/** Threshold for rehashing */
const double __ac_HASH_UPPER = 0.77;

/*
 * Constants and macros for retrieve/set flags "isempty" and "isdel".
 */
typedef uint32_t __ac_flag_t;
const int __ac_FLAG_SHIFT = 4;
const int __ac_FLAG_MASK = 0xful;
const __ac_flag_t __ac_FLAG_DEFAULT = 0xaaaaaaaaul;

#define __ac_isempty(flag, i) ((flag[i>>__ac_FLAG_SHIFT]>>((i&__ac_FLAG_MASK)<<1))&2)
#define __ac_isdel(flag, i) ((flag[i>>__ac_FLAG_SHIFT]>>((i&__ac_FLAG_MASK)<<1))&1)
#define __ac_isboth(flag, i) ((flag[i>>__ac_FLAG_SHIFT]>>((i&__ac_FLAG_MASK)<<1))&3)
#define __ac_set_isdel_false(flag, i) (flag[i>>__ac_FLAG_SHIFT]&=~(1ul<<((i&__ac_FLAG_MASK)<<1)))
#define __ac_set_isempty_false(flag, i) (flag[i>>__ac_FLAG_SHIFT]&=~(2ul<<((i&__ac_FLAG_MASK)<<1)))
#define __ac_set_isboth_false(flag, i) (flag[i>>__ac_FLAG_SHIFT]&=~(3ul<<((i&__ac_FLAG_MASK)<<1)))
#define __ac_set_isdel_true(flag, i) (flag[i>>__ac_FLAG_SHIFT]|=1ul<<((i&__ac_FLAG_MASK)<<1))

/*
 * Auxiliary functions for search/insert/erase.
 */

template <class keytype_t, class hashf_t, class hasheq_t>
inline khashint_t __ac_hash_search_aux(const keytype_t &key, khashint_t m, const keytype_t *keys, const __ac_flag_t *flag,
                                      const hashf_t &hash_func, const hasheq_t &hash_equal)
{
    if (!m) return 0;
    khashint_t inc, k, i;
    k = hash_func(key);
    i = k % m;
    inc = 1 + k % (m - 1);
    khashint_t last = i;
    while (!__ac_isempty(flag, i) && (__ac_isdel(flag, i) || !hash_equal(keys[i], key))) {
        if (i + inc >= m) i = i + inc - m; // inc < m, and so never write this line as: "i += inc - m;"
        else i += inc;
        if (i == last) return m; // fail to find
    }
    return i;
}

template <class keytype_t, class hashf_t, class hasheq_t>
inline khashint_t __ac_hash_insert_aux(const keytype_t &key, khashint_t m, const keytype_t *keys, const __ac_flag_t *flag,
                                       const hashf_t &hash_func, const hasheq_t &hash_equal)
{
    khashint_t inc, k, i, site;
    site = m;
    k = hash_func(key);
    i = k % m;
    inc = 1 + k % (m - 1);

    khashint_t last = i;
    while (!__ac_isempty(flag, i) && (__ac_isdel(flag, i) || !hash_equal(keys[i], key))) {
        if (__ac_isdel(flag, i)) site = i;
        if (i + inc >= m) i = i + inc - m;
        else i += inc;
        if (i == last) return site;
    }
    if (__ac_isempty(flag, i) && site != m) return site;
    else return i;
}

template <class keytype_t, class hashf_t, class hasheq_t>
inline khashint_t __ac_hash_erase_aux(const keytype_t &key, khashint_t m, const keytype_t *keys, __ac_flag_t *flag,
                                      const hashf_t &hash_func, const hasheq_t &hash_equal)
{
    if (!m) return 0;
    khashint_t i;
    i = __ac_hash_search_aux(key, m, keys, flag, hash_func, hash_equal);
    if (i != m && !__ac_isempty(flag, i)) {
        if (__ac_isdel(flag, i)) return m; // has been deleted
        __ac_set_isdel_true(flag, i); // set "isdel" flag as "true"
        return i;
    } else return m;
}

/** "iterator" class for "hash_set_char" and "hash_set_misc" */
template <class keytype_t>
class __ac_hash_base_iterator
{
protected:
    khashint_t i;
    const keytype_t *keys;
    const __ac_flag_t *flags;
public:
    __ac_hash_base_iterator() {} // No initialization. This is unsafe, but reasonable use will not cause any problems.
    __ac_hash_base_iterator(khashint_t _i, const keytype_t *_keys, const __ac_flag_t *_flags)
        : i(_i), keys(_keys), flags(_flags) {};
    inline const keytype_t &operator & () const { return keys[i]; } // Keys should never be changed by an iterator.
    inline const keytype_t &key() const { return keys[i]; } // an alias of the operator "&"
    inline bool operator != (const __ac_hash_base_iterator &iter) const { return i != iter.i; }
    inline bool operator == (const __ac_hash_base_iterator &iter) const { return i == iter.i; }
    inline bool operator < (const __ac_hash_base_iterator &iter) const { return i < iter.i; }
    inline bool operator > (const __ac_hash_base_iterator &iter) const { return i > iter.i; }
    inline void operator ++ () { ++i; }
    inline void operator ++ (int) { ++i; }
    inline void operator -- () { --i; }
    inline void operator -- (int) { --i; }
    inline bool isfilled() const { return !__ac_isboth(flags, i); }
    inline bool operator + () const { return isfilled(); } // an alias of "isfilled()"
    inline khashint_t pos() const { return i; }
};

/** "iterator" class for "hash_map_char" and "hash_map_misc" */
template <class keytype_t, class valtype_t>
class __ac_hash_val_iterator : public __ac_hash_base_iterator<keytype_t>
{
protected:
    valtype_t *vals;
public:
    __ac_hash_val_iterator() {}
    __ac_hash_val_iterator(khashint_t _i, const keytype_t *_keys, const __ac_flag_t *_flags, valtype_t *_vals)
        : __ac_hash_base_iterator<keytype_t>(_i, _keys, _flags), vals(_vals) {};
    inline valtype_t &operator * () { return vals[this->i]; } // Values can be changed here.
    inline const valtype_t &value() const { return vals[this->i]; }
    inline void value(const valtype_t &v) { vals[this->i] = v; }
};

/** Base class of all hash classes */
template <class keytype_t, class hashf_t = khashf_t, class hasheq_t = khasheq_t>
class __ac_hash_base_class
{
protected:
    khashint_t n_capacity; /**< maximum size of the hash table */
    khashint_t n_size; /**< number of elements in hash table */
    khashint_t n_occupied; /**< number of cells that have not been flaged as "isempty" (n_capacity >= n_occupied >= n_size) */
    khashint_t upper_bound; /**< The upper bound. When n_occupied exceeds this, rehashing will be performed. */
    __ac_flag_t *flags; /**< flag array which stores the status "isempty" or "isdel" of each hash cell. */
    keytype_t *keys; /**< array that stores hash keys */
    // return 0 for unchanged, 1 for empty, 2 for deleted
    inline int direct_insert_aux(const keytype_t &key, khashint_t m, keytype_t *K, __ac_flag_t *F, khashint_t *i) {
        *i = __ac_hash_insert_aux(key, m, K, F, hashf_t(), hasheq_t());
        if (__ac_isempty(F, *i)) {
            K[*i] = key;
            __ac_set_isboth_false(F, *i);
            return 1;
        } else if (__ac_isdel(F, *i)) {
            K[*i] = key;
            __ac_set_isboth_false(F, *i);
            return 2;
        } else return 0;
    }
    inline bool resize_aux1(khashint_t *new_capacity, __ac_flag_t **new_flags) {
        khashint_t t;
        t = __ac_HASH_PRIME_SIZE - 1;
        while (__ac_prime_list[t] > *new_capacity) --t;
        *new_capacity = __ac_prime_list[t+1];
        if (n_size >= khashint_t(*new_capacity * __ac_HASH_UPPER + 0.5)) return false; // do not rehash
        keys = (keytype_t*)realloc(keys, *new_capacity * sizeof(keytype_t));
        if (keys == 0) return false; // insufficient memory?
        *new_flags = (__ac_flag_t*)malloc(((*new_capacity>>__ac_FLAG_SHIFT) + 1) * sizeof(__ac_flag_t));
        if (*new_flags == 0) { // insufficient memory?
            ::free(*new_flags); return false;
        }
        for (t = 0; t < ((*new_capacity>>__ac_FLAG_SHIFT) + 1); ++t)
            (*new_flags)[t] = __ac_FLAG_DEFAULT;
        return true;
    }
    inline void resize_aux2(khashint_t new_capacity, __ac_flag_t *new_flags) {
        ::free(flags);
        flags = new_flags;
        n_capacity = new_capacity;
        n_occupied = n_size;
        upper_bound = khashint_t(n_capacity * __ac_HASH_UPPER + 0.5);
    }
    /** Test whether rehashing is needed and perform rehashing if this is the fact. */
    inline void rehash() {
        if (n_occupied >= upper_bound) {
            if (n_capacity > (n_size<<1)) resize(n_capacity - 1); // do not enlarge
            else resize(n_capacity + 1); // enlarge the capacity
        }
    }
public:
    __ac_hash_base_class(void) {
        keys = 0; flags = 0;
        n_capacity = n_size = n_occupied = upper_bound = 0;;
    }
    ~__ac_hash_base_class(void) { ::free(keys); ::free(flags); }
    /** resize the hash table and perform rehashing */
    inline bool resize(khashint_t new_capacity) {
        __ac_flag_t *new_flags;
        if (!resize_aux1(&new_capacity, &new_flags)) return false;
        for (khashint_t j = 0; j != n_capacity; ++j) {
            if (__ac_isboth(flags, j) == 0) {
                keytype_t key = keys[j]; // take out the key
                __ac_set_isdel_true(flags, j); // mark "deleted"
                while (1) {
                    khashint_t inc, k, i;
                    k = hashf_t()(key);
                    i = k % new_capacity; // calculate the new position
                    inc = 1 + k % (new_capacity - 1);
                    while (!__ac_isempty(new_flags, i)) {
                        if (i + inc >= new_capacity) i = i + inc - new_capacity;
                        else i += inc;
                    }
                    __ac_set_isempty_false(new_flags, i);
                    if (i < this->n_capacity && __ac_isboth(flags, i) == 0) { // something is here
                        { keytype_t tmp = keys[i]; keys[i] = key; key = tmp; } // take it out
                        __ac_set_isdel_true(flags, i);
                    } else { // put key and quit the loop
                        keys[i] = key;
                        break;
                    }
                }
            }
        }
        resize_aux2(new_capacity, new_flags);
        return true;
    }
    inline void rehash(khashint_t n) { resize(n); }
    /** get n_size */
    inline khashint_t size(void) const { return n_size; };
    /** get n_capacity */
    inline khashint_t bucket_count(void) const { return n_capacity; };
    /** clear the hash table, but do not free the memory */
    inline void clear(void) {
        if (flags) {
            for (khashint_t t = 0; t < ((n_capacity>>__ac_FLAG_SHIFT) + 1); ++t)
                flags[t] = __ac_FLAG_DEFAULT;
        }
        n_size = 0;
    }
    /** clear the hash table and free the memory */
    inline void free() {
        ::free(keys); ::free(flags);
        keys = 0; flags = 0;
        n_capacity = n_size = n_occupied = upper_bound = 0;;
    }
};

/** hash_set_misc class */
template <class keytype_t, class hashf_t = khashf_t, class hasheq_t = khasheq_t>
class khset_t : public __ac_hash_base_class<keytype_t, hashf_t, hasheq_t>
{
    typedef khset_t<keytype_t, hashf_t, hasheq_t> selftype_t;
    typedef kpair_t<__ac_hash_base_iterator<keytype_t>, bool> inspair_t;
public:
    typedef __ac_hash_base_iterator<keytype_t> iterator;

    khset_t(void) {};
    ~khset_t(void) {};

    /** clone */
    selftype_t *clone() const {
        selftype_t *h2;
        h2 = new selftype_t;
        memcpy(h2, this, sizeof(selftype_t));
        h2->flags = (__ac_flag_t*)malloc(((this->n_capacity>>__ac_FLAG_SHIFT)+1) * sizeof(__ac_flag_t));
        memcpy(h2->flags, this->flags, sizeof(__ac_flag_t) * ((this->n_capacity>>__ac_FLAG_SHIFT)+1));
        h2->keys = (keytype_t*)malloc(this->n_capacity * sizeof(keytype_t));
        memcpy(h2->keys, this->keys, this->n_capacity * sizeof(keytype_t));
        return h2;
    }
    /** search a key */
    inline iterator find(const keytype_t &key) {
        khashint_t i = __ac_hash_search_aux(key, this->n_capacity, this->keys, this->flags, hashf_t(), hasheq_t());
        return (i == this->n_capacity || __ac_isboth(this->flags, i))? this->end()
            : iterator(i, this->keys, this->flags);
    }
    /** insert a key */
    inline inspair_t insert(const keytype_t &key) {
        __ac_hash_base_class<keytype_t>::rehash();
        khashint_t i;
        int ret = direct_insert_aux(key, this->n_capacity, this->keys, this->flags, &i);
        if (ret == 0) return inspair_t(iterator(i, this->keys, this->flags), false);
        if (ret == 1) { ++(this->n_size); ++(this->n_occupied); }
        else ++(this->n_size); // then ret == 2
        return inspair_t(iterator(i, this->keys, this->flags), true);
    }
    /** delete a key */
    inline iterator erase(const keytype_t &key) {
        khashint_t i = __ac_hash_erase_aux(key, this->n_capacity, this->keys, this->flags, hashf_t(), hasheq_t());
        if (i != this->n_capacity) {
            --(this->n_size);
            return iterator(i, this->keys, this->flags);
        } else return this->end();
    }
    inline void erase(iterator &p) {
        if (p != this->end() && !__ac_isempty(this->flags, p.pos())) {
            if (!__ac_isdel(this->flags, p.pos())) {
                __ac_set_isdel_true(this->flags, p.pos());
                --(this->n_size);
            }
        }
    }
    /** the first iterator */
    inline iterator begin() { return iterator(0, this->keys, this->flags); }
    /** the last iterator */
    inline iterator end() { return iterator(this->n_capacity, this->keys, this->flags); }
};

/** hash_map_misc class */
template <class keytype_t, class valtype_t, class hashf_t = khashf_t, class hasheq_t = khasheq_t>
class khmap_t : public khset_t<keytype_t, hashf_t, hasheq_t>
{
    valtype_t *vals;
    /** a copy of __ac_hash_base_class<keytype_t>::rehash() */
    inline void rehash() {
        if (this->n_occupied >= this->upper_bound) {
            if (this->n_capacity > (this->n_size<<1)) resize(this->n_capacity - 1);
            else resize(this->n_capacity + 1);
        }
    }
    typedef khmap_t<keytype_t, valtype_t, hashf_t, hasheq_t> selftype_t;
    typedef kpair_t<__ac_hash_val_iterator<keytype_t, valtype_t>, bool> inspair_t;
public:
    khmap_t(void) { vals = 0; };
    ~khmap_t(void) { ::free(vals); };
    typedef __ac_hash_val_iterator<keytype_t, valtype_t> iterator;
    /** clone */
    selftype_t *clone() const {
        selftype_t *h2;
        h2 = new selftype_t;
        memcpy(h2, this, sizeof(selftype_t));
        h2->flags = (__ac_flag_t*)malloc(((this->n_capacity>>__ac_FLAG_SHIFT)+1) * sizeof(__ac_flag_t));
        memcpy(h2->flags, this->flags, sizeof(__ac_flag_t) * ((this->n_capacity>>__ac_FLAG_SHIFT)+1));
        h2->keys = (keytype_t*)malloc(this->n_capacity * sizeof(keytype_t));
        memcpy(h2->keys, this->keys, this->n_capacity * sizeof(keytype_t));
        h2->vals = (valtype_t*)malloc(this->n_capacity * sizeof(valtype_t));
        memcpy(h2->vals, this->vals, this->n_capacity * sizeof(valtype_t));
        return h2;
    }
    /** analogy of __ac_hash_base_class<keytype_t>::resize(khashint_t) */
    inline bool resize(khashint_t new_capacity) {
        __ac_flag_t *new_flags;
        if (!__ac_hash_base_class<keytype_t, hashf_t, hasheq_t>::resize_aux1(&new_capacity, &new_flags)) return false;
        vals = (valtype_t*)realloc(vals, sizeof(valtype_t) * new_capacity);
        if (vals == 0) { // insufficient enough memory?
            ::free(new_flags);
            return false;
        }
        for (khashint_t j = 0; j != this->n_capacity; ++j) {
            if (__ac_isboth(this->flags, j) == 0) {
                keytype_t key = this->keys[j]; // take out the key
                valtype_t val = vals[j];
                __ac_set_isdel_true(this->flags, j); // mark "deleted"
                while (1) {
                    khashint_t inc, k, i;
                    k = hashf_t()(key);
                    i = k % new_capacity; // calculate the new position
                    inc = 1 + k % (new_capacity - 1);
                    while (!__ac_isempty(new_flags, i)) {
                        if (i + inc >= new_capacity) i = i + inc - new_capacity;
                        else i += inc;
                    }
                    __ac_set_isempty_false(new_flags, i);
                    if (i < this->n_capacity && __ac_isboth(this->flags, i) == 0) { // something is here
                        { keytype_t tmp = this->keys[i]; this->keys[i] = key; key = tmp; } // take it out
                        { valtype_t tmp = vals[i]; vals[i] = val; val = tmp; } // take it out
                        __ac_set_isdel_true(this->flags, i);
                    } else { // clear
                        this->keys[i] = key;
                        vals[i] = val;
                        break;
                    }
                }
            }
        }
        __ac_hash_base_class<keytype_t, hashf_t, hasheq_t>::resize_aux2(new_capacity, new_flags);
        return true;
    }
    inline void rehash(khashint_t n) { resize(n); }
    inline iterator find(const keytype_t &key) {
        khashint_t i = __ac_hash_search_aux(key, this->n_capacity, this->keys, this->flags, hashf_t(), hasheq_t());
        if (i != this->n_capacity && __ac_isboth(this->flags, i) == 0) {
            return iterator(i, this->keys, this->flags, vals);
        } else return this->end();
    }
    inline inspair_t insert(const keytype_t &key, const valtype_t &val) {
        rehash();
        khashint_t i;
        int ret = direct_insert_aux(key, this->n_capacity, this->keys, this->flags, &i);
        vals[i] = val;
        if (ret == 0) return inspair_t(iterator(i, this->keys, this->flags, vals), false);
        if (ret == 1) { ++(this->n_size); ++(this->n_occupied); }
        else ++(this->n_size); // then ret == 2
        return inspair_t(iterator(i, this->keys, this->flags, vals), true);
    }
    inline valtype_t &operator[](const keytype_t &key) {
        rehash();
        khashint_t i;
        int ret = direct_insert_aux(key, this->n_capacity, this->keys, this->flags, &i);
        if (ret == 0) return vals[i];
        if (ret == 1) { ++(this->n_size); ++(this->n_occupied); }
        else ++(this->n_size); // then ret == 2
        vals[i] = valtype_t();
        return vals[i];
    }
    inline void erase(iterator &p) {
        if (p != this->end() && !__ac_isempty(this->flags, p.pos())) {
            if (!__ac_isdel(this->flags, p.pos())) {
                __ac_set_isdel_true(this->flags, p.pos());
                --(this->n_size);
            }
        }
    }
    inline iterator erase(const keytype_t &key) {
        khashint_t i = __ac_hash_erase_aux(key, this->n_capacity, this->keys, this->flags, hashf_t(), hasheq_t());
        if (i != this->n_capacity) {
            --(this->n_size);
            return iterator(i, this->keys, this->flags, vals);
        } else return this->end();
    }
    inline iterator begin() { return iterator(0, this->keys, this->flags, vals); }
    inline iterator end() { return iterator(this->n_capacity, this->keys, this->flags, vals); }
    inline void free() {
        khset_t<keytype_t, hashf_t, hasheq_t>::free(); ::free(vals); vals = 0;
    }
};

#endif // AC_KHASH_HH_

#endif /* TW_KHASH2_H_ */
