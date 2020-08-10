/*
 * test-re.cc
 *
 *  Created on: Jan 30, 2017
 *      Author: ed
 * (c) 2017, WigWag Inc.
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


#include <string>
#include <iostream>
#include <re2/re2.h>
#include <assert.h>

int main() {
	RE2 re_1("([0-9]*)\\s+([a-zA-z]*)");
	int d;
	std::string s;

	assert(RE2::FullMatch("09811 ksajd",re_1, &d, &s));

	printf("capture: %d\n",d);
	printf("cap2: %s\n",s.c_str());

}



/**
 * Build:
 *
 * g++ -std=c++11 -Wall -Ideps/build/include -pthread  test-re.cc -o test-re deps/build/lib/libre2.a
 *
 */
