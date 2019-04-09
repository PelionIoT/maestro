#!/bin/bash

# http://stackoverflow.com/questions/59895/can-a-bash-script-tell-what-directory-its-stored-in
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  SELF="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done

DEPS_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
LOG=${DEPS_DIR}/../install_deps.log
GPERF_DIR=${DEPS_DIR}/gperftools-2.4
LIBUV_DIR=${DEPS_DIR}/libuv-v1.10.1
LIBTW_DIR=${DEPS_DIR}/twlib
#PCRE_DIR=${DEPS_DIR}/pcre2-10.22  # not used
#RE2_DIR=${DEPS_DIR}/re2-2017-01-01

rm -f $LOG
mkdir -p ${DEPS_DIR}/build


platform='unknown'
unamestr=`uname`
if [[ "$unamestr" == 'Linux' ]]; then
   platform='linux'
elif [[ "$unamestr" == 'FreeBSD' ]]; then
   platform='freebsd'
elif [[ "$unamestr" == 'Darwin' ]]; then
   platform='darwin'
fi



pushd $GPERF_DIR
touch $LOG
cp configure.orig configure
cp Makefile.in.orig Makefile.in
#make clean
echo "Echo building dependencies..." >> $LOG
./configure $CONFIG_OPTIONS --prefix=${DEPS_DIR}/build --enable-frame-pointers --with-pic 2>&1 >> $LOG || echo "Failed in configure for gperftools" >> $LOG
make -j4 2>&1 >> $LOG || echo "Failed to compile gperftools" >> $LOG
make install 2>&1 >> $LOG || echo "Failed to install gperftools to: $DEPS_DIR/build" >> $LOG
# does not copy libstacktrace for some reason
cp .libs/libstacktrace.* ../build/lib
#echo "Error building gperftools-2.4" > $LOG
#make clean
rm -f $GPERF_DIR/Makefile
if [ -e "$DEPS_DIR/build/include/google/tcmalloc.h" ]; then
    echo "Successful build of depenencies" >> $LOG
    echo "ok"
#    exit 0
else
    echo "Missing tcmalloc.h!!" >> $LOG
    echo "notok"
    exit 1
fi
rm -f Makefile.in
popd

echo "build libuv...."

pushd $LIBUV_DIR
if [ ! -d "build" ]; then
    echo "Need build/gyp folder. Will try to decompress..."
    tar xvfz gyp.tar.gz
fi

if [[ "$platform" == 'darwin' ]]; then
	./gyp_uv.py -f xcode
	xcodebuild -ARCHS="x86_64" -project uv.xcodeproj -configuration Release -target All
	cp ./build/Release/libuv.a $DEPS_DIR/build/lib
else
	./gyp_uv.py -f make
	make -C out
	cp ./out/Debug/libuv.a $DEPS_DIR/build/lib
fi

popd

#pushd $PCRE_DIR
#./configure --prefix=${DEPS_DIR}/build
#make
#make install
#popd

# pushd $RE2_DIR
# make
# make install prefix=${DEPS_DIR}/build
# popd


pushd $LIBTW_DIR

make tw_lib
cp libTW.a ../build/lib
cd ../build/include
if [ ! -e TW ]; then
    ln -s ../../twlib/include/TW .
fi
popd
