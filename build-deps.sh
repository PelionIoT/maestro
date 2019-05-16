#!/bin/bash


SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  SELF="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done

THIS_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

pushd $THIS_DIR

cd $THIS_DIR/vendor/github.com/armPelionEdge/greasego/deps/src/greaseLib/deps/libuv-v1.10.1

if [ ! -d build ]; then
    git clone https://chromium.googlesource.com/external/gyp.git build/gyp
fi

cd $THIS_DIR/vendor/github.com/armPelionEdge/greasego/deps/src/greaseLib/deps
./install-deps.sh

cd $THIS_DIR/vendor/github.com/armPelionEdge/greasego
./build-deps.sh

cd $THIS_DIR/vendor/github.com/armPelionEdge/greasego

# build greasego
DEBUG=1 ./build.sh

popd

