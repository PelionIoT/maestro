twlib
=====

Tupperware container and utility library.

twlib aims to be a lightweight container and OS abstraction library for C++. It is built off ideas in the [ACE library](http://www.cs.wustl.edu/~schmidt/ACE-overview.html "ACE Overview"), a few features (very few) in Boost, and implementations of high performance container from various authors.

The library is built for simplicity, and code readability, while still using the most important features of C++. 

twlib provides the following and more:

* Buffer management, block chaining, and reference counting based on the ideas from the [ACE_Message_Block](http://www.dre.vanderbilt.edu/Doxygen/5.5/html/ace/classACE__Message__Block.html) except a simpler implementation with some addition functionality.
* Semaphore driven, thread-safe queueing classes.
* Allocator support via classes similar to that in boost / ACE.
* Thread safe logging.
* C++11 templates for thread-safe FIFOs and circular buffers.
* Simple, fast, non-STL C++ hashmap implementation based on [Google sparsehash and densehash](http://code.google.com/p/sparsehash/ "densehash") and [khash](Always update with latests code at: https://github.com/attractivechaos/klib/blob/master/test/khash_keith.c) by [Attractive Chaos](http://attractivechaos.wordpress.com/2008/08/28/comparison-of-hash-table-libraries/)
* Red-black tree implementation via C++ template from FreeBSD's RB tree, [originally](http://www.freebsd.org/cgi/cvsweb.cgi/~checkout~/src/sys/sys/tree.h?rev=1.9.4.2;content-type=text%2Fplain) by [Niels Provos](http://t-t-travails.blogspot.com/2008/04/left-leaning-red-black-trees-are-hard.html)

###Templates###

Many things such as FIFOs, hash tables and circular buffers are only templates - in that case no library build is needed.

###Building###

(only needed if using calls beyond the template library)

You will need [Google sparsehash and densehash](http://code.google.com/p/sparsehash/ "densehash") and [gtest](http://code.google.com/p/googletest/).

After cloning, take a look at the Makefile. You will need to place gtest and sparsehash in __exapanded_prereqs__ or another directory you provide.

Then just:

```
    make tw_lib
```

###Status###

twlib is a work in progress right now, but much of the code is well tested. 

___Todos:___

* automated build process
* move to [libuv](https://github.com/joyent/libuv) for cross platform support.
* disk based hash tables





