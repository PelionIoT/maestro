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


#ifndef Callbacks_H_
#define Callbacks_H_

#include "grease_lib.h"

void greasego_startGreaseLibCB(int in);
void greasego_addTargetCB(GreaseLibError *err, void *d);
int greasego_wrapper_addTarget(GreaseLibTargetOpts *opts);
int greasego_wrapper_modifyDefaultTarget(GreaseLibTargetOpts *opts);
void greasego_commonTargetCB(GreaseLibError *err, void *d, uint32_t targetId);
void greasego_setSelfOriginLabel(char *s) ;

void greasego_childClosedFDCallback (GreaseLibError *err, int stream_type, int fd);
void zero_meta( logMeta *m );


extern logMeta go_meta_info;
extern logMeta go_meta_warning;
extern logMeta go_meta_error;
extern logMeta go_meta_debug;
extern logMeta go_meta_success;

extern logMeta go_self_meta_info;
extern logMeta go_self_meta_warning;
extern logMeta go_self_meta_error;
extern logMeta go_self_meta_debug;
extern logMeta go_self_meta_success;
#endif