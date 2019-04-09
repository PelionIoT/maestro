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

var util = require('util');
var fs = require('fs');
var _ = require('underscore');

var LEVELS_default = {  // these are also defined in greaseLog/index.js
    'log'      : 0x01,
    'error'    : 0x02,
    'warn'     : 0x04,
    'debug'    : 0x08,
    'debug2'   : 0x10,
    'debug3'   : 0x20,
    'user1'    : 0x40,
    'user2'    : 0x80,  // Levels can use the rest of the bits too...
    'success'  : 0x100,
    'info'     : 0x200,
    'verbose'  : 0x400,
    'trace'    : 0x800
};

var originalProcessStdoutWrite = null;
var originalProcessStderrWrite = null;

var dbg_logger = function() {};

var setup_fallback = function(opts,config,donecb) {
    var ret = {
        log : function() {
            if(arguments.length > 0) {
                arguments[0] = "[LOG]     " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        warn : function() {
            if(arguments.length > 0) {
                arguments[0] = "[WARN]    " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        error : function() {
            if(arguments.length > 0) {
                arguments[0] = "[ERROR]   " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        user1 : function() {
            if(arguments.length > 0) {
                arguments[0] = "[USER1]   " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        user2 : function() {
            if(arguments.length > 0) {
                arguments[0] = "[USER2]   " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        debug : function() {
            if(arguments.length > 0) {
                arguments[0] = "[DEBUG]   " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        debug2 : function() {
            if(arguments.length > 0) {
                arguments[0] = "[DEBUG2]  " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        debug3 : function() {
            if(arguments.length > 0) {
                arguments[0] = "[DEBUG3]  " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        info : function() {
            if(arguments.length > 0) {
                arguments[0] = "[INFO]    " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        verbose : function() {
            if(arguments.length > 0) {
                arguments[0] = "[VERBOSE] " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        success : function() {
            if(arguments.length > 0) {
                arguments[0] = "[SUCCESS] " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        trace : function() {
            if(arguments.length > 0) {
                arguments[0] = "[TRACE]   " + arguments[0];
                console.log.apply(undefined,arguments);
            }
        },
        setGlobalOpts: function() {}
    }
    if (typeof donecb == 'function') {
        donecb();
    }
    return ret;
};

var grease = null;


var setup = function(opts,    // standard options for grease
                     config,donecb)  // config file with Filters, Targets, etc.
{
    if(opts && opts.verbose > 1) {
        dbg_logger = function() {
            if(arguments[0]) arguments[0] = "LOGGER: " + arguments[0];
            console.log.apply(undefined,arguments);
        }
    }
    dbg_logger("In setup()")
    if(!opts) opts = {};
    if (grease) {
        dbg_logger("grease native require()ed")
        if(opts.always_use_origin !== false) {
            dbg_logger("always_use_origin = true")
            opts.always_use_origin = true;
        } else {
            dbg_logger("always_use_origin = false")
        }
        var logger = grease(opts); // TODO custom opts here later
        dbg_logger("grease native past cstor")
        if(!opts.client_only) {
            var logger_default_target = {
                'default': {
                    tty: 'console',
                    format: {
                        time: "[%ld:%03d] ",
                        level: "%-10s ", // left align
                        tag: "\x1B[33m%-10s\x1B[39m ",
                        origin: "\x1B[37m\x1B[100m%-10s\x1B[39m\x1B[49m "
                    },
                    filters: [
                        {
                            target: 'default',
                            mask: 'debug',
                            pre: "\x1B[36m",  // grey // ['\x1B[36m', '\x1B[39m'],
                            post: "\x1B[39m"
                        },
                        {
                            target: 'default',
                            mask: 'debug2',
                            pre: "\x1B[36m",  // grey // ['\x1B[36m', '\x1B[39m'],
                            post: "\x1B[39m"
                        },
                        {
                            target: 'default',
                            mask: 'debug3',
                            pre: "\x1B[36m",  // grey // ['\x1B[36m', '\x1B[39m'],
                            post: "\x1B[39m"
                        },
                        {
                            target: 'default',
                            mask: 'trace',
                            pre: "\x1B[90m",  // grey // ['\x1B[36m', '\x1B[39m'], //       'grey'      : ['\x1B[90m', '\x1B[39m'],
                            post: "\x1B[39m"
                        },
                        {
                            target: 'default',
                            mask: 'warn',
                            pre: '\x1B[33m',  // yellow
                            post: '\x1B[39m'
                        },
                        {
                            target: 'default',
                            mask: 'error',
                            pre: '\x1B[31m',  // red
                            post: '\x1B[39m'
                        },
                        {
                            target: 'default',
                            mask: 'success',
                            pre: '\x1B[32m',  // green
                            post: '\x1B[39m'
                        },
                        {
                            target: 'default',
                            mask: 'log',
                            pre: '\x1B[39m'  // default
                            //				post: '\x1B[39m'
                        },
                        {
                            mask: 'log',
                            tag: "stdout",
                            post_fmt_pre_msg: '\x1B[90m[console] \x1B[39m'  // default
                        },
                        {
                            tag: "stderr",
                            mask: 'error',
                            post_fmt_pre_msg: '\x1B[90m[console] \x1B[31m',  // red
                            post: '\x1B[39m'
                        }

                    ]
                }
            };

            var log_config = config;

            var logger_builtin_tags = [];

            if (!log_config) {
                // still don't have setup. use the defaults.
                log_config = {
                    tags: [],
                    targets: logger_default_target
                    // levels - use defaults
                };
            }

            if (log_config['nobuiltins'] === undefined) {
                //        	log_config.targets.push(logger_default_target);
                //		_.defaults(log_config.targets,logger_default_target);
                //		log_config.filters.concat(logger_builtin_filters);
                // log_config._tags = {};
                // var tags = {};
                // for(var n=0;n<log_config.tags.length;n++) {
                // 	tags[log_config.tags] = null;
                // }
                // for(var n=0;n<log_config.tags.length;n++) {
                // 	tags[log_config.tags] = null;
                // }
                dbg_logger("config: " + util.inspect(log_config, {depth: null}));
                if (log_config.tags)
                    log_config.tags = log_config.tags.concat(logger_builtin_tags);
                else
                    log_config.tags = logger_builtin_tags;
                if (!log_config['targets']) log_config.targets = {};
                if (!log_config.targets['default'])
                    log_config.targets['default'] = logger_default_target['default'];
                dbg_logger('here 3.2');
                log_config.tags = _.uniq(log_config.tags);
            }


            /**
             * converts a string version of a log level, like 'debug' to a uint32_t as defined by greaseLog
             * @private
             * @param str
             */
            var convert_level = function (str) {
                if (str && typeof str === 'string') {
                    var s = str.toLowerCase();
                    return opts.levels[s];
                }
            };

            var waitCBs = 0;

            var do_callback_on_complete = function () {
                if (waitCBs < 1) {
                    if (typeof donecb == 'function')
                        donecb();
                } else {
                    dbg_logger("do_callback_on_complete: " + waitCBs);
                }
            }

            var add_target = function (T) {
                waitCBs++;
                dbg_logger("add_target() ------------- : " + util.inspect(T));
                logger.addTarget(T, function (id, err) {
                    dbg_logger("in add_target() cb");
                    if (err) {
                        console.error("devjsLogger: Error adding filter: " + util.inspect(err));
                    } else {
                        // lookup to see if it has filters...
                        T.id = id;
                        if (T.filters && T.filters.length) {
                            for (var n = 0; n < T.filters.length; n++) {
                                var filter = T.filters[n];
                                //									T.filters[n].target
                                if (filter.target) // with the 'default' target in greaseLog we don't need a targetid
                                    delete filter.target;
                                filter.mask = convert_level(filter.mask);
                                filter.target = id;
                                dbg_logger("Add Filter---------->" + util.inspect(filter));
                                logger.addFilter(filter);
                            }
                        }
                    }
                    waitCBs--;
                    do_callback_on_complete();
                });
                dbg_logger("post add_target()");
            }

            ////////// setup grease-log


            var targs = Object.keys(log_config.targets);
            //	console.log("begin for loop...");
            for (var n = 0; n < targs.length; n++) {
                try {
                    switch (targs[n]) {
                        case 'default':  // handle 'default'
                            dbg_logger("its default...");
                            if (log_config.targets['default'].tty) {
                                dbg_logger("Modify default TTY setup");
                                if (log_config.targets['default'].tty === 'console') {
                                    next_targ = 0; // the 'console' target is the default target which is 0
                                    if (log_config.targets['default'].format)
                                        logger.modifyDefaultTarget({
                                            format: log_config.targets['default'].format
                                        });
                                    if (log_config.targets['default'].filters) {
                                        for (var p = 0; p < log_config.targets['default'].filters.length; p++) {
                                            var filter = log_config.targets['default'].filters[p];
                                            //									T.filters[n].target
                                            if (filter.target) // with the 'default' target in loggerLog we don't need a targetid
                                                delete filter.target;
                                            filter.mask = convert_level(filter.mask);
                                            dbg_logger("Add Filter---------->" + util.inspect(filter));
                                            logger.addFilter(filter);
                                        }
                                    }
                                } else {
                                    console.error("LOGGER: can't do the TTY target like that.");
                                    }
                            } else if (log_config.targets['default'].file) {
                                logger.modifyDefaultTarget(log_config.targets['default']);
                                dbg_logger("Modified default target to file: " + util.inspect(log_config.targets['default'].file));
                                if (log_config.targets['default'].filters) {
                                    for (var p = 0; p < log_config.targets['default'].filters.length; p++) {
                                        var filter = log_config.targets['default'].filters[p];
                                        //									T.filters[n].target
                                        if (filter.target) // with the 'default' target in loggerLog we don't need a targetid
                                            delete filter.target;
                                        filter.mask = convert_level(filter.mask);
                                        dbg_logger("Add Filter---------->" + util.inspect(filter));
                                        logger.addFilter(filter);
                                    }
                                }
                            }
                            break;
                        default:
                            var t = log_config.targets[targs[n]];
                            //dbg_logger("Add Target------------>");
                            //console.dir(log_config.targets[targs[n]]);
                            if (t.pty) {
                                if (t.pty == "new") {
                                    logger.createPTS(function (err, pty) {
                                        if (err) {
                                            console.error("devjsLogger: Error creating PTS: " + util.inspect(err));
                                        } else {
                                            fd = pty.fd;
                                            dbg_logger("devjsLogger: target '" + targs[n] + "' new PTY name: " + pty.path);
                                            t.tty = pty.fd;
                                            add_target(t);
                                        }
                                    });
                                } else {
                                    console.error("devjsLogger: Don't understand 'pty' " + t.pty + " target.");
                                }
                            } else
                                add_target(t);

                            break;
                    }
                } catch (e) {
                    console.error("LOGGER: Error in setting up log target: " + targs[n] + " --> " + util.inspect(e));
                }
            }

            var default_sink = logger.getDefaultSink();
            if(typeof default_sink === 'object') {
                if(typeof default_sink.unixDgram === 'string') {
                    fs.chmodSync(default_sink.unixDgram,0666);
                    dbg_logger("Set permissions on default_sink: " + default_sink.unixDgram);
                } else {
                    console.error("WEIRD defaultSink: ",default_sink);
                }
            } else {
                console.error("NO defaultSink.");
            }

            if (log_config['filterOut'] && log_config['filterOut'].length) {
                var mask = 0;
                for (var n = 0; n < log_config.filterOut.length; n++) {
                    mask |= convert_level(log_config.filterOut[n]);
                }
                logger.setGlobalOpts({levelFilterOutMask: mask});
            }


            //console.dir(process.stdout);
            //console.dir(logger);

            process.stdout.write = logger.stdoutWriteOverridePID;  // use the PID as origin
            process.stderr.write = logger.stderrWriteOverridePID;

            ////////// end setup.

        } else {
            dbg_logger("Client only")
            logger.onSinkFailed(function(){
                process.stdout.write = originalProcessStdoutWrite;
                process.stderr.write = originalProcessStderrWrite;
            });
            originalProcessStdoutWrite = process.stdout.write
            originalProcessStderrWrite = process.stderr.write
            process.stdout.write = logger.stdoutWriteOverridePID;  // use the PID as origin
            process.stderr.write = logger.stderrWriteOverridePID;

            dbg_logger("Note: Grease took over stdout/stderr in node.js.")

        } //// end if !client_only

        GLOBAL.log = logger;
        return logger;
    }
    console.error("unreachable failure.");
};

var isCloud = function() {
    return !global.hasOwnProperty('dev$');
};


if(!GLOBAL.log) {
    module.exports = function(opts,config,cb) {

        if((opts && (opts.client_only || opts._force_grease)) || !isCloud()) {
            try {
                console.log("require grease-log-client");
                grease = require('grease-log-client');

                if(!opts || typeof opts != 'object')
                    opts = {};
                if(!opts.levels) {
                    opts.levels = LEVELS_default;
                }
                opts.debug = true
                console.log("setup() grease-log-client")
                return setup(opts,config,cb);
            } catch(e) {
                console.error("** Can't load grease-log-client. Going to fallback. **");
                console.error("   Failure @ " + util.inspect(e));
                return setup_fallback();
            }
        } else {
            return setup_fallback(opts,config,cb);
        }
    };
} else {
    module.exports = function(opts,config,donecb) {
        if (typeof donecb == 'function') {
            donecb();
        }
        return GLOBAL.log;
    }
}
