/*
 * tw_task.cpp
 *
 *  Created on: Nov 23, 2011
 *      Author: ed
 * (c) 2011, WigWag LLC
 */

#include <TW/tw_task.h>
#include <TW/tw_sema.h>

using namespace TWlib;


BaseTask::BaseTask() :
	_running( false ),
	_completed( false ),
	_thrd_retval( NULL ),
	_lwp_num( 0 ),
	_thread_mutex()
{
	_workdat._name = NULL;
}

/*
BaseTask::BaseTask(TaskManager *tmgr) :
	_running( false ),
	_completed( false ),
	_thrd_retval( NULL ),
	_lwp_num( 0 )
{
	tmgr->addTask(this);
}
*/

/** @return returns the thread's return value if its complete. NULL otherwise.
 */
long BaseTask::getLWP() {
	long ret = false;
	_thread_mutex.acquire();
	ret = _lwp_num;
	_thread_mutex.release();
	return ret;
}

/**
 * Starts the task's thread.
 * @param val
 * @return returns 0 on success. Returns value is the same as that of pthread_create()
 */
int BaseTask::startTask(void *val) {
	int ret = -1;
	bool start = false;
	_thread_mutex.acquire();
	if(!_running)
		start = true;
	_thread_mutex.release();
	if (start) {
		_workdat._param = val;
		_workdat._task = this;
		ret = pthread_create(&_pthread_dat, NULL, do_work, &_workdat);
		if (ret == 0) {
			_thread_mutex.acquire();
			_running = true;
			_thread_mutex.release();
		}
	}
	return ret;
}

/// @param s a NULL terminated string. A copy will be made, and will become the Thread's 'name'
void BaseTask::nameTask( const char *s ) {
	_thread_mutex.acquire();
	if(_workdat._name)
		TWTaskAllocator::free(_workdat._name);
	_workdat._name = (char *) TWTaskAllocator::malloc(strlen(s) + 1);
	::strcpy(_workdat._name,s);
	_thread_mutex.release();
}

/// @return a pointer to the given name of the string. If no name was given, this will be NULL.
char *BaseTask::name() {
	char *ret = NULL;
	_thread_mutex.acquire();
	ret =  _workdat._name;
	_thread_mutex.release();
	return ret;
}

/// @param s A std::string to append the Task's 'name' to. The string is assumed to be valid,
/// and will not be erased - only appended to.
/// If the Task does not have an assigned 'name' one will be created using the LWP number.
/// @return reference to same string.
std::string &BaseTask::appendName(std::string &s) {
	char *n = NULL;
	_thread_mutex.acquire();
	n =  _workdat._name;
	_thread_mutex.release();
	if(n) {
		s.append(n);
	} else {
		char b[20];
		string s2("Task LWP: ");
		s2.append(TWlib::convInt(b,(int) getLWP(),20));
		s.append(s2);
	}
	return s;
}


/**
 * this is the function handed to pthread_create. taskdat points to a Task<T>::workdata_t struct
 * which has both a parameter for the work() function, plus a pointer to the task object
 * @param taskdat
 */
void *BaseTask::do_work( void *workdat ) {
	struct workdata_t *dat = (struct workdata_t *) workdat;
	dat->_task->_lwp_num = ::_TW_getLWPnum(); // get the LWP number and assign it
	void *ret = dat->_task->work( dat->_param );
//	pthread_mutex_lock(&dat->_task->_pthread_mutex);
	dat->_task->_thread_mutex.acquire();
	dat->_task->_completed = true;
//	pthread_mutex_unlock(&dat->_task->_pthread_mutex);
	dat->_task->_thread_mutex.release();
	return ret;
}

/** @return returns true if the thread has been started. Returns false if threads is not started, or has ended.
 */
bool BaseTask::isRunning() {
	bool ret = false;
//	pthread_mutex_lock(&_pthread_mutex);
	_thread_mutex.acquire();
	ret = _running;
	_thread_mutex.release();
//	pthread_mutex_unlock(&_pthread_mutex);
	return ret;
}

/** @return returns true if the thread has executed & finished.
 */
bool BaseTask::isCompleted() {
	bool ret = false;
//	pthread_mutex_lock(&_pthread_mutex);
	_thread_mutex.acquire();
	ret = _completed;
	_thread_mutex.release();
//	pthread_mutex_unlock(&_pthread_mutex);
	return ret;
}

void *BaseTask::waitForTask(void) {
	int ret = pthread_join( _pthread_dat, &_thrd_retval );
	return _thrd_retval;
}

void *BaseTask::waitForTask(TimeVal &t) {
	// NOTABLE Not portable - Linux only
	// http://www.kernel.org/doc/man-pages/online/pages/man3/pthread_tryjoin_np.3.html
	int ret = pthread_timedjoin_np( _pthread_dat, &_thrd_retval, t.timespec() );
	return _thrd_retval;
}

void TaskManager::addTask( BaseTask *t ) {
	_list.add(t);
#ifdef _TW_TASK_DEBUG_THREADS_
		TW_DEBUG_LT("Adding thread LWP %d to TaskManager %x\n",t->getLWP(), this);
#endif
}

void TaskManager::joinAll() {
	tw_safeFIFO<BaseTask *,Allocator<Alloc_Std> >::iter _iter;
	_list.startIter(_iter);
	BaseTask *task;
#ifdef _TW_TASK_DEBUG_THREADS_
	TW_DEBUG_LT("Manager %x @ joinAll()\n",this);
#endif
	while(_iter.getNext(task)) {
#ifdef _TW_TASK_DEBUG_THREADS_
		TW_DEBUG_LT("Waiting for thread LWP %d ...\n",task->getLWP());
#endif
		task->waitForTask();
#ifdef _TW_TASK_DEBUG_THREADS_
		TW_DEBUG_LT("Thread LWP %d done.\n",task->getLWP());
#endif
	}
	_list.releaseIter(_iter);
}

void TaskManager::joinAll(TimeVal &v) {
	tw_safeFIFO<BaseTask *,Allocator<Alloc_Std> >::iter _iter;
	_list.startIter(_iter);
	BaseTask *task;
	while(_iter.getNext(task)) {
#ifdef _TW_TASK_DEBUG_THREADS_
		TW_DEBUG_LT("Waiting for thread LWP %d or timeout.\n",task->getLWP());
#endif
		task->waitForTask(v);
#ifdef _TW_TASK_DEBUG_THREADS_
		TW_DEBUG_LT("Thread LWP %d done.\n",task->getLWP());
#endif
	}
	_list.releaseIter(_iter);
}

string &TaskManager::dumpThreadInfo(string &s) {
	char buf[20];
	s.clear();
//	TW_DEBUG("dumpthreadinfo\n",NULL);
	s.append("TaskManager dump:\n");
	tw_safeFIFO<BaseTask *,Allocator<Alloc_Std> >::iter _iter;
	_list.startIter(_iter);
	BaseTask *task;
	int c = 0;
	while(_iter.getNext(task)) {
//		TW_DEBUG("dumpthreadinfo.getNext\n",NULL);
		s.append("   ");
		s.append(TWlib::convInt(buf,(int) c,20));
		s.append(": ");
		task->appendName(s);
//		TW_DEBUG("dumpthreadinfo.afterappendname\n",NULL);
		s.append(" (LWP:");
		s.append(TWlib::convInt(buf,(int) task->getLWP(),20));
//		TW_DEBUG("dumpthreadinfo.afterconvint\n",NULL);
		s.append(")\n");
	}
	_list.releaseIter(_iter);
//	TW_DEBUG("dumpthreadinfo.return\n",NULL);
	return s;
}

void TaskManager::shutdownAll() {
	tw_safeFIFO<BaseTask *,Allocator<Alloc_Std> >::iter _iter;
	_list.startIter(_iter);
	BaseTask *task;
	while(!_iter.getNext(task)) {
		task->shutdown();
	}
	_list.releaseIter(_iter);
}
