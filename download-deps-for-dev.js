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

var fs = require('fs')
var path = require('path')
const { execSync } = require('child_process')
var data = fs.readFileSync(path.join(__dirname,'Godeps','Godeps.json'))

var capRepoRe = /^([^\/]+)\/([^\/]+)(?:\/([^\/]+)){0,1}.*$/;

var importmap = {}

var getArgs = function() {
    var n = 0
    while(n < 20) {    
        if (process.argv[n] == __filename) {
            break
        }
        n++
    }
    if (n >= 20) {
        console.error("Error processing command line args.")
        process.exit(-1)
    }
    n++
    var z = process.argv.slice(n)
    var ret = {}
    for (var q=0;q<z.length;q++) {
        if (ret[z[q]]) {
            ret[z[q]]++
        } else {
            ret[z[q]] = 1
        }
    }
    return ret
} 

console.log("argv ", process.argv)
var argz = getArgs()
console.log("res=",argz)

if (data != null) {
    var json = JSON.parse(data)
    if (typeof json == 'object'
     && json.Deps && Array.isArray(json.Deps)) {
        // console.log("got it")
        var p = null
        var getpath = null
        for (var x=0;x<json.Deps.length;x++) {
            if (json.Deps[x].ImportPath) {
                p = null
                p = capRepoRe.exec(json.Deps[x].ImportPath)
                if (p && p.length > 2 && p[1] && p[2]) {
                    getpath = p[1] + "/" + p[2]
                    if (p.length > 3 && p[3]) {
                        getpath = getpath + "/" + p[3]
                    }
                    if (!importmap[getpath]) {
                        if (argz["-u"]) {
                            console.log("go get -u",getpath)
                        } else {
                            console.log("go get",getpath)
                        }
                        try {
                            let output = ""
                            if (argz["-u"]) {
                                output = execSync("go get -u "+ getpath)
                            } else {
                                output = execSync("go get "+ getpath)
                            }
                            console.log(">>",output.toString('utf8'))                                
                        } catch(e) {
                            console.error("error>>",e.toString('utf8'))
                        }
                        importmap[getpath] = 1    
                    }
                } else {
                    console.error("Error: bad line in Godeps.json:",json.Deps[x].ImportPath)
                }
            } else {
                console.error("Error: an entry in Godeps.json Deps has no ImportPath")
            }
        }
    } else {
        console.error("Malformed Godeps.json?")
    }
} else {
    console.error("No data in Godeps.json:",data)
}