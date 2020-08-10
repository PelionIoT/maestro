// WigWag LLC
// (c) 2010
// tw_socktask.h
// Author: ed
// Sep 29, 2010

/*
 * tw_socktask.h
 *
 *  Created on: Sep 29, 2010
 *      Author: ed
 */

#ifndef TW_SOCKTASK_H_
#define TW_SOCKTASK_H_


#include <tw_fifo.h>
#include <tw_task.h>
#include <tw_bufblk.h>

class tw_io_handle {

};


class tw_socket_queue : public tw_safeFIFO<tw_bufblk> {};


class tw_socklistenmgr : public tw_task<tw_socket_queue> {
public:
	void *work( tw_socket_queue *queue );
};

// this class manages writing one or more on-going connections...
class tw_sockwritemgr : public tw_task<tw_socket_queue> {
public:
	void *work( tw_socket_queue *queue );
};


// this class manages reading one or more on-going connections...
class tw_sockreadmgr : public tw_task<tw_socket_queue> {
public:
	void *work( tw_socket_queue *queue );
};


#endif /* TW_SOCKTASK_H_ */
