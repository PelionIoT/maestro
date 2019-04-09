/*
 * tw_hashes.h
 *
 *  Created on: Nov 25, 2011
 *      Author: ed
 * (c) 2014, WigWag Inc
 */

#ifndef TW_HASHES_H_
#define TW_HASHES_H_

#include <limits.h>

namespace TWlib {

template<class T>
struct tw_hash {
	size_t operator()(T x) const {
		return 0;
	}
};

struct hashInt {
	inline size_t operator()(const int &x) const {
		return (size_t) (x);
	}
};

struct hash_uint32t {
	inline size_t operator()(const uint32_t &x) const {
		return (size_t) (x);
	}
};

struct uint32_eqFunc {
	inline int operator() (const uint32_t *l, const uint32_t *r) const {
		return (*l == *r);
	}
};


template <typename T>
struct eqstr_numericP {
	  inline int operator() (const T *kt1,
	                  const T *kt2) const
	  {
//		  TW_DEBUG("--------- COMPARE...\n",NULL);
		  if (*kt1 == *kt2) return 1;
		  else return 0;
	  }
};

template <typename T>
struct eqstr_numeric {
	  inline int operator() (const T kt1,
	                  const T kt2) const
	  {
//		  TW_DEBUG("--------- COMPARE...\n",NULL);
		  if (kt1 == kt2) return 1;
		  else return 0;
	  }
};

} // end namespace
#endif /* TW_HASHES_H_ */
