{
  "targets": [
    {
      "target_name": "twlib",
      "type": "static_library",
      "defines": [
        "GIT_THREADS",
        # Node's util.h may be accidentally included so use this to guard
        # against compilation error.
        "SRC_UTIL_H_",
      ],
      "dependencies": [
      
      ],
      "cflags": [
        "-Wall",
        "-std=c++11",
        "-fPIC",
        "-D_USING_GLIBC_",
        "-Itwlib/include"
      ],
      "sources": [
       # most of twlib are just C++ templates...
        "twlib/tw_object.cpp",
        "twlib/tw_globals.cpp",
        "twlib/tw_socktask.cpp",
        "twlib/tw_globals.cpp",
        "twlib/tw_log.cpp",
        "twlib/tw_alloc.cpp",
        "twlib/tw_utils.cpp",
        "twlib/tw_task.cpp",
        "twlib/tw_stringmap.cpp"
        ]
    }
  ]
}