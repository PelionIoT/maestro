/*
 * Copyright (c) 2018, Arm Limited and affiliates.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include <stdio.h>
#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#include <unistd.h>
#include <sys/wait.h>
#include <stdlib.h>
#include <fcntl.h>

// Linux only:
#include <sys/prctl.h>

#include <uv.h>

#include "grease_lib.h"

#define DEBUG_MAESTRO_NATIVE 1
#include "process_utils.h"


#define PIPE_READ 0
#define PIPE_WRITE 1


uv_mutex_t forkLock;

void initNative() {
	uv_mutex_init(&forkLock);
}

// this error string stuff follow the same pattern in error-common.cc / .h in greaseLib
// ensure we get the XSI compliant strerror_r():
// see: http://man7.org/linux/man-pages/man3/strerror.3.html
extern int __xpg_strerror_r (int __errnum, char *__buf, size_t __buflen);
#define MAESTRO_ERRNO_STRING_MAX_BUF 255
#define ERR_STRERROR_R(ernum,b,len) __xpg_strerror_r(ernum, b, len)

char *get_error_str(int _errno) {
	char *ret = (char *) malloc(MAESTRO_ERRNO_STRING_MAX_BUF);
	int r = ERR_STRERROR_R(_errno,ret,MAESTRO_ERRNO_STRING_MAX_BUF);
	if ( r != 0 ) ERR_MAESTRO("strerror_r bad return: %d\n",r);
	return ret;
}

void free_error_str(char *str) {
	if (NULL != str) {
		free(str);
	}
}


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

// Here memory is allocated for array of strings
// Memory for string elements in arrat is not allocated here
// Look for `C.CString(s)` in func convertToCStrings for allocation
char **makeCStringArray(int n) {
	int size = sizeof(char *) * n;
	char **ret = (char **) malloc(size);
	if(ret) {
		memset(ret, 0, size);
	}
	return ret;
}

// Free all the memory allocated in array elements before freeing the array itself
void freeCStringArray(char **a) {
	int z = 0;
	if(a) {
		char *s = a[z];
		while(s) {
			free(s);
			z++;
			s = a[z];
		}
		free(a);
	}
}

void setCStringInArray(char **a, char *s, int pos) {
	a[pos] = s;
}

void childClosedFDCallback (GreaseLibError *err, int stream_type, int fd) {
	if(err) {

	} else {
		printf("CHILD CLOSED FD: type %d, fd %d for pid %d\n", stream_type,fd,(int)getpid());
		
		if(stream_type == 1) {
			GreaseLib_removeFDForStdout(fd);
		} else if (stream_type==2) {
			GreaseLib_removeFDForStderr(fd);	
		}
		
		sawClosedRedirectedFD(); // call back into Go land

		// pid_t lastpid;
		// while((lastpid = reapChildren()) > 0) {
		// 	printf("Reaped %d\n",lastpid);
		// }
	}
}


int createChild(char* szCommand,
		char* aArguments[],
		char* aEnvironment[],
		char* szMessage,
		childOpts *opts,
		execErr *err) {

	int aStdinPipe[2];
	int aStdoutPipe[2];
	int aStderrPipe[2];
	int aErrorPipe[2];
	int nChild;
	char nChar;
	int nResult;

	DBG_MAESTRO("createChild entry %s %p %p %p %p %p", szCommand, aArguments, aEnvironment, szMessage, opts, err);
	if(szMessage) {
		DBG_MAESTRO("send message:%s\n",szMessage);
	}

	if (pipe2(aErrorPipe,O_CLOEXEC) < 0) {
		err->_errno = errno;
		perror("allocating pipe for execve error capture.");
		return -1;
	}

	if (pipe(aStdinPipe) < 0) {
		err->_errno = errno;
		perror("allocating pipe for child input redirect");
		return -1;
	}
	if (pipe(aStdoutPipe) < 0) {
		close(aStdinPipe[PIPE_READ]);
		close(aStdinPipe[PIPE_WRITE]);
		err->_errno = errno;
		perror("allocating pipe for child output redirect");
		return -1;
	}

	if (pipe(aStderrPipe) < 0) {
		close(aStdoutPipe[PIPE_READ]);
		close(aStdoutPipe[PIPE_WRITE]);
		close(aStdinPipe[PIPE_READ]);
		close(aStdinPipe[PIPE_WRITE]);
		err->_errno = errno;
		perror("allocating pipe for child stderr redirect");
		return -1;
	}

	DBG_MAESTRO("createChild 1");

	uint32_t childStartingOriginID = 1000;
	GreaseLib_getUnusedOriginId(&childStartingOriginID);
	if(opts) {
		opts->originLabel = childStartingOriginID;
	}
	if(opts && opts->jobname) {
		GreaseLib_addOriginLabel( childStartingOriginID, opts->jobname, strlen(opts->jobname) );
		DBG_MAESTRO("Logging: set label to %s %d",opts->jobname,childStartingOriginID);
	} else {
		GreaseLib_addOriginLabel( childStartingOriginID, szCommand, strlen(szCommand) );
	}
	DBG_MAESTRO("createChild 1.1");
	// GreaseLib_addFDForStdout( aStdoutPipe[PIPE_READ], childStartingOriginID, childClosedFDCallback );
	DBG_MAESTRO("createChild 1.2");
	if(!opts->ok_string) {
	} else {
		DBG_MAESTRO("Not redirecting STDOUT - have ok_string opt");
		opts->stdout_fd = aStdoutPipe[PIPE_READ];
	}

	char *tempEnv[1]; // used if a environmental array was not passed in
	if (opts->env_GREASE_ORIGIN_ID) {
		DBG_MAESTRO("createChild 1.2a");
		char *out = (char *) malloc(30);
		sprintf(out,"GREASE_ORIGIN_ID=%d",childStartingOriginID);
		DBG_MAESTRO("Logging: %s",out);
		int z = 0;
		// See processMgmt.go: convertToCStrings() - we make some extra room there in case this is needed.
		if(aEnvironment) {
			while(aEnvironment[z] != NULL) {
				z++;
			}
			aEnvironment[z] = out;
		} else {
			aEnvironment = tempEnv;
			tempEnv[0] = out;
		}
	}

	DBG_MAESTRO("createChild - about to fork()");
	uv_mutex_lock(&forkLock);

	pid_t ppid_before_fork = getpid();

	nChild = fork();
	DBG_MAESTRO("createChild - fork() == %d",nChild);
	if (0 == nChild) {
		// child continues here

		// all these are for use by parent only
		close(aStdinPipe[PIPE_WRITE]);
		close(aStdoutPipe[PIPE_READ]);
		close(aStderrPipe[PIPE_READ]);
		close(aErrorPipe[PIPE_READ]);

		// check for die_on_parent_sig
		if (opts->die_on_parent_sig) {
			DBG_MAESTRO("createChild - see die_on_parent_sig");
			// https://stackoverflow.com/questions/284325/how-to-make-child-process-die-after-parent-exits
			// WARNING:
			// Unfortunately, if a child forks from a thread, and then the thread exit, the child process wil get the SIGTERM.
			int r = prctl(PR_SET_PDEATHSIG, SIGTERM);
			if (getppid() != ppid_before_fork)
				exit(1);
		}

		// redirect stdin
		if (dup2(aStdinPipe[PIPE_READ], STDIN_FILENO) == -1) {
			perror("redirecting stdin");
			close(aErrorPipe[PIPE_WRITE]);
			close(aStdinPipe[PIPE_READ]);
			close(aStdoutPipe[PIPE_WRITE]);
			close(aStderrPipe[PIPE_WRITE]);
			exit(1);
		}

		// redirect stdout
		if (dup2(aStdoutPipe[PIPE_WRITE], STDOUT_FILENO) == -1) {
			perror("redirecting stdout");
			close(aErrorPipe[PIPE_WRITE]);
			close(aStdinPipe[PIPE_READ]);
			close(aStdoutPipe[PIPE_WRITE]);
			close(aStderrPipe[PIPE_WRITE]);
			exit(1);
		}

		// redirect stderr
		if (dup2(aStderrPipe[PIPE_WRITE], STDERR_FILENO) == -1) {
			perror("redirecting stderr");
			close(aErrorPipe[PIPE_WRITE]);
			close(aStdinPipe[PIPE_READ]);
			close(aStdoutPipe[PIPE_WRITE]);
			close(aStderrPipe[PIPE_WRITE]);
			exit(1);
		}

		// tell the kernel to close this FD if execve kicks off correctly
		fcntl(aErrorPipe[PIPE_WRITE], F_SETFD, FD_CLOEXEC);
		close(aStdinPipe[PIPE_READ]);
		close(aStdoutPipe[PIPE_WRITE]);
		close(aStderrPipe[PIPE_WRITE]);
		if(!opts) {
			// Default:
			// make this process in a new process group
			setpgid(getpid(),0);
		} else {
			if(opts->flags & PROCESS_NEW_SID) {
				setsid(); // create a new session
			} else {
				if(!(opts->flags & PROCESS_USE_PGID)) {
					setpgid(getpid(),0);
				} else {
					setpgid(getpid(),opts->pgid);
				}
			}
		}

		DBG_MAESTRO("createChild about to execvpe()\n");
		// run child process image
		// replace this with any exec* function find easier to use ("man exec")
		nResult = execvpe(szCommand, aArguments, aEnvironment);

		char *errs = get_error_str(errno);
		DBG_MAESTRO("createChild's child process past execvpe(%s) ERROR: %d %d %s\n",szCommand,nResult,errno,errs);
		fprintf(stderr,"createChild's child process past execvpe(%s) ERROR: %d %d %s\n",szCommand,nResult,errno,errs);
		free_error_str(errs);

		// FAILURE...

		// this causing a SIGSEGV ?? --->
		if(write(aErrorPipe[PIPE_WRITE],&errno, sizeof(int)) < sizeof(int)) {
			perror("Pipe aErrorPipe communication failed when createChild() failed\n");
		}
		// <----

		//    close(aErrorPipe[PIPE_WRITE]);

		DBG_MAESTRO("createChild exit() child");
		exit(nResult);
	} else if (nChild > 0) {
		// parent continues here
		uv_mutex_unlock(&forkLock);

		DBG_MAESTRO("createChild parent past fork() and lock.\n");

		// close unused file descriptors, these are for child only
		close(aStdinPipe[PIPE_READ]);
		close(aStdoutPipe[PIPE_WRITE]);
		close(aStderrPipe[PIPE_WRITE]);
		close(aErrorPipe[PIPE_WRITE]);

		int _errno = 0;

		int n = read(aErrorPipe[PIPE_READ],&_errno,sizeof(int));
		DBG_MAESTRO("createChild reading error pipe %d for pid %d\n",n,(int) getpid ());
		if(n > 0) {
			// if non zero, then the error pipe was not closed, and
			// execve failed
			// if(errno) {
			ERR_MAESTRO("******** createChild ********* errno: %d\n",_errno);
			// }
			err->_errno = _errno;

		} else {
			err->pid = nChild;
		}
		close(aErrorPipe[PIPE_READ]);

		if (!_errno && szMessage != NULL) {
			DBG_MAESTRO("createChild writing szMessage\n");
			int ret = write(aStdinPipe[PIPE_WRITE], szMessage, strlen(szMessage));
			// Include error check here
		}

		close(aStdinPipe[PIPE_WRITE]);

		GreaseLib_addFDForStderr( aStderrPipe[PIPE_READ], childStartingOriginID, childClosedFDCallback );
		GreaseLib_addFDForStdout( aStdoutPipe[PIPE_READ], childStartingOriginID, childClosedFDCallback );

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
		uv_mutex_unlock(&forkLock);
		// failed to create child
		close(aStdinPipe[PIPE_READ]);
		close(aStdinPipe[PIPE_WRITE]);
		close(aStdoutPipe[PIPE_READ]);
		close(aStdoutPipe[PIPE_WRITE]);
		close(aStderrPipe[PIPE_READ]);
		close(aStderrPipe[PIPE_WRITE]);
		close(aErrorPipe[PIPE_READ]);
		close(aErrorPipe[PIPE_WRITE]);
	}
	return nChild;
}
