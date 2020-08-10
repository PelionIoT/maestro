// WigWag LLC
// (c) 2010
// tw_thread.h
// Author: ed

#ifndef _TW_TASK_H
#define _TW_TASK_H


#include <pthread.h>

#include <TW/tw_alloc.h>
#include <TW/tw_utils.h>
#include <TW/tw_sema.h>
#include <TW/tw_syscalls.h>
#include <TW/tw_fifo.h>

#include <string>

namespace TWlib {

class TaskManager;

typedef Allocator<Alloc_Std> TWTaskAllocator;  // the allocator used for some in-house thread info (plain-old malloc)

class BaseTask {
public:
	BaseTask();
//	BaseTask(TaskManager *manager );
	int startTask(void *val=NULL);            // starts the task with parameter val
	void nameTask( const char *s );
	char *name();
	std::string &appendName(std::string &s);
	void *waitForTask();                /// blocks until thread completes
	void *waitForTask(TimeVal &t);
	virtual void shutdown() = 0;           /// shuts down the task
	virtual ~BaseTask() {}
	bool isRunning();                  // started ?
	bool isCompleted();                // has task finished? (non-blocking)
	long getLWP();
protected:
	friend class TaskManager;
    virtual void *work( void *d ) = 0;                 // pure virtual - calls static work()

    struct workdata_t {     // this struct is to help us launch a pthread using do_work above.
    	BaseTask *_task;  // pointer to this object
    	void *_param;          // the parameter passed to work thread
    	char *_name;
    };
	workdata_t _workdat;
    pthread_t _pthread_dat;

    static void *do_work( void *taskdat ); // this is a work-around which basically just calls work above, but does so as a C call
	                                 // so pthread_create will work, see: http://www.osix.net/modules/article/?id=450

//	pthread_mutex_t _pthread_mutex;  // mutex used to protect internal Task vars
    TW_Mutex _thread_mutex; /// mutex used to protect internal Task vars
	bool _running;
	bool _completed;
	void *_thrd_retval;
	long _lwp_num;
};
/*
template <typename T>
typedef T*(*taskFunc)(T *obj);

template <class T, taskFunc<T> W>
class Task2 : public BaseTask {
	public:
		Task2();
		int startTask(T *val);            // starts the task with parameter val

		// should be specialized, can be overriden
		virtual T *worktask( T *v);

		virtual void shutdown();           // shuts down the task
		void *waitForTask();                // blocks until thread completes
		void *getRetval();
	protected:
		virtual void *work( void *d );
	//    struct taskdata_t {     // this struct is to help us launch a pthread using do_work above.
	// /   	Task<T> *_task;  // pointer to this object
	//    	T *_param;          // the parameter passed to work thread
	// /   };

	//	taskdata_t _taskdat;
	};

};

*/
template <class T>
class Task : public BaseTask {
public:
	Task();
	int startTask(T *val);            // starts the task with parameter val

	// should be specialized, can be overriden as virtual
	virtual T *worktask( T *v);

	virtual void shutdown();           // shuts down the task
	//void *waitForTask();                // blocks until thread completes
//	bool isRunning();                  // started ?
//	bool isCompleted();                // has task finished? (non-blocking)
	void *getRetval();
protected:
	virtual void *work( void *d );
//    struct taskdata_t {     // this struct is to help us launch a pthread using do_work above.
// /   	Task<T> *_task;  // pointer to this object
//    	T *_param;          // the parameter passed to work thread
// /   };

//	taskdata_t _taskdat;
};

template <typename T, typename WORKFUNC>
class Job {
protected:
	T workdat;
	friend class JobRunner;
public:
	void setWorkData( T t );
	void getCompletedData( T t );
	// specialize this function
	void work( T *t );
	void waitForJobCompletion( TimeVal &t );
	void waitForJobCompletion();
};

class JobRunner : public BaseTask {
public:
	JobRunner( bool wait_for_jobs = false );
	virtual void shutdown();           // shuts down the task
	void *waitForTask();                // blocks until thread completes
	bool isRunning();                  // started ?
	bool isCompleted();                // has task finished? (non-blocking)
protected:
    virtual void *work( void *d ) = 0;   // pure virtual - must be implemented by user. does the work of sending the client...

    static void *do_work( void *taskdat ); // this is a work-around which basically just calls work above, but does so as a C call
	                                       // so pthread_create will work, see: http://www.osix.net/modules/article/?id=450
};

class TaskManager {
public:
	TaskManager() : _listMutex(), _list() {}
	void addTask( BaseTask *t );
	void joinAll();
	void joinAll(TimeVal &t);
	void shutdownAll();
	void shutdownAll(TimeVal &t);
	// DEBUG functions: to work properly these will need _TW_TASK_DEBUG_THREADS_ defined
	string &dumpThreadInfo(string &s);
protected:
	TW_Mutex _listMutex;
 	tw_safeFIFO<BaseTask *,Allocator<Alloc_Std> > _list;
};




} // end namespace


template <class T>
Task<T>::Task() : BaseTask()
{ }

/** starts the task with parameter val
 *
 * @param val Initial value to hand thread upon start
 * @return returns results from pthread_create. Zero if successful, and -1 if already started.
 */
/*template <class T>
int Task<T>::startTask(T *val) {
	int ret = -1;
	bool start = false;
	pthread_mutex_lock(&_pthread_mutex);
	if(!_running)
		start = true;
	pthread_mutex_unlock(&_pthread_mutex);
	if (start) {
		_taskdat._param = val;
		_taskdat._task = this;
		ret = pthread_create(&_pthread_dat, NULL, Task<T>::do_work,
				(void *) &_taskdat);
		if (ret == 0) {
			pthread_mutex_lock(&_pthread_mutex);
			_running = true;
			pthread_mutex_unlock(&_pthread_mutex);
		}
	}
	return ret;
}
*/

/**
 * Should be specialized by users. Can be overriden virtual as well.
 * @param val
 * @return
 */
template <class T>
int Task<T>::startTask(T *val) {
	return BaseTask::startTask((void *) val);
}

template <class T>
void *Task<T>::work(void *d) {
	return worktask( reinterpret_cast<T *>(_workdat._param) );
}

//template <class T>
//virtual T *worktask( T *v) { }

/** waits for the Task class's thread to complete.
 *  this blocks until completion.
 */
//template <class T>
//void *Task<T>::waitForTask() {
//	int ret = pthread_join( _pthread_dat, &_thrd_retval );
//	return _thrd_retval;
//}



/** @return returns the thread's return value if its complete. NULL otherwise.
 */
template <class T>
void *Task<T>::getRetval() {
	void *ret = NULL;
	_thread_mutex.acquire();
	ret = _thrd_retval;
	_thread_mutex.release();
	return ret;
}




template <class T>
void Task<T>::shutdown( ) { }


#endif
