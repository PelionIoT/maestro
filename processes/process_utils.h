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

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#include <unistd.h>
#include <stdint.h>

#define PROCESS_USE_PGID 0x1
#define PROCESS_NEW_SID 0x2  // overrides the above

typedef struct {
	char *chroot;
	int die_on_parent_sig;
	uint32_t flags;
	uint32_t originLabel; // set by ExecFile - the ID used by the logger for this process
	int env_GREASE_ORIGIN_ID; // if non-zero, then a GREASE_ORIGIN_ID env var will be created with the origin label ID
	pid_t pgid;
	char *message;
	char *jobname; // if not NULL, then this will be the 'origin' label
	int ok_string;
	int stdout_fd; // sometimes set, if we need to watch the output
	int stderr_fd;
} childOpts;

typedef struct {
	int _errno;
	pid_t pid;
} execErr;

void initNative();

int createChild(char* szCommand, char* aArguments[], char* aEnvironment[], char* szMessage, childOpts *opts, execErr *err);

void setCStringInArray(char **array, char *s, int pos);

// defined in Go land:
void sawClosedRedirectedFD(void);

char **makeCStringArray(int n);
void freeCStringArray(char **a);
void setCStringInArray(char **a, char *s, int pos);

#ifdef DEBUG_MAESTRO_NATIVE
#define DBG_MAESTRO(s,...) fprintf(stderr, "**DEBUG MAESTRO** " s "\n", ##__VA_ARGS__ )
#else
#define DBG_MAESTRO(s,...) {}
#endif

#define ERR_MAESTRO(s,...) fprintf(stderr, "**ERROR MAESTRO** " s "\n", ##__VA_ARGS__ )