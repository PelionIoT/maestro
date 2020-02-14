#!/bin/bash

#
# Copyright (c) 2019, Arm Limited and affiliates.
#
# SPDX-License-Identifier: MIT
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to
# deal in the Software without restriction, including without limitation the
# rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
# sell copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#


SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do 
    SELF="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" 
done

SELF="$( cd -P "$( dirname "$SOURCE" )" && pwd )" # script's directory

cd $SELF
mkdir -p deps/bin deps/lib deps/include deps/src

cd $SELF/deps/src


GREASE_LIB="greaseLib"
GREASE_LIB_REPO="https://github.com/armPelionEdge/greaseLib.git"
GREASE_LIB_BRANCH="master"

if [ ! -d $GREASE_LIB ]; then
    echo ">>>>>>>>> Getting greaselib repo: $GREASE_LIB_REPO @branch $GREASE_LIB_BRANCH"
    git clone $GREASE_LIB_REPO
fi

cd $GREASE_LIB
if [ -d ".git" ]; then
    echo ">>>>>>>>> Updating greaselib repo: $GREASE_LIB_REPO @branch $GREASE_LIB_BRANCH"
    git pull
    git checkout $GREASE_LIB_BRANCH
fi

if [ "$1" != "skipdeps" ]; then
    echo ">>>>>>>>> Building libgrease dependencies..."
    # install the libgrease dependencies
    deps/install-deps.sh
else
    echo ">>>>>>>>> Ok. Skipping dependencies"
fi

echo "Building libgrease.a"
# make sure we build fresh
rm -f *.o *.a
# make the server version - which basically just bypasses checks for symbols
# on the client logging code
make libgrease.a-server
make libgrease.so.1
cp *.h $SELF/deps/include


if [ -e libgrease.a ]; then
    # migrate all of the greaselib dependencies up to the folder Go will use
    cp -r deps/build/lib/* $SELF/deps/lib
    cp -r deps/build/include/* $SELF/deps/include
    # move our binary into lib - static is all we use
    cp libgrease.a $SELF/deps/lib
    cp *.h $SELF/deps/include
    echo ">>>>>>>>> Success. libgrease.a ready."
else
    echo ">>>>>>>>> ERROR: libgrease.a missing or not built."
fi


if [ -e libgrease.so.1 ]; then
    # migrate all of the greaselib dependencies up to the folder Go will use
    cp -r deps/build/lib/* $SELF/deps/lib
    cp -r deps/build/include/* $SELF/deps/include
    cp $SELF/deps/src/greaseLib/deps/libuv-v1.10.1/include/uv* $SELF/deps/include
    # move our binary into lib - static is all we use
    cp libgrease.so.1 $SELF/deps/lib
    cp *.h $SELF/deps/include
    echo ">>>>>>>>> Success. libgrease.so.1 ready."
    cd $SELF/deps/lib
    ln -sf libgrease.so.1 libgrease.so
    echo ">>>>>>>>> Success. libgrease.so link ready."
else
    echo ">>>>>>>>> ERROR: libgrease.so.1 missing or not built."
fi
