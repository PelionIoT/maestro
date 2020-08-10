/*
 * tw_array.h
 *
 *  Created on: Nov 28, 2011
 *      Author: ed
 * (c) 2011, WigWag LLC
 */

#ifndef TW_ARRAY_H_
#define TW_ARRAY_H_

#include <stdlib.h>

#include <TW/tw_alloc.h>
#include <TW/tw_log.h>

namespace TWlib {
/**
 * Dynamic Array
 * Template requirements:
 * T must support: (needs to be a primitive type, read on)
 * assignment operator
 * default constructor
 * sizeof() operator
 * (currently this limits T to primitive types, which is just fine. In C++11 it will be possible to overload sizeof() - until then, sorry.
 * The T class would also need to support placement new for arrays of objects.)
 *
 * ALLOC is a a TWlib Allocator style object.
 */
template<typename T, typename ALLOC>
class DynArray {
protected:
	ALLOC *_alloc;
	T *_array;
	int _size;
	void setup() {
		if(!_alloc)
			_array = (T *) ALLOC::malloc(sizeof(T) * _size);
		else
			_array = (T *) _alloc->i_malloc(sizeof(T) * _size);
	}

public:
	DynArray(ALLOC *a = NULL) : _alloc( a ), _array( NULL ), _size( 0 ) { }
	DynArray( DynArray<T,ALLOC> &o ) : _alloc( o._alloc ), _size( o._size ), _array( NULL ) {
		setup();
		ALLOC::memcpy((void *) _array, (void *) o._array, o._size * sizeof(T));
	}
	DynArray( int size, ALLOC *a = NULL) : _alloc( a ), _size( size ), _array( NULL ) {
		setup();
	}

	/**
	 * fills entire array with 0's. be careful not to loose objects with this.
	 */
	void zeroArray() {
		ALLOC::memset((void *) _array, 0, _size * sizeof(T) );
	}
	/**
	 * returns true if the value is in range, and fills 'val' with value
	 */
	bool get(int loc, T &val) {
		if(loc < _size) {
			val = *(_array + loc);
			return true;
		} else {
#ifdef _TW_DYNARRAY_RANGE_CHECK
		TW_DEBUG_L("DynArray<> - loc %d out of range\n",loc);
#endif
			return false;
		}
	}

	bool putByRef(int loc, T &val) {
		if(loc < _size) {
			*(_array + loc) = val;
			return true;
		} else {
#ifdef _TW_DYNARRAY_RANGE_CHECK
		TW_DEBUG_L("DynArray<> - loc %d out of range\n",loc);
#endif
			return false;
		}
	}

	bool put(int loc, const T val) {
		if(loc < _size) {
			*(_array + loc) = val;
			return true;
		} else {
#ifdef _TW_DYNARRAY_RANGE_CHECK
		TW_DEBUG_L("DynArray<> - loc %d out of range\n",loc);
#endif
			return false;
		}
	}

	/**
	 * dynamically grows (via realloc) array to hold this element
	 * @param val new value to add
	 */
	void addToEnd( T &val ) {
		_size++;
		resize(_size);
		*(_array + _size - 1) = val;
	}

	void addToEnd( DynArray<T,ALLOC> &o ) {
		if(o._size > 0) {
			int olds = _size;
			_size += o._size;
			resize(_size);
			ALLOC::memcpy((void *) (_array + olds), (void *) o._array, o._size * sizeof(T));
		}
	}

	void insert( int loc, DynArray<T,ALLOC> &o ) {
		if(o._size > 0) {
			int olds = _size;
			_size += o._size;
			resize(_size);
			ALLOC::memmove((void *) (_array+loc+o._size), (void *) (_array + loc), sizeof(T) * (olds - loc) );
			ALLOC::memcpy((void *) (_array + loc), (void *) o._array, o._size * sizeof(T));
		}
	}


	int size() { return _size; }

	void resize( int size, bool zeroout = false ) {
		int olds = _size;
		_size = size;
		if(_alloc)
			_array = (T *) _alloc->i_realloc(_array, _size * sizeof(T));
		else
			_array = (T *) ALLOC::realloc( _array, _size * sizeof(T));
		if(olds < _size && zeroout ) { // zeroout new area...
			ALLOC::memset( _array + olds, 0, (_size * sizeof(T)) - (olds * sizeof(T)));
		}

	}

	DynArray<T,ALLOC> &operator= (const DynArray<T,ALLOC> &o) {
		if(_array) {
			if (_alloc)
				_alloc->i_free(_array);
			else
				ALLOC::free(_array);
		}
		_size = o._size;
		_alloc = o._alloc;
		setup();
		ALLOC::memcpy((void *) _array, (void *) o._array, o._size * sizeof(T));
		return *this;
	}

	~DynArray() {
		if(_array) {
			if(_alloc)
				_alloc->i_free(_array);
			else
				ALLOC::free(_array);
		}
	}
};

} // end namespace
#endif /* TW_ARRAY_H_ */
