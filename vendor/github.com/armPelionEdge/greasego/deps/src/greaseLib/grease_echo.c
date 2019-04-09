/*
 * send-multiple-test.c
 *
 *  Created on: Sep 3, 2015
 *      Author: ed
 * (c) 2015, WigWag Inc.
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


#include <stdio.h>
#include <errno.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <stdlib.h>

//#define GREASE_DEBUG_MODE 1
#include "grease_common_tags.h"
#define GREASE_NO_DEFAULT_NATIVE_ORIGIN 1    // we dont need to specify an origin with this client
#define GLOG_DEFAULT_TAG GREASE_ECHO_TAG
#include "grease_client.h"

#define ERROR_OUT(s,...) fprintf(stderr, "**ERROR** " s, ##__VA_ARGS__ )//#define ERROR_PERROR(s,...) fprintf(stderr, "*****ERROR***** " s, ##__VA_ARGS__ );
#define ERROR_PERROR(s,...) { perror("**ERROR** [ %s ] " s, ##__VA_ARGS__ ); }
#define DBG_OUT(s,...) fprintf(stderr, "**DEBUG** " s, ##__VA_ARGS__ )
#define IF_DBG( x ) { x }


void bye(int e) {
	SHUTDOWN_GLOG;
	exit(e);
}

int main(int argc, char *argv[]) {
	if(argc < 2) {
		exit(1);
	}

	int n = 1;
	char *socket_path = NULL;
	int opt_n = 0;

	DECL_LOG_META( echo_meta, GREASE_ECHO_TAG, GREASE_LEVEL_LOG, GREASE_GREASEECHO_ORIGIN );


	if(argv[n][0] == '-' && argv[n][1] == '-') {
		if(!strcmp(argv[1]+2,"help")) {
			printf("Usage: grease_echo [--check] | { [--socket PATH] [--origin NUM] [--[LEVEL]] \"string here\" }\n"
				   "            --socket PATH  use a custom path to the grease socket. Must be first argument.\n"
				   "                           Needs an absolute path. default:" GREASE_DEFAULT_SINK_PATH "\n"
				   "            --origin NUM   use a specified origin-id number, instead of the 'grease echo' constant\n"
				   "            --check        will check to see if logger is live. (Use with no other args)\n"
				   "LEVELs:     --error\n"
				   "            --warn\n"
				   "            --success\n"
				   "            --info\n"
				   "            --debug\n"
				   "            --debug2\n"
				   "            --debug3\n"
				   "            --user1\n"
				   "            --user2\n");
			exit(1);
		}

		if(!strcmp(argv[1]+2,"check")) {
			if(grease_initLogger() != GREASE_OK) {
				printf("Grease not running.\n");
				bye(1);
			} else {
				printf("Grease running & OK.\n");
				bye(0);
			}
		}

		int _loop = 0;
		do {
			_loop = 0;
			if(argv[opt_n+1] && !strcmp(argv[opt_n+1]+2,"socket")){
				socket_path = argv[opt_n+2];
				opt_n += 2;
				_loop = 1;
			}

			if(argv[opt_n+1] && !strcmp(argv[opt_n+1]+2,"origin")) {
				printf("GOT %s\n", argv[opt_n+2]);
				int z = atoi(argv[opt_n+2]);
				printf("GOT %d\n", z);
				if(z == 0) {
					fprintf(stderr,"Can't get integer from --origin [num]. Ignoring\n");
					continue;
				}
				if(z < 1 || z > GREASE_RESERVED_ORIGINS_START) {
					fprintf(stderr,"Warning: provided origin value is out of range. Reserved numbers start at: %d\n",GREASE_RESERVED_ORIGINS_START);
				}
				echo_meta.origin = z;
				opt_n += 2;
				_loop = 1;
			}

		} while(_loop);




		if(grease_fastInitLogger_extended(socket_path) != GREASE_OK) {
			fprintf(stderr,"    Error: Grease not running.\n");
		}

		if(argc > opt_n+2 && argv[opt_n+2] && argv[opt_n+2][0] != '\0') {
			if(!strcmp(argv[n+1]+2,"info")) {
				echo_meta.level = GREASE_LEVEL_INFO;
				grease_printf(&echo_meta, argv[2] );
				bye(0);
			} else
			if(!strcmp(argv[opt_n+1]+2,"error")) {
				echo_meta.level = GREASE_LEVEL_ERROR;
				grease_printf(&echo_meta,argv[opt_n+2]);
				bye(0);
			} else
			if(!strcmp(argv[opt_n+1]+2,"warn")) {
				echo_meta.level = GREASE_LEVEL_WARN;
				grease_printf(&echo_meta,argv[opt_n+2]);
				bye(0);
			} else
			if(!strcmp(argv[opt_n+1]+2,"success")) {
				echo_meta.level = GREASE_LEVEL_SUCCESS;
				grease_printf(&echo_meta,argv[opt_n+2]);
				bye(0);
			} else
			if(!strcmp(argv[opt_n+1]+2,"debug")) {
				echo_meta.level = GREASE_LEVEL_DEBUG;
				printf("ORIGIN %d\n", echo_meta.origin);
				grease_printf(&echo_meta,argv[opt_n+2]);
				bye(0);
			} else
			if(!strcmp(argv[opt_n+1]+2,"debug2")) {
				echo_meta.level = GREASE_LEVEL_DEBUG2;
				grease_printf(&echo_meta,argv[opt_n+2]);
				bye(0);
			} else
			if(!strcmp(argv[opt_n+1]+2,"debug3")) {
				echo_meta.level = GREASE_LEVEL_DEBUG3;
				grease_printf(&echo_meta,argv[opt_n+2]);
				bye(0);
			} else
			if(!strcmp(argv[opt_n+1]+2,"user1")) {
				echo_meta.level = GREASE_LEVEL_USER1;
				grease_printf(&echo_meta,argv[opt_n+2]);
				bye(0);
			} else
			if(!strcmp(argv[opt_n+1]+2,"user2")) {
				echo_meta.level = GREASE_LEVEL_USER2;
				grease_printf(&echo_meta,argv[opt_n+2]);
				bye(0);
			} else {
				fprintf(stderr,"grease_echo: Unknown LEVEL.\n");
				echo_meta.level = GREASE_LEVEL_LOG;
				grease_printf(&echo_meta, argv[opt_n+2] );
				bye(1);
			}
		}
	}

	if(grease_fastInitLogger_extended(socket_path) != GREASE_OK) {
		fprintf(stderr,"    Error: Grease not running.\n");
	}
	if(argv[opt_n+1] && argv[opt_n+1][0] != '\0') {
		echo_meta.level = GREASE_LEVEL_LOG;
		grease_printf(&echo_meta, argv[opt_n+1] );
	}
	bye(0);

	return 0;
}


/**
 * BUILD:  gcc grease_echo.c  grease_client.c -std=c99 -o grease_echo -ldl
 */
