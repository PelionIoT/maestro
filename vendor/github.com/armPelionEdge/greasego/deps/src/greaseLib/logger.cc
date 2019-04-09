/*
 * logger.cc
 *
 *  Created on: Aug 27, 2014
 *      Author: ed
 * (c) 2014, WigWag Inc.
 */

/*
    MIT License

    Copyright (c) 2019, Arm Limited and affiliates.

    SPDX-License-Identifier: MIT
    
    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.
*/


#include "logger.h"

#include <stdio.h>
#include <string.h>
#include <errno.h>
#include <stdlib.h>

#include <TW/tw_fifo.h>
#include <TW/tw_khash.h>

//#include "nan.h"

#include "grease_client.h"
#include "error-common.h"

using namespace Grease;

#ifndef GREASE_LIB

Nan::Persistent<v8::Function> GreaseLogger::constructor;

NAN_METHOD(GreaseLogger::SetGlobalOpts) {

	GreaseLogger *l = GreaseLogger::setupClass();
	Local<Function> cb;

	if(info.Length() < 1 || !info[0]->IsObject()) {
		Nan::ThrowTypeError("setGlobalOpts: bad parameters");
		return;
	}

	Local<Object> jsObj = info[0]->ToObject();
//	l->levelFilterOutMask
	Local<Value> jsVal = jsObj->Get(Nan::New("levelFilterOutMask").ToLocalChecked());

	if(jsVal->Uint32Value()) {
		l->Opts.levelFilterOutMask = jsVal->Uint32Value();
	}

	jsVal = jsObj->Get(Nan::New("defaultFilterOut").ToLocalChecked());
	if(jsVal->IsBoolean()) {
		bool v = jsVal->ToBoolean()->BooleanValue();
		uv_mutex_lock(&l->modifyFilters);
		l->Opts.defaultFilterOut = v;
		uv_mutex_unlock(&l->modifyFilters);
	}

	jsVal = jsObj->Get(Nan::New("show_errors").ToLocalChecked());
	if(jsVal->IsBoolean()) {
		bool v = jsVal->ToBoolean()->BooleanValue();
		l->Opts.show_errors = v;
	}
	jsVal = jsObj->Get(Nan::New("callback_errors").ToLocalChecked());
	if(jsVal->IsBoolean()) {
		bool v = jsVal->ToBoolean()->BooleanValue();
		l->Opts.callback_errors = v;
	}
}

#endif

bool GreaseLogger::sift(logMeta &f) { // returns true, then the logger should log it
	bool ret = false;
	static uint64_t zero = 0;

	uint32_t mask = 0;
	uv_mutex_lock(&modifyFilters);
	mask = Opts.levelFilterOutMask;
	uv_mutex_unlock(&modifyFilters);

	if(mask & f.level)
		return false;

	if(!META_HAS_CACHE(f)) {  // if the hashes aren't cached, then we have not done this...
		getHashes(f.tag,f.origin,f._cached_hash);

		uv_mutex_lock(&modifyFilters);
		FilterList *list;
		if(filterHashTable.find(f._cached_hash[0],list)) { // check both tag and orgin
			ret = true;
			META_SET_LIST(f,0,list);
		}
		if(!ret && filterHashTable.find(f._cached_hash[1],list)) { // check just tag
			ret = true;
			META_SET_LIST(f,1,list);
		}
		if(!ret && filterHashTable.find(f._cached_hash[2],list)) { // check just origin...
			ret = true;
			META_SET_LIST(f,2,list);
		}

		if(!ret && filterHashTable.find(zero,list)) {
			ret = true;
			META_SET_LIST(f,3,list);
		}

		if(!ret && Opts.defaultFilterOut)        // if neither, then do we output by default?
			ret = false;
		else
			ret = true;

//		if(ret && META_HAS_IGNORES(f)) {
//			TargetId ignore_list = META_IGNORE_LIST(f);
//			int x = 0;
//			while(x < MAX_IGNORE_LIST) {
//				if(ignore_list[x] == 0) {
//					break;
//				}
//				x++;
//			}
//		}

		uv_mutex_unlock(&modifyFilters);
	} else
		ret = true;
	return ret;
}

// same as above, just uses a pointer
bool GreaseLogger::siftP(logMeta *f) { // returns true, then the logger should log it
	bool ret = false;
	static uint64_t zero = 0;

	uint32_t mask = 0;
	uv_mutex_lock(&modifyFilters);
	mask = Opts.levelFilterOutMask;
	uv_mutex_unlock(&modifyFilters);

	if(mask & f->level)
		return false;

	if(!META_HAS_CACHE((*f))) {  // if the hashes aren't cached, then we have not done this...
		getHashes(f->tag,f->origin,f->_cached_hash);

		uv_mutex_lock(&modifyFilters);
		FilterList *list;
		if(filterHashTable.find(f->_cached_hash[0],list)) { // check both tag and orgin
			ret = true;
			META_SET_LIST((*f),0,list);
		}
		if(!ret && filterHashTable.find(f->_cached_hash[1],list)) { // check just tag
			ret = true;
			META_SET_LIST((*f),1,list);
		}
		if(!ret && filterHashTable.find(f->_cached_hash[2],list)) { // check just origin...
			ret = true;
			META_SET_LIST((*f),2,list);
		}

		if(!ret && filterHashTable.find(zero,list)) {
			ret = true;
			META_SET_LIST((*f),3,list);
		}

		if(!ret && Opts.defaultFilterOut)        // if neither, then do we output by default?
			ret = false;
		else
			ret = true;

//		if(ret && META_HAS_IGNORES(f)) {
//			TargetId ignore_list = META_IGNORE_LIST(f);
//			int x = 0;
//			while(x < MAX_IGNORE_LIST) {
//				if(ignore_list[x] == 0) {
//					break;
//				}
//				x++;
//			}
//		}

		uv_mutex_unlock(&modifyFilters);
	} else
		ret = true;
	return ret;
}

GreaseLogger *GreaseLogger::LOGGER = NULL;

#if (UV_VERSION_MAJOR > 0)
void GreaseLogger::handleInternalCmd(uv_async_t *handle) {
#else
void GreaseLogger::handleInternalCmd(uv_async_t *handle, int status /*UNUSED*/) {
#endif
	GreaseLogger::internalCmdReq req;
	logTarget *t = NULL;
	bool ignore = false;
	while(GreaseLogger::LOGGER->internalCmdQueue.removeMv(req)) {
    	switch(req.c) {
    	case NEW_LOG:
    		{
    			// process a single log message
    			singleLog *l = (singleLog *) req.aux;
    			assert(l);
//    			FilterList *list = NULL;
    			if(GreaseLogger::LOGGER->sift(l->meta.m)) { // should always be true (checked in v8 thread)
    				bool wrote = false;
    				for(int i=0;i<GREASE_META_HASHLIST_CACHE_SIZE;i++) {
    					FilterList *LIST = META_GET_LIST(l->meta.m,i);
    					if (LIST && LIST->valid(l->meta.m.level)) {
    						int n = 0;
    						while(LIST->list[n].id != 0) {
    							logTarget *t = NULL;
    							ignore = false;
    							if(((LIST->list[n].levelMask & l->meta.m.level) > 0) && !LIST->list[n]._disabled) { //if we have a Filter that matches...
    								if(META_HAS_IGNORES(l->meta.m)) {
    									int x = 0;
    									while(x < MAX_IGNORE_LIST) {
    										if(l->meta.ignore_list[x] == 0) { // 0 tells end of list
    											break;
    										}
    										if(l->meta.ignore_list[x] == LIST->list[n].targetId) {
    											ignore = true;
    											break;
    										}
    										x++;
    									}
    								}
    								if(LIST->list[n].targetId == 0) {
    			    					//        						DBG_OUT("write out: %s",l->buf.handle.base);
    									if(GreaseLogger::LOGGER->defaultTarget)
    										GreaseLogger::LOGGER->defaultTarget->write(l->buf.handle.base,(uint32_t) l->buf.used,l->meta.m, &LIST->list[n]);
    									wrote = true;
    								} else
    								if(!ignore && GreaseLogger::LOGGER->targets.find(LIST->list[n].targetId, t)) {
    			    					//        						DBG_OUT("write out: %s",l->buf.handle.base);
        								t->write(l->buf.handle.base,(uint32_t) l->buf.used,l->meta.m, &LIST->list[n]);
        								wrote = true;
    								} else {
    			    					//        						DBG_OUT("Orphaned target id: %d\n",list->list[n].targetId);
    								}
    							} else {
    							}
    							n++;
    						}
    					}
    				}
    				if(!wrote && GreaseLogger::LOGGER->defaultTarget) {
        				GreaseLogger::LOGGER->defaultTarget->write(l->buf.handle.base,l->buf.used,l->meta.m,&GreaseLogger::LOGGER->defaultFilter);
        			}
//        			}
//        			else {  // write out to default target if the list does not apply to this level
//        		//		HEAVY_DBG_OUT("pass thru to default");
//        				GreaseLogger::LOGGER->defaultTarget->write(l->buf.handle.base,(uint32_t)l->buf.used,l->m,&GreaseLogger::LOGGER->defaultFilter);
//        			}
    			}
    			l->clear();
    			GreaseLogger::LOGGER->masterBufferAvail.add(l); // place buffer back in queue
    		}
    		break;
    	case TARGET_ROTATE_BUFFER:
		    {
		    	if(req.d > 0)
		    		t = GreaseLogger::LOGGER->getTarget(req.d);
		    	else
		    		t = GreaseLogger::LOGGER->defaultTarget;
//		    	assert(t);
		    	if(t) {
					DBG_OUT("TARGET_ROTATE_BUFFER [%d]", req.auxInt);
					t->flushAll(false);
		    	} else {
		    		DBG_OUT("No target!");
		    	}
//		    	t->flush(req.auxInt);
		    }
			break;
		case WRITE_TARGET_OVERFLOW:

			{
				// process a single log message - this is a large overflow message which is not in the masterBuffer
				singleLog *l = (singleLog *) req.aux;
				assert(l);
				if(GreaseLogger::LOGGER->sift(l->meta.m)) { // should always be true (checked in v8 thread)
					bool wrote = false;
					for(int i=0;i<GREASE_META_HASHLIST_CACHE_SIZE;i++) {
						FilterList *LIST = META_GET_LIST(l->meta.m,i);
						if (LIST && LIST->valid(l->meta.m.level)) {
							int n = 0;
							while(LIST->list[n].id != 0) {
								logTarget *t = NULL;
								ignore = false;
								if(((LIST->list[n].levelMask & l->meta.m.level) > 0) && !LIST->list[n]._disabled) { //if we have a Filter that matches...
									if(META_HAS_IGNORES(l->meta.m)) {
										int x = 0;
										while(x < MAX_IGNORE_LIST) {
											if(l->meta.ignore_list[x] == 0) { // 0 tells end of list
												break;
											}
											if(l->meta.ignore_list[x] == LIST->list[n].targetId) {
												ignore = true;
												break;
											}
											x++;
										}
									}
									if(LIST->list[n].targetId == 0) {
										//        						DBG_OUT("write out: %s",l->buf.handle.base);
										if(GreaseLogger::LOGGER->defaultTarget) {
											GreaseLogger::LOGGER->defaultTarget->flushAll();
											l->incRef();
											DBG_OUT("writeOverflow to defaultTarget Filter %p",  &LIST->list[n]);
											GreaseLogger::LOGGER->defaultTarget->writeOverflow(l, &LIST->list[n]);
										}
										wrote = true;
									} else
									if(!ignore && GreaseLogger::LOGGER->targets.find(LIST->list[n].targetId, t)) {
										//        						DBG_OUT("write out: %s",l->buf.handle.base);
										t->flushAll();
										l->incRef();
										DBG_OUT("writeOverflow to target %p Filter %p",  t, &LIST->list[n]);
										t->writeOverflow(l, &LIST->list[n]);
										wrote = true;
									} else {
										//        						DBG_OUT("Orphaned target id: %d\n",list->list[n].targetId);
									}
								} else {
								}
								n++;
							}
						}
					}
					if(!wrote && GreaseLogger::LOGGER->defaultTarget) {
						GreaseLogger::LOGGER->defaultTarget->flushAll();
						l->incRef();
						DBG_OUT("writeOverflow to defaultTarget and default Filter");
						GreaseLogger::LOGGER->defaultTarget->writeOverflow(l,&GreaseLogger::LOGGER->defaultFilter);
					}
				}
				l->decRef(); // we are done with the singleLog
			}


//			{
//		    	if(req.d > 0)
//		    		t = GreaseLogger::LOGGER->getTarget(req.d);
//		    	else
//		    		t = GreaseLogger::LOGGER->defaultTarget;
//		    	if(t) {
//			    	DBG_OUT("WRITE_TARGET_OVERFLOW");
//					t->flushAll();
//					singleLog *big = (singleLog *) req.aux;
//					t->writeAsync(big);
//		    	} else {
//		    		DBG_OUT("No target!");
//		    	}
////				delete big; // handled by callback
//
//	//			t->writeAsync() //overflowBuf
//			}

			break;
    	case INTERNAL_SHUTDOWN:
    	{
    		DBG_OUT("Got INTERNAL_SHUTDOWN");
    		// disable timer
    		uv_timer_stop(&LOGGER->flushTimer);
    		// disable all queues
    		uv_close((uv_handle_t *)&LOGGER->asyncExternalCommand, NULL);
    		uv_close((uv_handle_t *)&LOGGER->asyncInternalCommand, NULL);
    		// flush all queues
    		flushAllSync(true,true); // do rotation but no callbacks
    		// kill thread

    	}
    	}
	}
}

#if (UV_VERSION_MAJOR > 0)
	void GreaseLogger::handleExternalCmd(uv_async_t *handle) {
#else
	void GreaseLogger::handleExternalCmd(uv_async_t *handle, int status /*UNUSED*/) {
#endif
	GreaseLogger::nodeCmdReq req;
	while(LOGGER->nodeCmdQueue.removeMv(req)) {
		logTarget *t = NULL;
//		switch(req->c) {
//		case TARGET_ROTATE_BUFFER:
//		    {
//		    	t = GreaseLogger::LOGGER->getTarget(req->d);
//		    	t->flush(req->auxInt);
//		    }
//			break;
//
//		case WRITE_TARGET_OVERFLOW:
//			{
//				t = GreaseLogger::LOGGER->getTarget(req->d);
//				t->flushAll();
//	//			t->writeAsync() //overflowBuf
//			}
//
//			break;
//		}
	}

}

#if (UV_VERSION_MAJOR > 0)
void GreaseLogger::flushTimer_cb(uv_timer_t* handle) {
#else
void GreaseLogger::flushTimer_cb(uv_timer_t* handle, int status) {
#endif
	DBG_OUT("!! FLUSH TIMER !!");
	flushAll();
}

void GreaseLogger::startFlushTimer() {
	uv_timer_start(&flushTimer, LOGGER->flushTimer_cb, 2000, 500);
}

void GreaseLogger::stopFlushTimer() {
	uv_timer_stop(&flushTimer);
}

void GreaseLogger::mainThread(void *p) {
	GreaseLogger *SELF = (GreaseLogger *) p;
	uv_timer_init(SELF->loggerLoop, &SELF->flushTimer);
	uv_unref((uv_handle_t *)&LOGGER->flushTimer);
//	uv_timer_start(&SELF->flushTimer, flushTimer_cb, 2000, 500);
	uv_async_init(SELF->loggerLoop, &SELF->asyncInternalCommand, handleInternalCmd);
	uv_async_init(SELF->loggerLoop, &SELF->asyncExternalCommand, handleExternalCmd);

	uv_run(SELF->loggerLoop, UV_RUN_DEFAULT);

}

int GreaseLogger::log(const logMeta &f, const char *s, int len) { // does the work of logging
//	FilterList *list = NULL;
	if(len > GREASE_MAX_MESSAGE_SIZE)
		return GREASE_OVERFLOW;
	logMeta m = f;
	if(sift(m)) {
		return _log(m,s,len);
	} else
		return GREASE_OK;
}

int GreaseLogger::logP(logMeta *f, const char *s, int len) { // does the work of logging
//	FilterList *list = NULL;

	if(len > GREASE_MAX_MESSAGE_SIZE)
		return GREASE_OVERFLOW;
//	logMeta m = *f;
//
//	DBG_OUT("  log() have origin %d\n",f->origin);
//
	if(sift(*f)) {
		return _log((*f),s,len);
	} else
		return GREASE_OK;
}


/**
 * create a log entry for use across the network to Grease.
 * Memory layout: [PREAMBLE][Length (type RawLogLen)][logMeta][logdata - string - no null termination]
 * @param f a pointer to a meta data for logging. If NULL, it will be empty meta data
 * @param s string to log
 * @param len length of the passed in string
 * @param tobuf A raw buffer to store the log output ready for processing
 * @param len A pointer to an int. This will be read to know the existing length of the buffer, and then set
 * the size of the buffer that should be sent
 * @return returns GREASE_OK if successful, or GREASE_NO_BUFFER if the buffer is too small. If parameters are
 * invalid returns GREASE_INVALID_PARAMS
 */
//int logToRaw(logMeta *f, char *s, RawLogLen len, char *tobuf, int *buflen) {
//	if(!tobuf || *buflen < (GREASE_RAWBUF_MIN_SIZE + len))  // sanity check
//		return GREASE_NO_BUFFER;
//	int w = 0;
//
//	memcpy(tobuf,&__grease_preamble,SIZEOF_SINK_LOG_PREAMBLE);
//	w += SIZEOF_SINK_LOG_PREAMBLE;
//	RawLogLen _len =sizeof(logMeta) + sizeof(RawLogLen) + w;
//	memcpy(tobuf+w,&_len,sizeof(RawLogLen));
//	w += sizeof(RawLogLen);
//	if(f)
//		memcpy(tobuf+w,f,sizeof(logMeta));
//	else
//		memcpy(tobuf+w,&__noMetaData,sizeof(logMeta));
//	w += sizeof(logMeta);
//	if(s && len > 0) {
//		memcpy(tobuf+w,s,len);
//	}
//	*buflen = len;
//	return GREASE_OK;
//}



int GreaseLogger::logFromRaw(char *base, int len) {
	logMeta m;
	RawLogLen l;
	if(len >= GREASE_RAWBUF_MIN_SIZE) {

//		memcpy(&l,base+SIZEOF_SINK_LOG_PREAMBLE,sizeof(RawLogLen));
//		if(l > GREASE_MAX_MESSAGE_SIZE)  // don't let crazy memory blocks through.
//			return GREASE_OVERFLOW;
//		memcpy(&m,base+GREASE_CLIENT_HEADER_SIZE,sizeof(logMeta));
		int l = len - sizeof(logMeta);
		if( l > 0) {
			return logP((logMeta *)base,base+sizeof(logMeta),l);
		} else {
			ERROR_OUT("logRaw() malformed data. message too small.\n");
			return GREASE_OK;
		}
	} else
		return GREASE_NO_BUFFER;
}


int GreaseLogger::logSync(const logMeta &f, const char *s, int len) { // does the work of logging. now. will empty any buffers first.
	FilterList *list = NULL;
	logMeta m = f;
	if(sift(m)) {
		return _logSync(f,s,len);
	} else
		return GREASE_OK;
}
void GreaseLogger::flushAll(bool nocallbacks) { // flushes buffers. Synchronous
	if(LOGGER->defaultTarget) {
		LOGGER->defaultTarget->flushAll();
	} else
		ERROR_OUT("No default target!");

	logTarget **t; // shut down other targets.
	GreaseLogger::TargetTable::HashIterator iter(LOGGER->targets);
	while(!iter.atEnd()) {
		t = iter.data();
		(*t)->flushAll(true,nocallbacks);
		iter.getNext();
	}
}
void GreaseLogger::flushAllSync(bool rotate, bool nocallbacks) { // flushes buffers. Synchronous
	if(LOGGER->defaultTarget) {
		LOGGER->defaultTarget->flushAllSync();
	} else
		ERROR_OUT("No default target!");

	logTarget **t; // shut down other targets.
	GreaseLogger::TargetTable::HashIterator iter(LOGGER->targets);
	while(!iter.atEnd()) {
		t = iter.data();
		(*t)->flushAllSync(rotate,nocallbacks);
		iter.getNext();
	}
}

GreaseLogger::logTarget::~logTarget() {
	LFREE(_buffers);
}

GreaseLogger::logTarget::logTarget(int buffer_size, uint32_t id, GreaseLogger *o,
		targetReadyCB cb, delim_data _delim, target_start_info *readydata, uint32_t num_banks) :
		_disabled(false),
		readyCB(cb),                  // called when the target is ready or has failed to setup
		readyData(readydata),
		_buffers(NULL),
		logCallbackSet(false),
		logCallback(NULL),
		delim(std::move(_delim)),
		numBanks(num_banks),
		availBuffers(num_banks),
		writeOutBuffers(num_banks-1), // there should always be one buffer available for writingTo
		waitingOnCBBuffers(num_banks-1),
		err(), _log_fd(0),
		currentBuffer(NULL), bankSize(buffer_size), owner(o), myId(id), flags(0),
		timeFormat(),tagFormat(),originFormat(),levelFormat(),preFormat(),postFormat(),preMsgFormat()
{
	uv_mutex_init(&writeMutex);
	_buffers = (logBuf **) LMALLOC(sizeof(logBuf *)*numBanks);
	for (int n=0;n<numBanks;n++) {
		_buffers[n] = new logBuf(bankSize,n,delim.duplicate());
	}
	for (int n=1;n<numBanks;n++) {
		availBuffers.add(_buffers[n]);
	}
	currentBuffer = _buffers[0];
}


void GreaseLogger::targetReady(bool ready, _errcmn::err_ev &err, logTarget *t) {
	target_start_info *info = (target_start_info *) t->readyData;
	if(!info) {
		ERROR_OUT("Non fatal: target->readyData is NULL. this is wrong.");
		return;
	}
	info->targId = t->myId;
	DBG_OUT("@ target ready");

	if(ready) {
		if(t->myId == DEFAULT_TARGET) {
			t->owner->defaultTarget = t;
			if(info && info->cb) {
				info->cb(t->owner,err,info);
			}
		} else {
			t->owner->targets.addReplace(t->myId,t);
			if(info && info->cb) {
				info->cb(t->owner,err,info);
			}
		}
	} else {
		if(err.hasErr()) {
			ERROR_PERROR("Error on creating target!", err._errno);
		}
		ERROR_OUT("Failed to create target: %d\n", t->myId);
		if(info && info->cb) {
			info->targId = NULL;
			info->cb(t->owner,err,info);
		}
		// TODO shutdown?
		delete t;
	}
	if(t->myId == DEFAULT_TARGET && info) delete info;
}



void GreaseLogger::setupDefaultTarget(actionCB cb, target_start_info *i) {
	delim_data defaultdelim;
	defaultdelim.delim.sprintf("\n");
	i->cb = cb;
	this->Opts.lock();
	int size = this->Opts.bufferSize;
	this->Opts.unlock();

	// test - test for failure of ttyTarget >>
//	_errcmn::err_ev err;
//	err.setError(65001,"Test");
//	cb(this,err,i);
	// << end test
	new ttyTarget(size, DEFAULT_TARGET, this, targetReady,std::move(defaultdelim), i);
}


void GreaseLogger::start_logger_cb(GreaseLogger *l, _errcmn::err_ev &err, void *d) {
	GreaseLogger::target_start_info *info = (GreaseLogger::target_start_info *) d;
	if(info->needsAsyncQueue) {
		info->err = std::move(err);
		if(!l->targetCallbackQueue.add(info, TARGET_CALLBACK_QUEUE_WAIT)) {
			ERROR_OUT("Failed to queue callback. Buffer Full! Target: %d\n", info->targId);
		}
		uv_async_send(&l->asyncTargetCallback);
	} else {
		if(info->targetStartCB) {
			const unsigned argc = 1;
			if(err.hasErr()) {
				ERROR_OUT("Error on starting default target. greaseLog starting anyway...\n");
			}
#ifndef GREASE_LIB
			info->targetStartCB->Call(Nan::GetCurrentContext()->Global(),0,NULL);
#else
			info->targetStartCB(0,NULL);
#endif
//				else {
//					argv[0] = _errcmn::err_ev_to_JS(err, "Default target startup error: ")->ToObject();
//					info->targetStartCB->Call(Context::GetCurrent()->Global(),1,argv);
//				}
		}
	}
//	delete info;
}


void GreaseLogger::start(actionCB cb, target_start_info *data) {
	// FIXME use options for non default target
	setupDefaultTarget(cb,data);

	uv_thread_create(&logThreadId,GreaseLogger::mainThread,this);
}

#ifndef GREASE_LIB

NAN_METHOD(GreaseLogger::Start) {
	GreaseLogger *l = GreaseLogger::setupClass();
	GreaseLogger::target_start_info *startinfo = new GreaseLogger::target_start_info();

	if(info.Length() > 0 && info[0]->IsFunction()) {
		startinfo->targetStartCB = new Nan::Callback(Local<Function>::Cast(info[0]));
	}
	l->start(start_logger_cb, startinfo);
}


#endif

void GreaseLogger::start_target_cb(GreaseLogger *l, _errcmn::err_ev &err, void *d) {
	GreaseLogger::target_start_info *info = (GreaseLogger::target_start_info *) d;
	if(info->needsAsyncQueue) {
		info->err = std::move(err);
		if(!l->targetCallbackQueue.add(info, TARGET_CALLBACK_QUEUE_WAIT)) {
			ERROR_OUT("Failed to queue callback. Buffer Full! Target: %d\n", info->targId);
		}
		uv_async_send(&l->asyncTargetCallback);
	} else {
		if(info->targetStartCB) {
#ifndef GREASE_LIB
			const unsigned argc = 2;
			Local<Value> argv[argc];
			if(!err.hasErr()) {
				argv[0] = Nan::New((uint32_t) info->targId);
				info->targetStartCB->Call(Nan::GetCurrentContext()->Global(),1,argv);
			} else {
				argv[0] = Nan::New((int32_t)-1);
				argv[1] = _errcmn::err_ev_to_JS(err, "Target startup error: ")->ToObject();
				info->targetStartCB->Call(Nan::GetCurrentContext()->Global(),2,argv);
			}
#else
			if(!err.hasErr()) {
				uint32_t val = info->targId;
				info->targetStartCB(NULL,&val);
			} else {
				GreaseLibError *liberror = err.toGreaseLibError(NULL);
				info->targetStartCB(liberror,0);
			}
#endif
		}
		delete info;
	}
}

#ifndef GREASE_LIB

void GreaseLogger::callV8LogCallbacks(uv_async_t *h) {
	GreaseLogger *l = GreaseLogger::setupClass();
	GreaseLogger::logTarget::writeCBData data;
	while(l->callerLogCallbacks.removeMv(data)) {
		if(data.t->logCallback) {
			_doV8Callback(data);
		}
		l->unrefFromV8_inV8();
	}
}

void GreaseLogger::_doV8Callback(GreaseLogger::logTarget::writeCBData &data) {
	const unsigned argc = 2;
	Local<Value> argv[argc];
	Nan::HandleScope scope;
	if(data.b) {
		Nan::MaybeLocal<String> s = Nan::New( data.b->handle.base, (int) data.b->handle.len );
		Local<String> sconv;
		if(s.ToLocal(&sconv)) {
			argv[0] = sconv;
			argv[1] = Nan::New( data.t->myId );
			data.t->logCallback->Call(1,argv);
		} else {
			ERROR_OUT("Memory: Could not convert for log target's callback (1)");
		}
		data.t->finalizeV8Callback(data.b);
	} else if(data.overflow) {
		int l = data.overflow->totalSize();
		char *d = (char *) LMALLOC(l);
		if(d) {
			data.overflow->copyAllTo(d);
			Local<String> sconv;
			Nan::MaybeLocal<String> s = Nan::New( d, l );
			if(s.ToLocal(&sconv)) {
				argv[0] = sconv;
				argv[1] = Nan::New( data.t->myId );
				data.t->logCallback->Call(1,argv);
			} else {
				ERROR_OUT("Memory: Could not convert for log target's callback (2)");
			}
			LFREE(d);
		} else {
			ERROR_OUT("Error on LMALLOC. alloc was %d bytes.\n",l);
		}
		data.freeOverflow();
	}
}

void GreaseLogger::callTargetCallback(uv_async_t *h) {
	target_start_info *info = NULL;
	GreaseLogger *l = GreaseLogger::setupClass();
	Nan::HandleScope scope;
	while(l->targetCallbackQueue.remove(info)) {
		HEAVY_DBG_OUT("********* REMOVE targetCallbackQueue: %p\n", info);
		const unsigned argc = 2;
		Local<Value> argv[argc];
		if(info->targetStartCB) {
			if(!info->err.hasErr()) {
				argv[0] = Nan::New((int32_t) info->targId);
				info->targetStartCB->Call(1,argv);
			} else {
				argv[0] = Nan::New((int32_t) -1);
				argv[1] = _errcmn::err_ev_to_JS(info->err, "Target startup error: ")->ToObject();
				info->targetStartCB->Call(2,argv);
			}
		}
		delete info;
	}
	uv_unref((uv_handle_t *)&l->asyncTargetCallback);  // don't hold up event loop for this call back queue
}
#else

void GreaseLogger::callV8LogCallbacks(uv_async_t *h) {
	GreaseLogger *l = GreaseLogger::setupClass();
	GreaseLogger::logTarget::writeCBData data;
	while(l->callerLogCallbacks.removeMv(data)) {
		if(data.t->logCallback) {
			_doLibCallback(data);
		}
//		l->unrefFromV8_inV8();
	}
}


void GreaseLogger::callTargetCallback(uv_async_t *h) {
	target_start_info *info = NULL;
	GreaseLogger *l = GreaseLogger::setupClass();

	while(l->targetCallbackQueue.remove(info)) {
		HEAVY_DBG_OUT("********* REMOVE targetCallbackQueue: %p\n", info);
		const unsigned argc = 2;
		if(info->targetStartCB) {
			if(!info->err.hasErr()) {
				uint32_t val = info->targId;
				info->targetStartCB(NULL,&val);
			} else {
				GreaseLibError *liberror = info->err.toGreaseLibError(NULL);
				info->targetStartCB(liberror,0);
			}
		}
		delete info;
	}
	uv_unref((uv_handle_t *)&l->asyncTargetCallback);  // don't hold up event loop for this call back queue
}


void GreaseLogger::returnBufferToTarget(GreaseLibBuf *buf) {
	GreaseLogger *l = GreaseLogger::setupClass();

	TargetId id = (TargetId) buf->_id;
	GreaseLogger::logTarget *t;
	if(l->targets.find(id, t)) {
		logBuf *orig = (logBuf *) buf->_shadow;
		DBG_OUT("   ----Returning buffer to target----");
		t->finalizeV8Callback(orig);
	}
}

void GreaseLogger::_doLibCallback(GreaseLogger::logTarget::writeCBData &data) {
	if(data.b) {
//		Nan::MaybeLocal<String> s = Nan::New( data.b->handle.base, (int) data.b->handle.len );
//		Local<String> sconv;
//		if(s.ToLocal(&sconv)) {
//			argv[0] = sconv;
//			argv[1] = Nan::New( data.t->myId );
//		GreaseLibBuf buf;
//		GreaseLib_init_GreaseLibBuf(&buf);
		GreaseLibBuf *buf = _greaseLib_new_empty_GreaseLibBuf();
		buf->data = data.b->handle.base;
		buf->size = data.b->handle.len;
		buf->_id = (int) data.t->myId;
		buf->_shadow = data.b; // hide the original object here in void *
		data.t->logCallback(NULL,buf,data.t->myId);

// Below replaced, see returnBufferToTarget() above
//		data.t->finalizeV8Callback(data.b);
	} else if(data.overflow) {
		int l = data.overflow->totalSize();
//		char *d = (char *) LMALLOC(l);
		GreaseLibBuf *buf = GreaseLib_new_GreaseLibBuf((size_t) l);
		if(buf) {
			data.overflow->copyAllTo(buf->data);
//			Local<String> sconv;
//			Nan::MaybeLocal<String> s = Nan::New( d, l );
//			if(s.ToLocal(&sconv)) {
//				argv[0] = sconv;
//				argv[1] = Nan::New( data.t->myId );
			data.t->logCallback(NULL,buf, data.t->myId);
//			} else {
//				ERROR_OUT("Memory: Could not convert for log target's callback (2)");
//			}
//			LFREE(d);
		} else {
			ERROR_OUT("Error on LMALLOC. alloc was %d bytes.\n",l);
		}
		data.freeOverflow();
	}
}
#endif



#ifndef GREASE_LIB

NAN_METHOD(GreaseLogger::createPTS) {

	GreaseLogger *l = GreaseLogger::setupClass();
	Local<Function> cb;

	if(info.Length() < 1 || !info[0]->IsFunction()) {
		Nan::ThrowError("Malformed call to addTarget()");
		return;
	} else {
		cb = Local<Function>::Cast(info[0]);
	}

	char *slavename = NULL;
	const unsigned argc = 2;
	Local<Value> argv[argc];

	_errcmn::err_ev err;

	int fdm = open("/dev/ptmx", O_RDWR);  // open master

	if(fdm < 0) {
		err.setError(errno);
	} else {
		if(grantpt(fdm) != 0) {  // change permission of slave
			err.setError(errno);
		} else {
			if(unlockpt(fdm) != 0) {                     // unlock slave
				err.setError(errno);
			}
			slavename = ptsname(fdm);      	   // get name of slave
		}
	}

	if(err.hasErr() || !slavename) {
		argv[0] = _errcmn::err_ev_to_JS(err, "Error in creating pseudo-terminal: ")->ToObject();
		cb->Call(Nan::GetCurrentContext()->Global(),1,argv);
	} else {
		argv[0] = Nan::Null();

		Local<Object> obj = Nan::New<Object>();
		Nan::Set(obj,Nan::New("fd").ToLocalChecked(),Nan::New((int32_t)fdm));
		Nan::Set(obj,Nan::New("path").ToLocalChecked(),
				Nan::New(slavename,strlen(slavename)).ToLocalChecked());
		argv[1] = obj;
		cb->Call(Nan::GetCurrentContext()->Global(),2,argv);
	}

}

#endif

const char GreaseLogger::empty_label[] = "--";

#ifndef GREASE_LIB

/**
 * addTagLabel(id,label)
 * @param args id is a number, label a string
 *
 * @return v8::Undefined
 */
NAN_METHOD(GreaseLogger::AddTagLabel) {

	GreaseLogger *l = GreaseLogger::setupClass();

	if(info.Length() > 1 && info[0]->IsUint32() && info[1]->IsString()) {
		Nan::Utf8String v8str(info[1]->ToString());
		logLabel *label = logLabel::fromUTF8(v8str.operator *(),v8str.length());
		l->tagLabels.addReplace(info[0]->Uint32Value(),label);
	} else {
		return Nan::ThrowTypeError("addTagLabel: bad parameters");
	}

};


/**
 * addOriginLabel(id,label)
 * @param args id is a number, label a string
 *
 * @return v8::Undefined
 */
NAN_METHOD(GreaseLogger::AddOriginLabel) {

	GreaseLogger *l = GreaseLogger::setupClass();

	if(info.Length() > 1 && info[0]->IsUint32() && info[1]->IsString()) {
		Nan::Utf8String v8str(info[1]->ToString());
		logLabel *label = logLabel::fromUTF8(v8str.operator *(),v8str.length());
		l->originLabels.addReplace(info[0]->Uint32Value(),label);
	} else {
		return Nan::ThrowTypeError("addOriginLabel: bad parameters");
	}
};


/**
 * addOriginLabel(id,label)
 * @param args id is a number, label a string
 *
 * @return v8::Undefined
 */
NAN_METHOD(GreaseLogger::AddLevelLabel) {
	GreaseLogger *l = GreaseLogger::setupClass();

	if(info.Length() > 1 && info[0]->IsUint32() && info[1]->IsString()) {
		Nan::Utf8String v8str(info[1]->ToString());
		logLabel *label = logLabel::fromUTF8(v8str.operator *(),v8str.length());
		uint32_t v = info[0]->Uint32Value();
		l->levelLabels.addReplace(v,label);
	} else {
		return Nan::ThrowTypeError("addLevelLabel: bad parameters");
	}
};

NAN_METHOD(GreaseLogger::ModifyDefaultTarget) {
	if(info.Length() > 0 && info[0]->IsObject()){

		logTarget *targ = GreaseLogger::LOGGER->defaultTarget;
		Local<Object> jsTarg = info[0]->ToObject();

		Local<Value> jsDelim = jsTarg->Get(Nan::New("delim").ToLocalChecked());
		Local<Value> jsDelimOut = jsTarg->Get(Nan::New("delim_output").ToLocalChecked());
		// If either of these is set, the user means to change the default target
		Local<Value> jsTTY = jsTarg->Get(Nan::New("tty").ToLocalChecked());
		Local<Value> jsFile = jsTarg->Get(Nan::New("file").ToLocalChecked());

		bool makenewtarget = false;

		if(jsTTY->IsString()) {
			target_start_info *i = new target_start_info();

			i->cb = NULL;
			i->targId = DEFAULT_TARGET;


			delim_data defaultdelim;
			if(jsDelim->IsString()) {
				v8::String::Utf8Value v8str(jsDelim);
				defaultdelim.setDelim(v8str.operator *(),v8str.length());
			} else {
				defaultdelim.delim.sprintf("\n");
			}

			GreaseLogger::LOGGER->Opts.lock();
			int size = GreaseLogger::LOGGER->Opts.bufferSize;
			GreaseLogger::LOGGER->Opts.unlock();
			targ = new ttyTarget(size, DEFAULT_TARGET, GreaseLogger::LOGGER, targetReady,std::move(defaultdelim), i);

		} else if(jsFile->IsString()) {

			target_start_info *i = new target_start_info();

			delim_data defaultdelim;
			if(jsDelim->IsString()) {
				v8::String::Utf8Value v8str(jsDelim);
				defaultdelim.setDelim(v8str.operator *(),v8str.length());
			} else {
				defaultdelim.delim.sprintf("\n");
			}

			GreaseLogger::LOGGER->Opts.lock();
			int size = GreaseLogger::LOGGER->Opts.bufferSize;
			GreaseLogger::LOGGER->Opts.unlock();

			Local<Value> jsMode = jsTarg->Get(Nan::New("mode").ToLocalChecked());
			Local<Value> jsFlags = jsTarg->Get(Nan::New("flags").ToLocalChecked());
			Local<Value> jsRotate = jsTarg->Get(Nan::New("rotate").ToLocalChecked());
			int mode = DEFAULT_MODE_FILE_TARGET;
			int flags = DEFAULT_FLAGS_FILE_TARGET;
			if(jsMode->IsInt32()) {
				mode = jsMode->Int32Value();
			}
			if(jsFlags->IsInt32()) {
				flags = jsFlags->Int32Value();
			}


			i->cb = NULL;
			i->targId = DEFAULT_TARGET;

			Nan::Utf8String v8str(jsFile);
			if(info.Length() > 0 && info[1]->IsFunction())
				i->targetStartCB = new Nan::Callback(Local<Function>::Cast(info[1]));
			fileTarget::rotationOpts opts;
			if(jsRotate->IsObject()) {
				Local<Object> jsObj = jsRotate->ToObject();
				Local<Value> js_max_files = jsObj->Get(Nan::New("max_files").ToLocalChecked());
				Local<Value> js_max_file_size = jsObj->Get(Nan::New("max_file_size").ToLocalChecked());
				Local<Value> js_total_size = jsObj->Get(Nan::New("max_total_size").ToLocalChecked());
				Local<Value> js_on_start = jsObj->Get(Nan::New("rotate_on_start").ToLocalChecked());
				if(js_max_files->IsUint32()) {
					opts.max_files = js_max_files->Uint32Value(); opts.enabled = true;
				}
				if(js_max_file_size->IsUint32()) {
					opts.max_file_size = js_max_file_size->Uint32Value(); opts.enabled = true;
				}
				if(js_total_size->IsUint32()) {
					opts.max_total_size = js_total_size->Uint32Value(); opts.enabled = true;
				}
				if(js_on_start->IsBoolean()) {
					if(js_on_start->IsTrue())
						opts.rotate_on_start = true; opts.enabled = true;
				}
			}
			HEAVY_DBG_OUT("********* NEW target_start_info: %p\n", i);

			if(opts.enabled)
				targ = new fileTarget(size, DEFAULT_TARGET, GreaseLogger::LOGGER, flags, mode, v8str.operator *(), std::move(defaultdelim), i, targetReady, opts);
			else
				targ = new fileTarget(size, DEFAULT_TARGET, GreaseLogger::LOGGER, flags, mode, v8str.operator *(), std::move(defaultdelim), i, targetReady);

		} else {
			if(jsDelim->IsString() && targ) {
				v8::String::Utf8Value v8str(jsDelim);
				targ->delim.setDelim(v8str.operator *(),v8str.length());
			}
		}


		Local<Value> jsFormat = jsTarg->Get(Nan::New("format").ToLocalChecked());

		if(jsFormat->IsObject() && targ) {
			Local<Object> jsObj = jsFormat->ToObject();
			Local<Value> jsKey = jsObj->Get(Nan::New("pre").ToLocalChecked());
			if(jsKey->IsString()) {
				v8::String::Utf8Value v8str(jsKey);
				targ->setPreFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("time").ToLocalChecked());
			if(jsKey->IsString()) {
				v8::String::Utf8Value v8str(jsKey);
				targ->setTimeFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("level").ToLocalChecked());
			if(jsKey->IsString()) {
				v8::String::Utf8Value v8str(jsKey);
				targ->setLevelFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("tag").ToLocalChecked());
			if(jsKey->IsString()) {
				v8::String::Utf8Value v8str(jsKey);
				targ->setTagFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("origin").ToLocalChecked());
			if(jsKey->IsString()) {
				v8::String::Utf8Value v8str(jsKey);
				targ->setOriginFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("post").ToLocalChecked());
			if(jsKey->IsString()) {
				v8::String::Utf8Value v8str(jsKey);
				targ->setPostFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("pre_msg").ToLocalChecked());
			if(jsKey->IsString()) {
				Nan::Utf8String v8str(jsKey);
				targ->setPreMsgFormat(v8str.operator *(),v8str.length());
			}

		}
	}
}


/**
 * Allows caller to disable logging on a tag / origin / level combination
 */
NAN_METHOD(GreaseLogger::FilterOut) {
	// FIXME - it's implemented in the C-version in grease_lib.cc
}

/**
 * Allows caller to enable logging on a tag / origin / level combination. (The default)
 */
NAN_METHOD(GreaseLogger::FilterIn) {
	bool ok = false;
	if(info.Length() > 0 && info[0]->IsObject()) {
		Local<Object> jsObj = info[0]->ToObject();

		Local<Value> jsTag = jsObj->Get(Nan::New("tag").ToLocalChecked());
		Local<Value> jsOrigin = jsObj->Get(Nan::New("origin").ToLocalChecked());

		TagId tagId = 0;
		OriginId originId = 0;

		if(jsTag->IsUint32()) {
			tagId = (TagId) jsTag->Uint32Value();
			ok = true;
		}

		if(jsOrigin->IsUint32()) {
			originId = (OriginId) jsOrigin->Uint32Value();
			ok = true;
		}

		// FIXME ?? this does nothing!

	}
}



/**
 * addFilter(obj) where
 * obj = {
 *      // at least origin and/or tag must be present
 *      origin: 0x33,    // any positive number > 0
 *      tag: 0x25        // any positive number > 0,
 *      target: 3,       // mandatory
 *      mask:  0x4000000 // optional (default, show everything: 0xFFFFFFF),
 *      format: {        // optional (formatting settings)
 *      	pre: 'targ-pre>', // optional pre string
 *      	post: '<targ-post;
 *      }
 * }
 */
NAN_METHOD(GreaseLogger::AddFilter) {
	GreaseLogger *l = GreaseLogger::setupClass();
	FilterId id;
	TagId tagId = 0;
	OriginId originId = 0;
	TargetId targetId = 0;
	LevelMask mask = ALL_LEVELS;
	Handle<Value> ret = Nan::New(false);
	if(info.Length() > 0 && info[0]->IsObject()) {
		Local<Object> jsObj = info[0]->ToObject();

		bool ok = false;
		bool set_disable = false;
		Local<Value> jsTag = jsObj->Get(Nan::New("tag").ToLocalChecked());
		Local<Value> jsOrigin = jsObj->Get(Nan::New("origin").ToLocalChecked());
		Local<Value> jsTarget = jsObj->Get(Nan::New("target").ToLocalChecked());
		Local<Value> jsMask = jsObj->Get(Nan::New("mask").ToLocalChecked());
		Local<Value> jsDisable = jsObj->Get(Nan::New("disable").ToLocalChecked());

		if(jsTag->IsUint32()) {
			tagId = (TagId) jsTag->Uint32Value();
			ok = true;
		}
		if(jsOrigin->IsUint32()) {
			originId = (OriginId) jsOrigin->Uint32Value();
			ok = true;
		}
		if(jsMask->IsUint32()) {
			mask = jsMask->Uint32Value();
			ok = true;
		} else if (!jsMask->IsUndefined()) {
			Nan::ThrowTypeError("addFilter: bad parameters (mask)");
			return;
		}
		if(jsTarget->IsUint32()) {
			targetId = (OriginId) jsTarget->Uint32Value();
		} else {
//			ok = false;
			targetId = 0;
		}

		if((!jsDisable.IsEmpty() && !jsDisable->IsUndefined()) && jsDisable->IsBoolean()) {
			if(jsDisable->IsTrue()) {
				set_disable = true;
			}
		}

		if(!ok) {
			Nan::ThrowTypeError("addFilter: bad parameters");
			return;
		}

		logLabel *preFormat = NULL;
		logLabel *postFormat = NULL;
		logLabel *postFormatPreMsg = NULL;

		Local<Value> jsKey = jsObj->Get(Nan::New("pre").ToLocalChecked());
		if(jsKey->IsString()) {
			v8::String::Utf8Value v8str(jsKey);
			preFormat = logLabel::fromUTF8(v8str.operator *(),v8str.length());
		}
		jsKey = jsObj->Get(Nan::New("post").ToLocalChecked());
		if(jsKey->IsString()) {
			v8::String::Utf8Value v8str(jsKey);
			postFormat= logLabel::fromUTF8(v8str.operator *(),v8str.length());
		}
		jsKey = jsObj->Get(Nan::New("post_fmt_pre_msg").ToLocalChecked());
		if(jsKey->IsString()) {
			v8::String::Utf8Value v8str(jsKey);
			postFormatPreMsg= logLabel::fromUTF8(v8str.operator *(),v8str.length());
		}

		if(l->_addFilter(targetId,originId,tagId,mask,id,preFormat,postFormatPreMsg,postFormat))
			ret = Nan::New((uint32_t) id);
		else
			ret = Nan::New(false);

		if(set_disable) {
			Filter *found;
			if(l->_lookupFilter(originId,tagId,id,found)) {
				found->_disabled = true;
			}
		}
	} else {
		Nan::ThrowTypeError("addFilter: bad parameters");
		return;
	}
	info.GetReturnValue().Set(v8::Local<Value>(ret));
}

NAN_METHOD(GreaseLogger::RemoveFilter) {
}

NAN_METHOD(GreaseLogger::ModifyFilter) {
	GreaseLogger *l = GreaseLogger::setupClass();
	FilterId id;
	TagId tagId = 0;
	OriginId originId = 0;
	LevelMask mask = ALL_LEVELS;
	Handle<Value> ret = Nan::New(false);
	if(info.Length() > 0 && info[0]->IsObject()) {
		Local<Object> jsObj = info[0]->ToObject();

		bool ok = false;
		Local<Value> jsTag = jsObj->Get(Nan::New("tag").ToLocalChecked());
		Local<Value> jsOrigin = jsObj->Get(Nan::New("origin").ToLocalChecked());
		Local<Value> jsTarget = jsObj->Get(Nan::New("target").ToLocalChecked());
		Local<Value> jsMask = jsObj->Get(Nan::New("mask").ToLocalChecked());
		Local<Value> jsDisable = jsObj->Get(Nan::New("disable").ToLocalChecked());
		Local<Value> jsId = jsObj->Get(Nan::New("id").ToLocalChecked());

		if(jsTag->IsUint32()) {
			tagId = (TagId) jsTag->Uint32Value();
		}
		if(jsOrigin->IsUint32()) {
			originId = (OriginId) jsOrigin->Uint32Value();
		}
		if(jsMask->IsUint32()) {
			mask = jsMask->Uint32Value();
			Nan::ThrowTypeError("modifyFilter: bad parameters... can't modify mask");
			return;
		}
		if(jsId->IsUint32()) {
			id = (OriginId) jsId->Uint32Value();
			if(id > 0)
				ok = true;
		} else {
			ok = false;
		}

		if(!ok) {
			Nan::ThrowTypeError("modifyFilter: bad parameters");
			return;
		}

		Filter *found = NULL;
		if(ok && l->_lookupFilter(originId,tagId,id,found)) {
			if((!jsDisable.IsEmpty() && !jsDisable->IsUndefined()) && jsDisable->IsBoolean()) {
				if(jsDisable->IsTrue()) {
					found->_disabled = true;
				} else {
					found->_disabled = false;
				}
			}
			if(jsTarget->IsUint32()) {
				found->targetId = (TargetId) jsTarget->Uint32Value();
			}

			ret = Nan::New(true);
		} else
			ret = Nan::New(false);

	} else {
		Nan::ThrowTypeError("modifyFilter: bad parameters");
	}
	info.GetReturnValue().Set(v8::Local<Value>(ret));
}

/**
 * obj = {
 *    pipe: "/var/mysink"   // currently our only option is a named socket / pipe
 *    newConnCB: function()
 * }
 */
NAN_METHOD(GreaseLogger::AddSink) {
	GreaseLogger *l = GreaseLogger::setupClass();

	if(info.Length() > 0 && info[0]->IsObject()) {
		Local<Object> jsSink = info[0]->ToObject();
		Local<Value> isPipe = jsSink->Get(Nan::New("pipe").ToLocalChecked());
		Local<Value> isUnixDgram = jsSink->Get(Nan::New("unixDgram").ToLocalChecked());
		Local<Value> newConnCB = jsSink->Get(Nan::New("newConnCB").ToLocalChecked()); // called when a new connection is made on the callback

		if(isPipe->IsString()) {
			v8::String::Utf8Value v8str(isPipe);

			uv_mutex_lock(&l->nextIdMutex);
			SinkId id = l->nextSinkId++;
			uv_mutex_unlock(&l->nextIdMutex);

			PipeSink *sink = new PipeSink(l, v8str.operator *(), id, l->loggerLoop);
			Sink *base = dynamic_cast<Sink *>(sink);

			sink->bind();
			sink->start();

			l->sinks.addReplace(id,base);
		} else if(isUnixDgram->IsString()) {
			v8::String::Utf8Value v8str(isUnixDgram);

//			DBG_OUT("Opening socket unix dgram: %s\n",);

			uv_mutex_lock(&l->nextIdMutex);
			SinkId id = l->nextSinkId++;
			uv_mutex_unlock(&l->nextIdMutex);

			UnixDgramSink *sink = new UnixDgramSink(l, v8str.operator *(), id, l->loggerLoop);
			Sink *base = dynamic_cast<Sink *>(sink);

			sink->bind();
			sink->start();

			l->sinks.addReplace(id,base);
		} else {
			Nan::ThrowError("addSink: unsupported Sink type");
		}
	} else
		Nan::ThrowTypeError("addSink: bad parameters");
};


/**
 * addTarget(obj,finishCb) where obj is
 * obj = {
 *    tty: "/path"
 * }
 * obj = {
 *    file: "/path",
 *    append: true,   // default
 *    delim: "\n",    // optional. The delimitter between log entries. Any UTF8 string is fine
 *    format: {
 *       time: "[%ld]",
 *       level: "<%s>"
 *    }
 *    rotate: {
 *       max_files: 5,
 *       max_file_size:  100000,  // 100k
 *       max_total_size: 10000000,    // 10MB
 *       rotate_on_start: false   // default false
 *    }
 * }
 * obj = {
 *    callback: cb // some function
 * }
 *
 * finishCb(id) {
 *  // where id == the target's ID
 * }
 */
NAN_METHOD(GreaseLogger::AddTarget) {
	GreaseLogger *l = GreaseLogger::setupClass();
	uint32_t target = DEFAULT_TARGET;

//	iterate_plhdr();

	if(info.Length() > 1 && info[0]->IsObject()){
		Local<Object> jsTarg = info[0]->ToObject();
		Local<Value> isTty = jsTarg->Get(Nan::New("tty").ToLocalChecked());
//		Local<Value> isFile = jsTarg->Get(Nan::New("file").ToLocalChecked());
		Nan::MaybeLocal<Value> isFile = Nan::Get(jsTarg, Nan::New("file").ToLocalChecked());

		l->Opts.lock();
		int buffsize = l->Opts.bufferSize;
		l->Opts.unlock();
		Local<Value> jsBufSize = jsTarg->Get(Nan::New("bufferSize").ToLocalChecked());
		if(jsBufSize->IsInt32()) {
			buffsize = jsBufSize->Int32Value();
			if(buffsize < 0) {
				l->Opts.lock();
				buffsize = l->Opts.bufferSize;
				l->Opts.unlock();
			}
		}

		TargetId id;
		logTarget *targ = NULL;
		uv_mutex_lock(&l->nextIdMutex);
		id = l->nextTargetId++;
		uv_mutex_unlock(&l->nextIdMutex);

		Local<Value> jsDelim = jsTarg->Get(Nan::New("delim").ToLocalChecked());
		Local<Value> jsDelimOut = jsTarg->Get(Nan::New("delim_output").ToLocalChecked());
		Local<Value> jsCallback = jsTarg->Get(Nan::New("callback").ToLocalChecked());
		Local<Value> jsFormat = jsTarg->Get(Nan::New("format").ToLocalChecked());
		delim_data delims;

		if(jsDelim->IsString()) {
			Nan::Utf8String v8str(jsDelim);
			delims.delim.malloc(v8str.length());
			delims.delim.memcpy(v8str.operator *(),v8str.length());
		}

		if(jsDelimOut->IsString()) {
			Nan::Utf8String v8str(jsDelimOut);
			delims.delim_output.malloc(v8str.length());
			delims.delim_output.memcpy(v8str.operator *(),v8str.length());
		}

		if(isTty->IsString()) {
			v8::String::Utf8Value v8str(isTty);
//			obj->setIfName(v8str.operator *(),v8str.length());
			target_start_info *i = new target_start_info();
//			i->system_start_info = data;
			i->cb = start_target_cb;
			i->targId = id;
			if(info.Length() > 0 && info[1]->IsFunction()) {
				i->targetStartCB = new Nan::Callback(Local<Function>::Cast(info[1]));
				uv_ref((uv_handle_t*)&l->asyncTargetCallback); // hold up exit of the event loop until this call back is complete.
			}
			targ = new ttyTarget(buffsize, id, l, targetReady,  std::move(delims), i, v8str.operator *());
		} else if (!isTty.IsEmpty() && isTty->IsInt32()) {
			target_start_info *i = new target_start_info();
//			i->system_start_info = data;
			i->cb = start_target_cb;
			i->targId = id;
			if(info.Length() > 0 && info[1]->IsFunction()) {
				uv_ref((uv_handle_t*)&l->asyncTargetCallback); // hold up exit of the event loop until this call back is complete.
				i->targetStartCB = new Nan::Callback(Local<Function>::Cast(info[1]));
			}
			targ = ttyTarget::makeTTYTargetFromFD(isTty->ToInt32()->Value(), buffsize, id, l, targetReady,  std::move(delims), i);
		} else if (!isFile.IsEmpty()) {
			Local<Value> jsMode = jsTarg->Get(Nan::New("mode").ToLocalChecked());
			Local<Value> jsFlags = jsTarg->Get(Nan::New("flags").ToLocalChecked());
			Local<Value> jsRotate = jsTarg->Get(Nan::New("rotate").ToLocalChecked());
			int mode = DEFAULT_MODE_FILE_TARGET;
			int flags = DEFAULT_FLAGS_FILE_TARGET;
			if(jsMode->IsInt32()) {
				mode = jsMode->Int32Value();
			}
			if(jsFlags->IsInt32()) {
				flags = jsFlags->Int32Value();
			}

			Nan::Utf8String v8str(isFile.ToLocalChecked());
//			obj->setIfName(v8str.operator *(),v8str.length());
			target_start_info *i = new target_start_info();
//			i->system_start_info = data;
			if(info.Length() > 0 && info[1]->IsFunction()) {
				i->targetStartCB = new Nan::Callback(Local<Function>::Cast(info[1]));
				uv_ref((uv_handle_t*)&l->asyncTargetCallback); // hold up exit of the event loop until this call back is complete.
			}
			i->cb = start_target_cb;
			i->targId = id;
			fileTarget::rotationOpts opts;
			if(jsRotate->IsObject()) {
				Local<Object> jsObj = jsRotate->ToObject();
				Local<Value> js_max_files = jsObj->Get(Nan::New("max_files").ToLocalChecked());
				Local<Value> js_max_file_size = jsObj->Get(Nan::New("max_file_size").ToLocalChecked());
				Local<Value> js_total_size = jsObj->Get(Nan::New("max_total_size").ToLocalChecked());
				Local<Value> js_on_start = jsObj->Get(Nan::New("rotate_on_start").ToLocalChecked());
				if(js_max_files->IsUint32()) {
					opts.max_files = js_max_files->Uint32Value(); opts.enabled = true;
				}
				if(js_max_file_size->IsUint32()) {
					opts.max_file_size = js_max_file_size->Uint32Value(); opts.enabled = true;
				}
				if(js_total_size->IsUint32()) {
					opts.max_total_size = js_total_size->Uint32Value(); opts.enabled = true;
				}
				if(js_on_start->IsBoolean()) {
					if(js_on_start->IsTrue())
						opts.rotate_on_start = true; opts.enabled = true;
				}
			}
			HEAVY_DBG_OUT("********* NEW target_start_info: %p\n", i);

			if(opts.enabled)
				targ = new fileTarget(buffsize, id, l, flags, mode, v8str.operator *(), std::move(delims), i, targetReady, opts);
			else
				targ = new fileTarget(buffsize, id, l, flags, mode, v8str.operator *(), std::move(delims), i, targetReady);

		} else if(jsCallback->IsFunction()) { // if not those, bu callback is set, then just make a 'do nothing' target - but the callback will get used
			target_start_info *i = new target_start_info();
//			i->system_start_info = data;
			i->cb = start_target_cb;
			i->targId = id;
			if(info.Length() > 0 && info[1]->IsFunction()) {
				i->targetStartCB = new Nan::Callback(Local<Function>::Cast(info[1]));
				uv_ref((uv_handle_t*)&l->asyncTargetCallback); // hold up exit of the event loop until this call back is complete.
			}
//			int buffer_size, uint32_t id, GreaseLogger *o,
//							targetReadyCB cb, delim_data &_delim, target_start_info *readydata
			targ = new callbackTarget(buffsize, id, l, targetReady, std::move(delims), i);
			_errcmn::err_ev err;
			targ->readyCB(true,err,targ);
		} else {
			Nan::ThrowTypeError("Unknown target type passed into addTarget()");
		}

		if(jsCallback->IsFunction() && targ) {
			Local<Function> f = Local<Function>::Cast(jsCallback);
			targ->setCallback(f);
		}

		if(jsFormat->IsObject()) {
			Local<Object> jsObj = jsFormat->ToObject();
			Local<Value> jsKey = jsObj->Get(Nan::New("pre").ToLocalChecked());
			if(jsKey->IsString()) {
				Nan::Utf8String v8str(jsKey);
				targ->setPreFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("time").ToLocalChecked());
			if(jsKey->IsString()) {
				Nan::Utf8String v8str(jsKey);
				targ->setTimeFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("level").ToLocalChecked());
			if(jsKey->IsString()) {
				Nan::Utf8String v8str(jsKey);
				targ->setLevelFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("tag").ToLocalChecked());
			if(jsKey->IsString()) {
				Nan::Utf8String v8str(jsKey);
				targ->setTagFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("origin").ToLocalChecked());
			if(jsKey->IsString()) {
				Nan::Utf8String v8str(jsKey);
				targ->setOriginFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("post").ToLocalChecked());
			if(jsKey->IsString()) {
				Nan::Utf8String v8str(jsKey);
				targ->setPostFormat(v8str.operator *(),v8str.length());
			}
			jsKey = jsObj->Get(Nan::New("pre_msg").ToLocalChecked());
			if(jsKey->IsString()) {
				Nan::Utf8String v8str(jsKey);
				targ->setPreMsgFormat(v8str.operator *(),v8str.length());
			}
		}
	} else {
		Nan::ThrowTypeError("Malformed call to addTarget()");
	}
}


/**
 * logstring and level manadatory
 * all else optional
 * log(message(number), level{number},tag{number},origin{number},extras{object})
 *
 * extras = {
 *    .ignores = {number|array}
 * }
 * @method log
 *
 */
NAN_METHOD(GreaseLogger::Log) {
	static extra_logMeta meta; // static - this call is single threaded from node.
	ZERO_LOGMETA(meta.m);
	uint32_t target = DEFAULT_TARGET;
	if(info.Length() > 1 && info[0]->IsString() && info[1]->IsInt32()){
		GreaseLogger *l = GreaseLogger::setupClass();
		v8::String::Utf8Value v8str(info[0]->ToString());
		meta.m.level = (uint32_t) info[1]->Int32Value(); // level

		if(info.Length() > 2 && info[2]->IsInt32()) // tag
			meta.m.tag = (uint32_t) info[2]->Int32Value();
		else
			meta.m.tag = 0;

		if(info.Length() > 3 && info[3]->IsInt32()) // origin
			meta.m.origin = (uint32_t) info[3]->Int32Value();
		else
			meta.m.origin = 0;

		if(l->sift(meta.m)) {
			if(info.Length() > 4 && info[4]->IsObject()) {
				Local<Object> jsObj = info[4]->ToObject();
				Local<Value> val = jsObj->Get(Nan::New("ignores").ToLocalChecked());
				if(val->IsArray()) {
					Local<Array> array = v8::Local<v8::Array>::Cast(val);
					uint32_t i = 0;
					for (i=0 ; i < array->Length() ; ++i) {
					  const Local<Value> value = array->Get(i);
					  if(i >= MAX_IGNORE_LIST) {
						  break;
					  } else {
						  meta.ignore_list[i] = value->Uint32Value();
					  }
					}
					meta.ignore_list[i] = 0;
				} else if(val->IsUint32()) {
					meta.ignore_list[0] = val->Uint32Value();
					meta.ignore_list[1] = 0;
				}
				meta.m.extras = 1;
			}
			FilterList *list = NULL;
			l->_log(meta.m,v8str.operator *(),v8str.length());
		}
	}
}

NAN_METHOD(GreaseLogger::LogSync) {
	static logMeta meta; // static - this call is single threaded from node.
	uint32_t target = DEFAULT_TARGET;
	if(info.Length() > 1 && info[0]->IsString() && info[1]->IsInt32()){
		GreaseLogger *l = GreaseLogger::setupClass();
		v8::String::Utf8Value v8str(info[0]->ToString());
		meta.level = (uint32_t) info[1]->Int32Value();

		if(info.Length() > 2 && info[2]->IsInt32()) // tag
			meta.tag = (uint32_t) info[2]->Int32Value();
		else
			meta.tag = 0;

		if(info.Length() > 3 && info[3]->IsInt32()) // tag
			meta.origin = (uint32_t) info[3]->Int32Value();
		else
			meta.origin = 0;

		FilterList *list = NULL;
		if(l->sift(meta)) {
			l->_logSync(meta,v8str.operator *(),v8str.length());
		}
	}
}


NAN_METHOD(GreaseLogger::DisableTarget) {
	GreaseLogger *l = GreaseLogger::setupClass();
	if(info.Length() > 0 && info[0]->IsUint32()){
		uint32_t tnum = (uint32_t) info[0]->Uint32Value();
		logTarget *t = NULL;
		if(l->targets.find(tnum,t)) {
			t->disableWrites(true);
			info.GetReturnValue().Set(Nan::True());
		} else {
			ERROR_OUT("grease.disableTarget(%d) Bad parameters.\n", tnum);
			info.GetReturnValue().Set(Nan::False());
		}
	} else {
		ERROR_OUT("grease.disableTarget() Bad parameters.\n");
		info.GetReturnValue().Set(Nan::False());
	}
}

NAN_METHOD(GreaseLogger::EnableTarget) {
	GreaseLogger *l = GreaseLogger::setupClass();
	if(info.Length() > 0 && info[0]->IsUint32()){
		uint32_t tnum = (uint32_t) info[0]->Uint32Value();
		logTarget *t = NULL;
		if(l->targets.find(tnum,t)) {
			t->disableWrites(false);
			info.GetReturnValue().Set(Nan::True());
		} else {
			ERROR_OUT("grease.enableTarget(%d) Unknown target.\n", tnum);
			info.GetReturnValue().Set(Nan::False());
		}
	} else {
		ERROR_OUT("grease.enableTarget() Bad parameters.\n");
		info.GetReturnValue().Set(Nan::False());
	}
}


NAN_METHOD(GreaseLogger::Flush) {
	GreaseLogger *l = GreaseLogger::setupClass();
	if(info.Length() > 1 && info[0]->IsUint32()){
		uint32_t tnum = (uint32_t) info[0]->Uint32Value();
		GreaseLogger::logTarget *t = NULL;
		if(l->targets.find(tnum,t)) {
			t->flushAll();
		}
	} else {
		l->flushAll();
	}

}

#endif



// Exmaple kernel log entry:
// <2>[1921219.107969] CPU1: Core temperature above threshold, cpu clock throttled (total events = 6581804)

//		KERN_EMERG	<0>	Emergency messages (precede a crash)
//		KERN_ALERT	<1>	Error requiring immediate attention
//		KERN_CRIT	<2>	Critical error (hardware or software)
//		KERN_ERR	<3>	Error conditions (common in drivers)
//		KERN_WARNING<4>	Warning conditions (could lead to errors)
//		KERN_NOTICE	<5>	Not an error but a significant condition
//		KERN_INFO	<6>	Informational message
//		KERN_DEBUG	<7>	Used only for debug messages
//		KERN_DEFAULT<d>	Default kernel logging level
//		KERN_CONT	<c>	Continuation of a log line (avoid adding new time stamp)

//
/**
 * A utility function for intenral use, which parses a kernel log, coming out of the kernel ring-buffer or by /dev/kmsg
 *
 * The best source to understand the ring buffer and how to read it is dmesg.c source, which is in the linux-utils package.
 *
 * return true if there is an entry to send to the logger, false if not
 */
bool GreaseLogger::parse_single_klog_to_singleLog(char *start, int &remain, klog_parse_state &begin_state, singleLog *entry, char *&moved) {
	klog_parse_state state = begin_state;
	char *look = start;
	char *cap = NULL;
	int klog_level;

	while(remain > 0 && state != INVALID && state != END_LOG) {
		switch(state) {
			case LEVEL_BEGIN:
				if (*look == '<') {
					state = IN_LEVEL;
					cap = look+1;
				} else {
					state = INVALID;
				}
				break;
			case IN_LEVEL:
				if (*look == '>') {
					*look = '\0';  // make it NULL, so we can capture the string

					if(look == cap + 1) { // there should be only 1 digit between < >
						if(*cap == 'c') {
							// this is a continuation line. Special case.
							// TODO
							state = CONT_BODY_BEGIN;
						} else
						if(*cap == 'd') {
							entry->meta.m.level = GREASE_KLOG_DEFAULT_LEVEL;
							state = TIME_STAMP_BEGIN;
						} else {
							klog_level = *cap - '0';
							if (klog_level < 8 && klog_level >= 0) {
								// valid level
								entry->meta.m.level = GREASE_KLOGLEVEL_TO_LEVEL_MAP[klog_level];
								state = TIME_STAMP_BEGIN;
							} else {
								state = INVALID;
							}
						}
					} else { // else, there was nothing in LEVEL brackets
						state = INVALID;
					}
				}
				break;
			case TIME_STAMP_BEGIN:
				if (*look == '[') state = IN_TIME_STAMP; else state = INVALID;
				break;
			case IN_TIME_STAMP:
				if (*look == ']') state = BODY_BEGIN;
				else {
					if (!isdigit(*look) && *look != '.' && *look != ' ') {
						state = INVALID;
					}
				}
				break;
			case BODY_BEGIN:
				cap = look + 1;
				if (*look != '\n') {
					state = IN_BODY;
				} else {
					state = END_LOG;
				}
				break;
			case IN_BODY:
				if (*look == '\n') {
					state = END_LOG;
				}
		}
		remain--;
		look++;
		moved = look;
	}
	if(state == END_LOG) {
		begin_state = LEVEL_BEGIN; // on the next call, move to next log entry
		entry->meta.m.tag = GREASE_TAG_KERNEL;
		entry->meta.m.origin = 0;
		if((look - cap) > 0) {
			// actually log stuff with some length
			if((look - cap - 1) > entry->buf.handle.len) {
				// its too big... (should probably never happen)
				// just truncate it
				::memcpy((void *)entry->buf.handle.base,cap,(int) entry->buf.handle.len);
				entry->buf.used = (int) entry->buf.handle.len;
			} else {
				::memcpy((void *)entry->buf.handle.base,cap,(int) (look-cap-1));
				entry->buf.used = (int)(look-cap-1);
			}
		} else {
			return false;
		}
		// we only return true if this loggable
		return true;
	} else {
		begin_state = state;
		return false;
	}

}

int fast_ascii_to_int(char *s, int n) {
    int ret = 0;
    int e = 1;
    while(n > 0) {
        ret += (s[n-1] - '0') * e;
        e = e * 10;
        n--;
    }
    return ret;
}


bool GreaseLogger::parse_single_syslog_to_singleLog(char *start, int &remain, syslog_parse_state &begin_state, singleLog *entry) { // , char *&moved) {
    syslog_parse_state state = begin_state;
    char *look = start;
    char *cap = NULL;
    int klog_level;
    char log_fac_buf[5];
    int log_fac_buf_n = 0;
    int fac_pri = 0;
    int pri; int fac;
    char *msg_mark = NULL;

// EXAMPLE LOG INPUT:<30>Feb  1 15:40:05 nm-dispatcher: req:2 'up' [wlan1]: s
//                   <30>Jan 31 15:40:05 nm-dispatcher: req:2 'up' [wlan1]: s

    while(remain > 0 && state != SYSLOG_INVALID && state != SYSLOG_END_LOG) {
        switch(state) {
            case SYSLOG_BEGIN:
                if (*look == '<') {
                    state = SYSLOG_IN_FAC;                    
                }
                break;
            case SYSLOG_IN_FAC:
                if(log_fac_buf_n > 4) {
                    state = SYSLOG_INVALID;
                    break;
                }
                if (*look >= '0' && *look <= '9') {
                    log_fac_buf[log_fac_buf_n] = *look;
                    log_fac_buf_n++;
                    break;
                }
                if (*look == '>') {
                    // not needed -> log_fac_buf[log_fac_buf_n] = '\0';
                    // ok - we have a log_fac number, parse it up
                    fac_pri = fast_ascii_to_int(log_fac_buf, log_fac_buf_n);

                    pri = LOG_PRI(fac_pri);
                    if( pri < 8) {
                        entry->meta.m.level = GREASE_SYSLOGPRI_TO_LEVEL_MAP[pri];
                    } else {
                        entry->meta.m.level = GREASE_LEVEL_LOG;
                        DBG_OUT("out of bounds LOG_PRI ");
                    }
                    fac = LOG_FAC(fac_pri);
//                                  DBG_OUT("fac_pri %d  %d\n",fac_pri,fac);
                    if( fac < sizeof(GREASE_SYSLOGFAC_TO_TAG_MAP)) {
                        entry->meta.m.tag = GREASE_SYSLOGFAC_TO_TAG_MAP[fac];
                    } else {
                        entry->meta.m.tag = GREASE_TAG_SYSLOG;
                        DBG_OUT("out of bounds LOG_FAC %d",fac);
                    }

                    // now the date
                    state = SYSLOG_IN_DATE_MO;
                }
                break;
            case SYSLOG_IN_DATE_MO:
                // let's cheat. We don't care about the date. So skip past it
                // the date-time stamp is 16 char, including last space
                if (remain > 16) {
                    look+= 16;
                    remain = remain - 16;
                    state = SYSLOG_IN_MESSAGE;
                    continue;
                } else {
                    state = SYSLOG_INVALID;
                    break;
                }
            case SYSLOG_IN_DATE_DAY:
            case SYSLOG_IN_DATE_TIME:
                
            case SYSLOG_IN_MESSAGE:
                msg_mark = look;
//                moved = look;
                // assume this is only one log message
                state = SYSLOG_END_LOG;
                continue;          
        }
        remain--;
        look++;
//        moved = look;
    }
    if(state == SYSLOG_END_LOG) {
        begin_state = SYSLOG_BEGIN; // on the next call, move to next log entry
        entry->meta.m.origin = 0;
        if(remain > 0) {
            // actually log stuff with some length
            if(remain > entry->buf.handle.len) {
                // its too big... (should probably never happen)
                // just truncate it
                ::memcpy((void *)entry->buf.handle.base,msg_mark,(int) entry->buf.handle.len);
                entry->buf.used = (int) entry->buf.handle.len;
            } else {
                ::memcpy((void *)entry->buf.handle.base,msg_mark,(int) remain);
                entry->buf.used = (int)remain;
            }
        } else {
            return false;
        }
        // we only return true if this loggable
        return true;
    } else {
        begin_state = state;
        return false;
    }
}

/**
 * Messages from the /dev/kmsg are a bit simpler:
 *
 * Example:
 * LEVEL,TIME,TIME2,?;BODY\n
 * 6,2000,8486026862,-;thinkpad_acpi: EC reports that Thermal Table has changed
 */
bool GreaseLogger::parse_single_devklog_to_singleLog(char *start, int &remain, klog_parse_state &begin_state, singleLog *entry, char *&moved) {
    klog_parse_state state = begin_state;
    char *look = start;
    char *cap = NULL;
    int klog_level;

    while(remain > 0 && state != INVALID && state != END_LOG) {
        switch(state) {
            case LEVEL_BEGIN:
                cap = look;
                state = IN_LEVEL;
                break;
            case IN_LEVEL:
                if (*look == ',') {
                    *look = '\0';  // make it NULL, so we can capture the string

                    if(look == cap + 1 || look == cap + 2) { // there should be only 1 or 2, digit
                        if(*cap == 'c') {
                            // this is a continuation line. Special case.
                            // TODO
                            state = CONT_BODY_BEGIN;
                        } else
                        if(*cap == 'd') {
                            entry->meta.m.level = GREASE_KLOG_DEFAULT_LEVEL;
                            state = TIME_STAMP_BEGIN;
                        } else {
                            if(look == cap + 1)
                                klog_level = *cap - '0';
                            else if (*cap == '1')
                                klog_level = *(cap+1) - '0' + 10;
                            else
                                klog_level = 20;

                            if (klog_level < 20 && klog_level >= 0) {
                                // valid level
                                entry->meta.m.level = GREASE_KLOGLEVEL_TO_LEVEL_MAP[klog_level];
                                state = TIME_STAMP_BEGIN;
                            } else {
                                // ?? dunno, something new, move on - use default
                                entry->meta.m.level = GREASE_KLOG_DEFAULT_LEVEL;
                                state = TIME_STAMP_BEGIN;
                            }
                        }
                    } else { // else, there was nothing confusing, skip to body - use default level
                        if (*look == ';') { // in this case,
                            entry->meta.m.level = GREASE_KLOG_DEFAULT_LEVEL;
                            state = BODY_BEGIN;
                        }

//                      state = INVALID;
                    }
                }
                break;
            case TIME_STAMP_BEGIN:
                if(*look == ';') {
                    state = BODY_BEGIN;
                }
                break;
            case BODY_BEGIN:
                cap = look;
                if(remain == 1) state = END_LOG;
                else state = IN_BODY;
                break;
            case IN_BODY:
                if(remain == 1) state = END_LOG;
                break;
//              if(remain == 1) {
//
//              }
//              cap = look;
//              if (*look == '\n') {
//                  state = END_LOG;
//              }
//              break;
        }
        remain--;
        look++;
        moved = look;
    }
    if(state == END_LOG) {
        begin_state = LEVEL_BEGIN; // on the next call, move to next log entry
        entry->meta.m.tag = GREASE_TAG_KERNEL;
        entry->meta.m.origin = 0;
        if((look - cap) > 0) {
            // actually log stuff with some length
            if((look - cap - 1) > entry->buf.handle.len) {
                // its too big... (should probably never happen)
                // just truncate it
                ::memcpy((void *)entry->buf.handle.base,cap,(int) entry->buf.handle.len);
                entry->buf.used = (int) entry->buf.handle.len;
            } else {
                ::memcpy((void *)entry->buf.handle.base,cap,(int) (look-cap-1));
                entry->buf.used = (int)(look-cap-1);
            }
        } else {
            return false;
        }
        // we only return true if this loggable
        return true;
    } else {
        begin_state = state;
        return false;
    }

}

int GreaseLogger::_grabInLogBuffer(singleLog* &buf) {
	if(masterBufferAvail.remove(buf)){
		return GREASE_OK;
	} else {
		return GREASE_OVERFLOW;
	}
}

int GreaseLogger::_returnBuffer(singleLog *buf) {
	buf->clear();
	masterBufferAvail.add(buf);
}

int GreaseLogger::_submitBuffer(singleLog *buf) {
	if(buf->buf.used < 1) {
		// if for some reason the caller does not put anything in the buffer,
		// then just return it
		buf->clear();
		masterBufferAvail.add(buf);
		return GREASE_OK;
	}
	internalCmdReq req(NEW_LOG);
	req.aux = buf;
	if(internalCmdQueue.addMvIfRoom(req))
		uv_async_send(&asyncInternalCommand);
	else {
		buf->clear();
		masterBufferAvail.add(buf);
		ERROR_OUT("internalCmdQueue is out of space!! Dropping. (_submitBuffer) \n");
	}
}

int GreaseLogger::_log( const logMeta &meta, const char *s, int len) { // internal log cmd
//	HEAVY_DBG_OUT("out len: %d\n",len);
//	DBG_OUT("meta.level %x",meta.level);
//	if(len > GREASE_MAX_MESSAGE_SIZE)
//		return GREASE_OVERFLOW;
	singleLog *l = NULL;
	if(len > Opts.bufferSize) {
		internalCmdReq req(WRITE_TARGET_OVERFLOW);
		int _len = len;
		if(len > MAX_LOG_MESSAGE_SIZE) _len = MAX_LOG_MESSAGE_SIZE;
		l = new singleLog(_len);
		l->buf.memcpy(s,len,"[!! OVERFLOW ENDING]");
		if(META_HAS_IGNORES(meta)) {
			extra_logMeta *extra = META_WITH_EXTRAS(meta);
			memcpy(&l->meta, extra, sizeof(extra_logMeta));
		} else {
			l->meta.m = meta;
		}
		req.aux = l;
		if(internalCmdQueue.addMvIfRoom(req))
			uv_async_send(&asyncInternalCommand);
		else {
			ERROR_OUT("internalCmdQueue is out of space!! Dropping. (_log) \n");
			delete l;
		}
	} else {
		if(masterBufferAvail.remove(l)) {
			l->buf.memcpy(s,len);
			if(META_HAS_IGNORES(meta)) {
				extra_logMeta *extra = META_WITH_EXTRAS(meta);
				memcpy(&l->meta, extra, sizeof(extra_logMeta));
			} else {
				l->meta.m = meta;
			}

			internalCmdReq req(NEW_LOG);
			req.aux = l;
			if(internalCmdQueue.addMvIfRoom(req))
				uv_async_send(&asyncInternalCommand);
			else {
				l->clear();
				masterBufferAvail.add(l);
				ERROR_OUT("internalCmdQueue is out of space!! Dropping. (_log) \n");
			}
		} else {
			ERROR_OUT("masterBuffer is out of space!! Dropping. (%d)\n", masterBufferAvail.remaining());
		}
	}

//	if(!list) {
//		defaultTarget->write(s,len,meta);
//	} else if (list->valid(meta.level)) {
//		int n = 0;
//		while(list->list[n].id != 0) {
//			logTarget *t = NULL;
//			if((list->list[n].levelMask & meta.level) && targets.find(list->list[n].targetId, t)) {
//				t->write(s,len,meta);
//			} else {
//				ERROR_OUT("Orphaned target id: %d\n",list->list[n].targetId);
//			}
//			n++;
//		}
//	} else {  // write out to default target if the list does not apply to this level
//		defaultTarget->write(s,len,meta);
//	}
	return GREASE_OK;
}

int GreaseLogger::_logSync( const logMeta &meta, const char *s, int len) { // internal log cmd
	if(len > GREASE_MAX_MESSAGE_SIZE)
		return GREASE_OVERFLOW;
// FIXME FIXME FIXME
	//	if(!list) {
//		defaultTarget->writeSync(s,len,meta);
//	} else {
//		int n = 0;
//		while(list->list[n].id != 0) {
//			logTarget *t = NULL;
//			if(targets.find(list->list[n].targetId, t)) {
//				t->writeSync(s,len,meta);
//			} else {
//				ERROR_OUT("Orphaned target id: %d\n",list->list[n].targetId);
//			}
//			n++;
//		}
//	}
	return GREASE_OK;
}

#ifndef GREASE_LIB

void GreaseLogger::Init(v8::Local<v8::Object> exports) {
	Nan::HandleScope scope;

	Local<FunctionTemplate> tpl = Nan::New<v8::FunctionTemplate>(New);
	tpl->SetClassName(Nan::New("GreaseLogger").ToLocalChecked());
	tpl->InstanceTemplate()->SetInternalFieldCount(1);

	tpl->PrototypeTemplate()->SetInternalFieldCount(2);

//	if(args.Length() > 0) {
//		if(args[0]->IsObject()) {
//			Local<Object> base = args[0]->ToObject();
//			Local<Array> keys = base->GetPropertyNames();
//			for(int n=0;n<keys->Length();n++) {
//				Local<String> keyname = keys->Get(n)->ToString();
//				tpl->InstanceTemplate()->Set(keyname, base->Get(keyname));
//			}
//		}
//	}

	Nan::SetPrototypeMethod(tpl,"start",Start);
	Nan::SetPrototypeMethod(tpl,"flush",Flush);
	Nan::SetPrototypeMethod(tpl,"logSync",LogSync);
	Nan::SetPrototypeMethod(tpl,"log",Log);
	Nan::SetPrototypeMethod(tpl,"enableTarget",EnableTarget);
	Nan::SetPrototypeMethod(tpl,"disableTarget",DisableTarget);
	Nan::SetPrototypeMethod(tpl,"createPTS",createPTS);
	Nan::SetPrototypeMethod(tpl,"setGlobalOpts",SetGlobalOpts);
	Nan::SetPrototypeMethod(tpl,"addTagLabel",AddTagLabel);
	Nan::SetPrototypeMethod(tpl,"addOriginLabel",AddOriginLabel);
	Nan::SetPrototypeMethod(tpl,"addLevelLabel",AddLevelLabel);
	Nan::SetPrototypeMethod(tpl,"modifyDefaultTarget",ModifyDefaultTarget);
	Nan::SetPrototypeMethod(tpl,"addFilter",AddFilter);
	Nan::SetPrototypeMethod(tpl,"removeFilter",RemoveFilter);
	Nan::SetPrototypeMethod(tpl,"modifyFilter",ModifyFilter);
	Nan::SetPrototypeMethod(tpl,"addTarget",AddTarget);
	Nan::SetPrototypeMethod(tpl,"addSink",AddSink);

//	tpl->InstanceTemplate()->Set(String::NewSymbol("start"), FunctionTemplate::New(Start)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("flush"), FunctionTemplate::New(Flush)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("logSync"), FunctionTemplate::New(LogSync)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("log"), FunctionTemplate::New(Log)->GetFunction());
//
//	tpl->InstanceTemplate()->Set(String::NewSymbol("enableTarget"), FunctionTemplate::New(EnableTarget)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("disableTarget"), FunctionTemplate::New(DisableTarget)->GetFunction());
//
//	tpl->InstanceTemplate()->Set(String::NewSymbol("createPTS"), FunctionTemplate::New(createPTS)->GetFunction());
//
//	tpl->InstanceTemplate()->Set(String::NewSymbol("setGlobalOpts"), FunctionTemplate::New(SetGlobalOpts)->GetFunction());
//
//	tpl->InstanceTemplate()->Set(String::NewSymbol("addTagLabel"), FunctionTemplate::New(AddTagLabel)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("addOriginLabel"), FunctionTemplate::New(AddOriginLabel)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("addLevelLabel"), FunctionTemplate::New(AddLevelLabel)->GetFunction());
//
//	tpl->InstanceTemplate()->Set(String::NewSymbol("modifyDefaultTarget"), FunctionTemplate::New(ModifyDefaultTarget)->GetFunction());
//
//	tpl->InstanceTemplate()->Set(String::NewSymbol("addFilter"), FunctionTemplate::New(AddFilter)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("removeFilter"), FunctionTemplate::New(RemoveFilter)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("modifyFilter"), FunctionTemplate::New(ModifyFilter)->GetFunction());
//
//	tpl->InstanceTemplate()->Set(String::NewSymbol("addTarget"), FunctionTemplate::New(AddTarget)->GetFunction());
//	tpl->InstanceTemplate()->Set(String::NewSymbol("addSink"), FunctionTemplate::New(AddSink)->GetFunction());

//	tpl->InstanceTemplate()->SetAccessor(String::New("lastErrorStr"), GetLastErrorStr, SetLastErrorStr);
//	tpl->InstanceTemplate()->SetAccessor(String::New("_readChunkSize"), GetReadChunkSize, SetReadChunkSize);

	constructor.Reset(tpl->GetFunction());
	exports->Set(Nan::New("Logger").ToLocalChecked(), tpl->GetFunction());
}


/** TunInterface(opts)
 * opts {
 * 	    ifname: "tun77"
 * }
 * @param args
 * @return
 **/
NAN_METHOD(GreaseLogger::New) {

	GreaseLogger* obj = NULL;

	if (info.IsConstructCall()) {
	    // Invoked as constructor: `new MyObject(...)`
		if(info.Length() > 0) {
			if(!info[0]->IsObject()) {
				Nan::ThrowTypeError("Improper first arg to TunInterface cstor. Must be an object.");
			}

			obj = GreaseLogger::setupClass();
		} else {
			obj = GreaseLogger::setupClass();
		}

		obj->Wrap(info.This());
		info.GetReturnValue().Set(info.This());
	} else {
	    // Invoked as plain function `MyObject(...)`, turn into construct call.
	    const int argc = 1;
	    Local<Value> argv[argc] = { info[0] };
	    v8::Local<v8::Function> cons = Nan::New<v8::Function>(constructor);
	    info.GetReturnValue().Set(cons->NewInstance(argc, argv));
	  }

}

#endif

//Handle<Value> GreaseLogger::NewInstance() {
//	if(info.Length() > 0) {
//		Handle<Value> argv[n];
//		for(int x=0;x<n;x++)
//			argv[x] = args[x];
//		instance = GreaseLogger::constructor->NewInstance(n, argv);
//	} else {
//		instance = GreaseLogger::constructor->NewInstance();
//	}
//}


#ifdef __cplusplus
extern "C" {
#endif

int grease_logLocal(logMeta *f, const char *s, RawLogLen len) {
	GreaseLogger *l = GreaseLogger::setupClass();
	return l->log(*f,s,len);
}

#ifdef __cplusplus
};
#endif

