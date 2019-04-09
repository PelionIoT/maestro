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

var run = function(){
	var n = 0;
	setInterval(function(){
		var ok = false
		if(dev$) {
			ok = true
		}
		console.log("**TEST " + n + "** (basic-devjs.js) dev$",ok,process.argv[2]);
		console.error("**TEST (stderr)" + n + "** (basic-devjs.js) dev$",ok,process.argv[2]);
		n++;
	},1000)	
}

// var readstr = ""
// var config = null;
// var config_re = /(.*)ENDCONFIG/;

// process.stdin.on('data', function(chunk) {
// 	readstr += chunk
//     // lines = chunk.split("\n");

//     // lines[0] = lingeringLine + lines[0];
//     // lingeringLine = lines.pop();

//     // lines.forEach(processLine);
// });

// process.stdin.on('end', function(){
// 	var m = config_re.exec(readstr);

// 	console.log("match",m);

// 	if(m && m.length > 1) {
// 		config = m[1];
// 		console.log("Got config:",config)
// 		console.error("Got config (stderr):",config)
// //		console.log("!!OK!!")
// 		run();
// 	} else {
// 		console.log("(console.log)Error - did not get a config!!\nxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\nxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n");
// 		console.error("Error - did not get a config!!\nxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\nxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n");
// 	}

// 	// if (readstr == "ENDCONFIG") {
// 	// 	console.log("!!OK!!") // send OK string to maestro waiting for success message		
// 	// 	run()
// 	// }
// });



process.on('exit', function(code) {
	console.log("test-node-read-config-stdin exiting",code)
	console.error("test-node-read-config-stdin exiting (stderr)",code)
})
