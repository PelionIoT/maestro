package templates

// Copyright (c) 2018, Arm Limited and affiliates.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const MAX_FILE_SIZE = 1024 * 1024 // one 1MB - if a config file is bigger than that then its rediculous
const DEFAULT_OUTPUT_FILE_PERMS = 0664

// our work ticket
type FileTemplateTicket struct {
	opname       string
	dict         *TemplateVarDictionary
	sourcepath   string
	targetpath   string
	sourcestring string
	checksum     string // use a sha256 sum right now
	//	original string
	output   []byte
	Filemode os.FileMode
}

func NewFileOp(name string, sourcepath string, sourcestring string, targetpath string, dict *TemplateVarDictionary) (ret *FileTemplateTicket) {
	ret = new(FileTemplateTicket)
	ret.opname = name
	ret.sourcepath = sourcepath
	ret.sourcestring = sourcestring
	ret.targetpath = targetpath
	ret.dict = dict
	ret.Filemode = DEFAULT_OUTPUT_FILE_PERMS
	return
}

func (this *FileTemplateTicket) GetOpName() string {
	return this.opname
}

func (this *FileTemplateTicket) ProcessTemplateFile() (err error) {
	var data string
	// use template from string or file?
	if len(this.sourcepath) < 1 {
		data = this.sourcestring

		if len(data) < 1 {
			templerr := &TemplateError{
				TemplateName: this.opname,
				Code:         TEMPLATEERROR_NO_TEMPLATE,
				ErrString:    "Template string / data is empty.",
			}
			err = templerr
			return
		}
	} else {
		// check file size
		fi, err2 := os.Stat(this.sourcepath)
		if err2 != nil {
			err = err2
			return
		}
		// check the size
		size := fi.Size()
		if size > MAX_FILE_SIZE {
			templerr := &TemplateError{
				TemplateName: this.opname,
				Code:         TEMPLATEERROR_BAD_FILE,
				ErrString:    "Template file too large: " + this.sourcepath,
			}
			err = templerr
			return
		}
		// read the file in

		b, err2 := ioutil.ReadFile(this.sourcepath) // just pass the file name
		if err != nil {
			return
		}
		// fmt.Println(b) // print the content as 'bytes'
		data = string(b) // convert content to a 'string'

	}

	data = this.dict.Render(data)

	this.output = []byte(data)

	hasher := sha256.New()
	hasher.Write(this.output)

	this.checksum = fmt.Sprintf("%x", hasher.Sum(nil)) // print the content as a 'string'

	return
}

// Generates the file only if the file has changed.

func (this *FileTemplateTicket) MaybeGenerateFile() (err error, wrotefile bool, current string) {
	var f *os.File
	newfile := false
	f, err = os.Open(this.targetpath)
	if err != nil {
		//		if e, ok := err.(*os.PathError); ok && e.Error == os.ENOENT {
		if os.IsNotExist(err) {
			err = nil
			newfile = true
		} else {
			return
		}
		// actual, ok := err.(*os.PathError)
		// if ok {
		// 	DEBUG_OUT("Error: %+v\n",actual)
		// }
	}

	if !newfile {
		hasher := sha256.New()
		defer f.Close()
		if _, err = io.Copy(hasher, f); err != nil {
			return
		}
		current = fmt.Sprintf("%x", hasher.Sum(nil))
	}

	if newfile || current != this.checksum {
		wrotefile = true

		// get dir portion of the path
		dir := filepath.Dir(this.targetpath)
		if len(dir) > 1 && dir != "." && dir != ".." && dir != "/" {
			// check to make sure path exists...
			info, err2 := os.Lstat(dir)
			if err2 != nil {
				if os.IsNotExist(err2) {
					err = os.MkdirAll(dir, this.Filemode)
					if err != nil {
						return
					}
				}
			} else if !info.IsDir() {
				err = errors.New("path is not directory")
				return
			}
			// else... let errors fall where they may on WriteFile
		}

		err = ioutil.WriteFile(this.targetpath, this.output, this.Filemode)
	}

	current = this.checksum // return the new checksum

	return
}
