#include <stdio.h>
#include <sys/wait.h>

#include "logger.h"
#include "grease_lib.h"
#include <unistd.h>
#include <string.h>
#include <pthread.h>

#include <TW/tw_alloc.h>
#include <TW/tw_circular.h>
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


void loggerStartedCB(GreaseLibError *, void *d) {
	printf("Logger started.\n");
}

GreaseLibFilter *f1, *f2, *f3;

void filterAddCB(GreaseLibError *err, void *d) {
	if(!err) {
		printf("Filter added\n" );
	} else {
		printf("Filter add failure: %d %s\n",err->_errno, err->errstr);
	}
}

char output_buf[5128];

uv_thread_t printThread;

TWlib::tw_safeCircular<GreaseLibBuf *, TWlib::Allocator<TWlib::Alloc_Std> > printThese( 10, true );

void *printWork(void *d) {
	GreaseLibBuf *buf = NULL;
	while(printThese.removeOrBlock(buf)) {
		if(buf->size < 5127) {
			memcpy(output_buf,buf->data,buf->size);
			*(buf->data+buf->size+1) = '\0';
			printf("CALLBACK TARGET>>>>>>>>>%s<<<<<<<<<<<<<<<<\n",output_buf);
		} else {
			printf("OOOPS. Overflow on test output. size was %lu %d\n",buf->size, buf->_id);
		}
	//	sleep(1);
		GreaseLib_cleanup_GreaseLibBuf(buf);  // comment out to test overrun if Buffers are not being taken
	}
	return NULL;
}


void targetCallback(GreaseLibError *err, void *d, uint32_t targetId) {
	GreaseLibBuf *buf = (GreaseLibBuf *)d;
	printf("**** in targetCallback - targId %d (%lu)\n", targetId, buf->size);

	if(!printThese.addIfRoom(buf)) {
		printf("OOPS - ran out of room in queue to printout\n");
	}

//	GreaseLibBuf *buf = (GreaseLibBuf *)d;
//	if(buf->size < 5127) {
//		memcpy(output_buf,buf->data,buf->size);
//		*(buf->data+buf->size+1) = '\0';
//		printf("CALLBACK TARGET>>>>>>>>>%s<<<<<<<<<<<<<<<<\n",output_buf);
//	} else {
//		printf("OOOPS. Overflow on test output. size was %lu\n",buf->size);
//	}
//	GreaseLib_cleanup_GreaseLibBuf(buf);
}

#define FILE_TARG_OPTID 98
#define CALLBACK_TARG_OPTID 99

void targetAddCB(GreaseLibError *err, void *d) {
	if(!err) {
		GreaseLibStartedTargetInfo *info = 	(GreaseLibStartedTargetInfo *) d;
		printf("Target added - ID:%d (optsID:%d)\n",info->targId,info->optsId );
		if(info->optsId == FILE_TARG_OPTID) {
			printf("adding Filter\n");
			GreaseLib_setvalue_GreaseLibFilter(f1,GREASE_LIB_SET_FILTER_MASK,GREASE_ALL_LEVELS);
			GreaseLib_setvalue_GreaseLibFilter(f1,GREASE_LIB_SET_FILTER_TARGET,info->targId);
			GreaseLib_addFilter(f1);
		}
		if(info->optsId == CALLBACK_TARG_OPTID) {
			printf("adding Filter for callback target\n");
			GreaseLib_setvalue_GreaseLibFilter(f3,GREASE_LIB_SET_FILTER_MASK,GREASE_ALL_LEVELS);
			GreaseLib_setvalue_GreaseLibFilter(f3,GREASE_LIB_SET_FILTER_TARGET,info->targId);
			GreaseLib_addFilter(f3);
		}
	} else {
		printf("Target Failure: %d %s\n",err->_errno, err->errstr);
	}
}

const char *level_format = "%-10s ";
const char *tag_format = " [%-15s] ";
const char *origin_format = " (%-15s) ";
const int WAITSECS = 90;

#define PIPE_READ 0
#define PIPE_WRITE 1

uint32_t childStartingOriginID = 1099;

pid_t reapChildren() {
	int status = 0;
	pid_t reaped = waitpid((pid_t) -1,&status,WNOHANG|WUNTRACED|WCONTINUED);
	int was_exit = 0;
	if(reaped > 0) {
		if(WIFEXITED(status)) {
			printf("PID %d exited\n",reaped);
			was_exit =1;
		}
		if(WEXITSTATUS(status)) {
			printf("Process had exit of %d\n",WEXITSTATUS(status));
		} else {
			printf("process exited normally\n");
		}
		if(WIFSIGNALED(status)) {
			was_exit = 1;
			printf("process got terminating signal %d\n",WTERMSIG(status));
		}
		if(WIFSTOPPED(status)) {
			printf("process got STOP signal %d\n",WSTOPSIG(status));
		}
		if(WIFCONTINUED(status)) {
			printf("process got CONTINUE job control\n");
		}
	}
}

void childClosedFDCallback (GreaseLibError *err, int stream_type, int fd) {
	if(err) {

	} else {
		printf("CHILD CLOSED FD: type %d, fd %d\n", stream_type,fd);
		GreaseLib_removeFDForStdout(fd);
		pid_t lastpid;
		while((lastpid = reapChildren()) > 0) {
			printf("Reaped\n");
		}
	}
}

int createChild(const char* szCommand, char* const aArguments[], char* const aEnvironment[], const char* szMessage) {
  int aStdinPipe[2];
  int aStdoutPipe[2];
  int nChild;
  char nChar;
  int nResult;

  if (pipe(aStdinPipe) < 0) {
    perror("allocating pipe for child input redirect");
    return -1;
  }
  if (pipe(aStdoutPipe) < 0) {
    close(aStdinPipe[PIPE_READ]);
    close(aStdinPipe[PIPE_WRITE]);
    perror("allocating pipe for child output redirect");
    return -1;
  }

  GreaseLib_addOriginLabel( childStartingOriginID, szCommand, strlen(szCommand) );
  GreaseLib_addFDForStdout( aStdoutPipe[PIPE_READ], childStartingOriginID, childClosedFDCallback );
  childStartingOriginID++;


  nChild = fork();
  if (0 == nChild) {
    // child continues here

    // redirect stdin

//    if (dup2(aStdinPipe[PIPE_READ], STDIN_FILENO) == -1) {
//      perror("redirecting stdin");
//      return -1;
//    }

    // redirect stdout
    if (dup2(aStdoutPipe[PIPE_WRITE], STDOUT_FILENO) == -1) {
      perror("redirecting stdout");
      return -1;
    }

    // redirect stderr
    if (dup2(aStdoutPipe[PIPE_WRITE], STDERR_FILENO) == -1) {
      perror("redirecting stderr");
      return -1;
    }

    // all these are for use by parent only
    close(aStdinPipe[PIPE_READ]);
    close(aStdinPipe[PIPE_WRITE]);
    close(aStdoutPipe[PIPE_READ]);
    close(aStdoutPipe[PIPE_WRITE]);

    // run child process image
    // replace this with any exec* function find easier to use ("man exec")
    nResult = execve(szCommand, aArguments, aEnvironment);

    // if we get here at all, an error occurred, but we are in the child
    // process, so just exit
    perror("exec of the child process");
    exit(nResult);
  } else if (nChild > 0) {
    // parent continues here

    // close unused file descriptors, these are for child only
    close(aStdinPipe[PIPE_READ]);
    close(aStdoutPipe[PIPE_WRITE]);

    // we aren't going to send stdin to it, so...
    close(aStdinPipe[PIPE_WRITE]);



    // Include error check here
//    if (NULL != szMessage) {
//      write(aStdinPipe[PIPE_WRITE], szMessage, strlen(szMessage));
//    }

    // Just a char by char read here, you can change it accordingly
//    while (read(aStdoutPipe[PIPE_READ], &nChar, 1) == 1) {
//      write(STDOUT_FILENO, &nChar, 1);
//    }
//
//    // done with these in this example program, you would normally keep these
//    // open of course as long as you want to talk to the child
//    close(aStdinPipe[PIPE_WRITE]);
//    close(aStdoutPipe[PIPE_READ]);
  } else {
    // failed to create child
    close(aStdinPipe[PIPE_READ]);
    close(aStdinPipe[PIPE_WRITE]);
    close(aStdoutPipe[PIPE_READ]);
    close(aStdoutPipe[PIPE_WRITE]);
  }
  return nChild;
}



int main() {
	GreaseLib_start(loggerStartedCB);

	printf("before sleep ... 5 seconds\n");
	sleep(5);
	printf("after sleep - setup sink\n");

	GreaseLib_setupStandardLevels();
	GreaseLib_setupStandardTags();
	// test setting up a sink

	GreaseLibSink *sink = GreaseLib_new_GreaseLibSink(GREASE_LIB_SINK_UNIXDGRAM,"/tmp/testsocket");
	GreaseLibSink *klog_sink = GreaseLib_new_GreaseLibSink(GREASE_LIB_SINK_KLOG2,NULL);

	// setup a file destination

	GreaseLibTargetOpts *target = GreaseLib_new_GreaseLibTargetOpts();

	char outFile[] = "/tmp/output.log";

	target->file = outFile;
	target->optsId = FILE_TARG_OPTID;
	target->format_level = (char *) level_format;
	target->format_level_len = strlen(level_format);
	target->format_tag = (char *) tag_format;
	target->format_tag_len = strlen(tag_format);
	target->format_origin = (char *) origin_format;
	target->format_origin_len = strlen(origin_format);
	target->fileOpts = GreaseLib_new_GreaseLibTargetFileOpts();
	f1 = GreaseLib_new_GreaseLibFilter();

	GreaseLib_addTarget(targetAddCB, target);

	f3 = GreaseLib_new_GreaseLibFilter();

	// another target... will be ignored - since nothing if directed to it yet
	GreaseLibTargetOpts *target2 = GreaseLib_new_GreaseLibTargetOpts();
//	target2->file = outFile;
//	target2->fileOpts = GreaseLib_new_GreaseLibTargetFileOpts();

	target2->optsId = CALLBACK_TARG_OPTID;
	target2->targetCB = targetCallback;
	target2->num_banks = 10; // default is 4, let's make it
	target2->format_level = (char *) level_format;
	target2->format_level_len = strlen(level_format);
	target2->format_tag = (char *) tag_format;
	target2->format_tag_len = strlen(tag_format);
	target2->format_origin = (char *) origin_format;
	target2->format_origin_len = strlen(origin_format);
	GreaseLib_set_flag_GreaseLibTargetOpts(target2,GREASE_JSON_ESCAPE_STRINGS);
	GreaseLib_addTarget(targetAddCB, target2);

	// an origin label for testing
	const char *testLabel = "testLabel";
	GreaseLib_addOriginLabel( 1055, testLabel, strlen(testLabel) );

	int ret;

	if((ret = GreaseLib_addSink(sink)) != GREASE_LIB_OK) {
		printf("ERROR on addSink(): %d",ret);
	}
	if((ret = GreaseLib_addSink(klog_sink)) != GREASE_LIB_OK) {
		printf("ERROR on addSink() - klog: %d",ret);
	}
	printf("after setup sink\n");


	GreaseLibSink *sink2 = GreaseLib_new_GreaseLibSink(GREASE_LIB_SINK_SYSLOGDGRAM,"/dev/log");

	if((ret = GreaseLib_addSink(sink2)) != GREASE_LIB_OK) {
		printf("ERROR on addSink(): %d",ret);
	}
	printf("after setup sink\n");


	pthread_t printThreadId;
	if(pthread_create(&printThreadId, NULL, printWork, NULL)) {
		fprintf(stderr, "Error printWork thread\n");
	}

//	printf("Will shutdown in %d seconds\n",WAITSECS);

	printf("Gonna exec a command in 5 seconds\n");
	sleep(10);
	printf("\nexec ls -al\n");
	char *args[4];

	args[0] = "/bin/ls";
	args[1] = "-al";//"test-re.cc";
	args[2] = NULL;
	createChild("/bin/ls",args,NULL,NULL);

	sleep(1);

	args[0] = "/home/ed/work/gostuff/bin/devicedb";
	args[1] = "start";
	args[2] = "-conf=/home/ed/work/gostuff/config.yaml";//"test-re.cc";
	args[3] = NULL;
	createChild("/home/ed/work/gostuff/bin/devicedb",args,NULL,NULL);

	GreaseLib_waitOnGreaseShutdown();
	//	sleep(WAITSECS);
//
//	printf("sleep over\n");
//
//	GreaseLib_shutdown(NULL);

}
;
