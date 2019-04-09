/*
 * tw_khash2.h
 *
 *  Created on: Jan 5, 2013
 *      Author: Ed Hemphill (ed@wigwag.com)
 * (c) 2012, WigWag LLC
 */

#ifndef TW_RBTREE_H
#define TW_RBTREE_H

#include <TW/tw_alloc.h>
#include <TW/provos_rb_tree.h>



namespace TWlib {

template <typename T>
class SplayTree {

};

/**
 * T is the data each node will point to. Note RB_Tree will manage pointers to T, not T itself. Therefore T can be any type.
 * CMP is a comparison function in a struct such as:
 *
struct cmp_ares_tasks {
int operator()(const ares_task_t* a, const ares_task_t* b) {
  if (a->sock < b->sock) return -1;
  if (a->sock > b->sock) return 1;
  return 0;
}
}
 * ALLOC is a TWlib::Allocator derived class, for instance TWlib::Allocator<TWlib::Alloc_Std>
 *
 */


template <typename T, typename CMP, typename ALLOC>
class RB_Tree {
protected:

	struct rb_node {
		T data;
		RB_ENTRY(rb_node) node;
	};
	RB_HEAD(rb_head_t, rb_node) rb_head;

	static inline int cmp_wrapper(const rb_node *a, const rb_node *b) {
		static CMP cmp;
		return cmp(a->data, b->data);
	}


	RB_GENERATE_STATIC_CPP(rb_head_t, rb_node, node, cmp_wrapper)

public:
	RB_Tree() {
		RB_INIT(&rb_head);
	}

	~RB_Tree() {
		rb_node *walk = NULL;
		RB_FOREACH(walk, rb_head_t, &rb_head) {
			;
		}
	}

	class Iter {

	protected:
		rb_node *node;
		RB_Tree &_tree;
	public:
		Iter(RB_Tree &tree) : _tree( tree ), node(NULL) {}
		void startMax() {
			node = RB_MAX(rb_head_t, &_tree.rb_head);
		}
		void startMin() {
			node = RB_MIN(rb_head_t, &_tree.rb_head);
		}
		/**
		 * Get's the next (greater) element of T
		 * return NULL if at end, T * of data otherwise.
		 */
		T getNext() {
			T ret = NULL;
			if(!node) return NULL;
			rb_node *nex = RB_NEXT(rb_head_t, &tree.rb_head, node);
			if(nex) {
				node = nex;
				ret = node->data;
			}
			return ret;
		}
		/* Get's the previous (less than) element of T
		 * return NULL if at end, T * of data otherwise.
		 */
		T getPrev() {
			T ret = NULL;
			if(!node) return NULL;
			rb_node *nex = RB_PREV(rb_head_t, &tree.rb_head, node);
			if(nex) {
				node = nex;
				ret = node->data;
			}
			return ret;
		}

		T current() {
			if(!node) return NULL;
			return node->data;
		}
	};

	void insert( T &data ) {
		rb_node *newnode = (rb_node*)ALLOC::calloc(1, sizeof(rb_node));
		newnode->data = data;
		RB_INSERT(rb_head_t, &rb_head, newnode);
	}

	/**
	 * Finds T entry where data == entry
	 */
	T find( const T &data ) {
		rb_node lookup;
		lookup.data = data;
		rb_node *found = RB_FIND(rb_head_t, &rb_head, &lookup);
		if(found)
			return found->data;
		else
			return NULL;
	}

	bool find( const T &data, T &fill ) {
		rb_node lookup;
		lookup.data = data;
		rb_node *found = RB_FIND(rb_head_t, &rb_head, &lookup);
		if(found) {
			fill = found->data;
			return true;
		}
		else
			return false;
	}

	/**
	 * Finds the first entry T where the entry in the tree is >= 'data' search key
	 */
	T nfind( const T &data ) {
		rb_node lookup;
		lookup.data = const_cast<T *>(&data);
		return RB_NFIND(rb_head_t, &rb_head, &lookup);
	}


	T remove( const T &data ) {
		T ret = NULL;
		rb_node lookup;
		lookup.data = data;
		rb_node *found = RB_FIND(rb_head_t, &rb_head, &lookup);
		if(found) {
			ret = found->data;
			RB_REMOVE(rb_head_t, &rb_head, found);
			ALLOC::free(found);
		}
		return ret;
	}

	bool isEmpty() {
		return RB_EMPTY(&rb_head);
	}

};

} // end namespace

#endif // TW_RBTREE_H
