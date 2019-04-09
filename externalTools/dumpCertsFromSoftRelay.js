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

// dumps certs and other data from a soft-relay XXXXX-config.json file

var fs = require('fs')

var fname = process.argv[2]

console.log("Reading file",fname,"...");

fs.readFile(fname, 'utf8',function(err, data) {
  if (err) throw err;
  try {
      var data = JSON.parse(data)
  } catch(e) {
      console.error("Bad JSON in file",e);
      process.exit(1)
  }

    console.log("Relay ID:",data.relayID);
    console.log("Client SSL:");
    console.log("  Key ")
    console.log(">>>>>>>>>>>>>>>>>>>>>>>")
    console.log(data.ssl.client.key)
    console.log("<<<<<<<<<<<<<<<<<<<<<<<")
    console.log("  Certificate ");
    console.log(">>>>>>>>>>>>>>>>>>>>>>>")
    console.log(data.ssl.client.certificate)
    console.log("<<<<<<<<<<<<<<<<<<<<<<<")

    console.log("Server SSL:");
    console.log("  Key ")
    console.log(">>>>>>>>>>>>>>>>>>>>>>>")
    console.log(data.ssl.server.key)
    console.log("<<<<<<<<<<<<<<<<<<<<<<<")
    console.log("  Certificate ");
    console.log(">>>>>>>>>>>>>>>>>>>>>>>")
    console.log(data.ssl.server.certificate)
    console.log("<<<<<<<<<<<<<<<<<<<<<<<")

    console.log("CA SSL:");
    console.log("  CA ")
    console.log(">>>>>>>>>>>>>>>>>>>>>>>")
    console.log(data.ssl.ca.ca)
    console.log("<<<<<<<<<<<<<<<<<<<<<<<")
    console.log("  Intermediate ");
    console.log(">>>>>>>>>>>>>>>>>>>>>>>")
    console.log(data.ssl.ca.intermediate)
    console.log("<<<<<<<<<<<<<<<<<<<<<<<")


});
