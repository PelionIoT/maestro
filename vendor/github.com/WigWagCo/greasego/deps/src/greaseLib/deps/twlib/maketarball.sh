#!/bin/bash


# makes TW_lib tarball

function print_usage () {
    echo "$0 tarball-name"
    echo "- makes a TW_lib tarball"
}


case "$SYSTYPE" in 
    Darw)
# OS X seems to not like this anymore
#	COLOR_BOLD="echo -n '\\[1m'"
#	COLOR_RED="echo -n '\\[31m'"
#	COLOR_MAGENTA="echo -n '\\[35m;'"
#	COLOR_YELLOW="echo -n '\\[33m'"
#	COLOR_GREEN="echo -n '\\[32m'"
#	COLOR_NORMAL="echo -n '\\[0m'"
	;;
    Linu|CYGW)   
	COLOR_BOLD="echo -ne '\E[1m'"
	COLOR_RED="echo -ne '\E[31m'"
	COLOR_MAGENTA="echo -ne '\E[35m'"
	COLOR_YELLOW="echo -ne '\E[33m'"
	COLOR_GREEN="echo -ne '\E[32m'"
	COLOR_NORMAL="echo -ne '\E[0m'"
	;;
esac




if [ "$#" -lt 1 ]; then
    print_usage
    exit -1
fi

TARNAME=$1

if [ -e manifest.lst ]; then
    echo "Run in top most dir of TW_lib project"
fi

THIS_DIR=`pwd`
MY_DIR=`basename $THIS_DIR`

pushd ..
tar cvfz $1 --exclude=.svn $MY_DIR/ 
popd
mv ../$1 .
