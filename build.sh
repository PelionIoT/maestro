#!/bin/bash

# Copyright (c) 2018, Arm Limited and affiliates.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  SELF="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done

THIS_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

pushd $THIS_DIR

# you need the -P !! --> http://stackoverflow.com/questions/32477816/safely-replacing-text-with-m4-disabling-new-defines



# m4 -P macro-setup.m4 -F m4setup.m4f

if [ ! -z "$DEBUG2" ]; then
	echo "DEBUG2 IS ON!"
	DEBUG2="-D DEBUG_OUT2=fmt.Printf(\"***DEBUG_GO2:\"+\$@) -D IFDEBUG2=\$1"
	# if [ "$DEBUG" -ge "1" ]; then
else
	DEBUG2="-D DEBUG_OUT2=  -D IFDEBUG2=\$2"
fi



if [ ! -z "$DEBUG" ]; then
	echo "DEBUG IS ON!"
	PREPROCESS="m4 -P -D IFDEBUG=\$1 -D NODEBUG= -D DEBUG=\$@ macro-setup.m4 -D DEBUG_OUT=fmt.Printf(\"***DEBUG_GO:\"+\$@) ${DEBUG2}"
	# if [ "$DEBUG" -ge "1" ]; then

	# find
	make native.a-debug
else
	PREPROCESS="m4 -P -D IFDEBUG=\$2 -D NODEBUG=\$@ -D DEBUG= -D DEBUG_OUT= macro-setup.m4  ${DEBUG2}"
	make native.a
fi

#if [ ! -z "$TIGHT" ]; then
#    GO_LDSWITCH="-ldflags=\"-s -w\""
#fi

if [ ! -z "$PRETEND" ]; then
	echo "PRETEND IS ON!"
fi



for f in $(find  src -name '*.go'); do
	echo "--> "$f
#	echo `basename $f`
	FILENAME=`basename $f`
	DIR1=`dirname $f`
	DIR1=${DIR1#*/}
#	echo "------"
	if [ "$DIR1" != "src" ]; then
		echo "mkdir -p $DIR1"
		mkdir -p $DIR1
#		echo "--->"$DIR1"<--"
#		echo "****"
#		echo "output: $DIR1/$FILENAME"
		echo $PREPROCESS "src/$DIR1/$FILENAME" "> $DIR1/$FILENAME"
		if [ -z "$PRETEND" ]; then $PREPROCESS src/$DIR1/$FILENAME > $DIR1/$FILENAME; fi
	else
#		echo "output: $FILENAME"
		echo $PREPROCESS "src/$FILENAME" "> $FILENAME"
		if [ -z "$PRETEND" ]; then $PREPROCESS src/$FILENAME > $FILENAME; fi
	fi
done

for f in $(find  src -name '*\.[ch]'); do
	echo "--> "$f
#	echo `basename $f`
	FILENAME=`basename $f`
	DIR1=`dirname $f`
	DIR1=${DIR1#*/}
#	echo "------"
	if [ "$DIR1" != "src" ]; then
		echo "mkdir -p $DIR1"
		mkdir -p $DIR1
#		echo "--->"$DIR1"<--"
#		echo "****"
#		echo "output: $DIR1/$FILENAME"
		# echo $PREPROCESS "src/$DIR1/$FILENAME" "> $DIR1/$FILENAME"
		# if [ -z "$PRETEND" ]; then $PREPROCESS src/$DIR1/$FILENAME > $DIR1/$FILENAME; fi
		echo "src/$DIR1/$FIELNAME --> $DIR1/$FILENAME"
		cp src/$DIR1/$FILENAME $DIR1/$FILENAME
	else
#		echo "output: $FILENAME"
		# echo $PREPROCESS "src/$FILENAME" "> $FILENAME"
		# if [ -z "$PRETEND" ]; then $PREPROCESS src/$FILENAME > $FILENAME; fi
		echo "src/$FIELNAME --> $FILENAME"
		cp src/$FILENAME $FILENAME
	fi
done


popd

if [ "$1" == "removesrc" ]; then
#	echo "mv src .src"
	if [ -d src ]; then
		mv src .src
	fi
	shift
fi

# let's get the current commit, and make sure Version() has this.
COMMIT=`git rev-parse --short=7 HEAD`
DATE=`date`
sed -i -e "s/COMMIT_NUMBER/${COMMIT}/g" maestroutils/status.go 
sed -i -e "s/BUILD_DATE/${DATE}/g" maestroutils/status.go 

# highlight errors: https://serverfault.com/questions/59262/bash-print-stderr-in-red-color
color()(set -o pipefail;"$@" 2>&1>&3|sed $'s,.*,\e[31m&\e[m,'>&2)3>&1

if [ "$1" != "preprocess_only" ]; then
	pushd $GOPATH/bin
	if [ ! -z "$TIGHT" ]; then
	    color go build -ldflags="-s -w" "$@" github.com/armPelionEdge/maestro/maestro 
	else
	    color go build "$@" github.com/armPelionEdge/maestro/maestro 
	fi

# 
#go install -x github.com/armPelionEdge/maestro/maestro
	popd
fi
