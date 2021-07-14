package maestro

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
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PelionIoT/maestro/debugging"
	"github.com/PelionIoT/maestro/log"
	"github.com/PelionIoT/maestro/maestroConfig"
	"github.com/PelionIoT/maestro/maestroutils"
	"github.com/PelionIoT/maestro/tasks"
	"github.com/PelionIoT/maestroSpecs"
	"github.com/cavaliercoder/grab"
)

var mgrImagePath string
var mgrScratchPath string

// internal image operation
// data
type imageop struct {
	client   *grab.Client
	resp     *grab.Response
	req      *grab.Request
	taskId   string
	imageDef maestroSpecs.ImageDefinition
	// url string
	// checksum string
	tempDir          string // used for downloading
	finalDir         string
	downloadFilepath string // the path to the downloaded file
	appName          string
	version          string
}

var imgInstance *ImageManagerInstance

const (
	shutdown_manager = iota
)

func ImageManagerGetInstance() *ImageManagerInstance {
	if imgInstance == nil {
		imgInstance = new(ImageManagerInstance)
		imgInstance.ticker = time.NewTicker(500 * time.Millisecond)
		imgInstance.controlChan = make(chan uint32, 100)
	}
	return imgInstance
}

func init() {
}

// quick DB - FIXME - this needs to use the storage driver

var imageDB sync.Map // appName to StoredImageEntry

type StoredImageEntry struct {
	appName    string
	version    string
	location   string // comes from 'finalDir'
	entrypoint string // command to start it
}

func newStoredImageEntry() (ret *StoredImageEntry) {
	ret = new(StoredImageEntry)
	return
}

func (this *ImageManagerInstance) registerImage(op *imageop) {
	image := newStoredImageEntry()
	image.appName = op.appName
	image.version = op.version
	image.location = op.finalDir
	// FIXME what to do about entrypoint??
	imageDB.Store(op.appName, image)
	debugging.DEBUG_OUT("ImageManagerInstance stored image \"%s\"\n", op.appName)
}

func (this *ImageManagerInstance) LookupImage(appname string) (ret *StoredImageEntry, ok bool) {
	val, ok := imageDB.Load(appname)
	if ok {
		ret = val.(*StoredImageEntry)
	}
	return
}

func InitImageManager(scratchPath string, imagePath string) (err error) {
	mgrImagePath = imagePath
	mgrScratchPath = scratchPath

	err = os.MkdirAll(scratchPath, 0700)
	if err != nil {
		log.MaestroErrorf("Failed to create scratch path directory: %s\n", err.Error())
	}
	err = os.MkdirAll(imagePath, 0700)
	if err != nil {
		log.MaestroErrorf("Failed to create scratch path directory: %s\n", err.Error())
	}

	tasks.RegisterHandler("image", ImageManagerGetInstance())

	return
}

func (this *ImageManagerInstance) initImageDownload(task *tasks.MaestroTask) (err *maestroSpecs.APIError) {
	requestedOp, ok := task.Op.(*maestroSpecs.ImageOpPayload)
	if ok {
		debugging.DEBUG_OUT(" IMAGE>>>initImageDownload()  ok - have image op\n")
		def := requestedOp.GetImageDefinition()
		if def != nil {
			imgop := new(imageop)
			imgop.taskId = requestedOp.GetTaskId()
			imgop.imageDef = requestedOp.GetImageDefinition()
			imgop.appName = requestedOp.GetAppName()
			imgop.version = requestedOp.GetVersion()

			debugging.DEBUG_OUT(" IMAGE>>>Got image operation definition %+v\n", def)

			// make the parent directory for temp directories, just in case not there.
			os.MkdirAll(mgrScratchPath, 0700)

			dirname, err2 := ioutil.TempDir(mgrScratchPath, def.GetJobName())

			if err2 == nil {
				debugging.DEBUG_OUT(" IMAGE>>>Setting up temporary folder %s\n", dirname)
				imgop.tempDir = dirname
			} else {
				debugging.DEBUG_OUT(" IMAGE>>>Failed to setup up temporary dir %s\n", err2)
				err = new(maestroSpecs.APIError)
				err.HttpStatusCode = 500
				err.ErrorString = "Failed to create temporary directory"
				err.Detail = err2.Error()
				return
			}

			if imgop.imageDef.GetSize() > 0 {
				// check disk space availability is Size if provided
				stats := maestroutils.DiskUsageAtPath(imgop.tempDir)
				if stats.Avail < imgop.imageDef.GetSize()+maestroConfig.ConfigGetMinDiskSpaceScratch() {
					debugging.DEBUG_OUT(" IMAGE>>>Disk space low in temp dir %s - can't move image\n", imgop.tempDir)
					log.MaestroErrorf("IMAGE>>>Disk space low in temp dir %s - can't move image\n", imgop.tempDir)
					err = new(maestroSpecs.APIError)
					err.HttpStatusCode = 507 // "Insufficient Storage"
					err.ErrorString = "Low disk space on scratch"
					err.Detail = "dir: " + imgop.tempDir
					return
				} else {
					debugging.DEBUG_OUT(" IMAGE>>>Disk space is OK.\n")
				}
			}

			// set the final destination path
			imgop.finalDir = mgrImagePath + "/" + imgop.imageDef.GetJobName()

			this.imageops.Store(task.Id, imgop)
		} else {
			err = new(maestroSpecs.APIError)
			err.HttpStatusCode = 400
			err.ErrorString = "no image definition"
			err.Detail = "in initImageDownload()"
		}
	} else {
		debugging.DEBUG_OUT(" IMAGE>>>initImageDownload()  ERROR not an image op\n")
		err = new(maestroSpecs.APIError)
		err.HttpStatusCode = 500
		err.ErrorString = "op was not an ImageOperation"
		err.Detail = "in initImageDownload()"
	}
	return
}

// download thread loop
func (this *ImageManagerInstance) doDownload(task *tasks.MaestroTask) {

	imgp, ok := this.imageops.Load(task.Id)
	if ok {
		img := imgp.(*imageop)
		img.client = grab.NewClient()
		var reqErr error
		img.req, reqErr = grab.NewRequest(img.tempDir, img.imageDef.GetURL())

		if reqErr != nil {
			log.MaestroErrorf("imageManager: Error on download request: %s\n", reqErr.Error())
			tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 500, ErrorString: "Error on creating http request.", Detail: reqErr.Error()})
			return
		}

		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()

		// start download
		log.MaestroInfof("IMAGE Downloading %v\n", img.req.URL())
		resp := img.client.Do(img.req)
		if resp != nil && resp.HTTPResponse != nil {
			debugging.DEBUG_OUT("IMAGE>>> download response: %v\n", resp.HTTPResponse.Status)
		} else {
			tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 0, ErrorString: "No server.", Detail: "Request failed"})
			return
		}

	Loop:

		for {
			select {
			case <-t.C:
				debugging.DEBUG_OUT("  transferred %v / %v bytes (%.2f%%)\n",
					resp.BytesComplete(),
					resp.Size,
					100*resp.Progress())

			case <-resp.Done:
				debugging.DEBUG_OUT("IMAGE>>>> DOWNLOAD DONE: Request complete: %+v\n", resp)

				img.downloadFilepath = resp.Filename
				// download is complete
				break Loop
			}
		}

		tasks.IterateTask(task.Id, nil)
	} else {
		debugging.DEBUG_OUT("IMAGE>>> doDownload() --> Could not find image op: %s\n", task.Id)
		tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 500, ErrorString: "Image not in table.", Detail: "task id:" + task.Id})
	}

}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

type unzipCB func(err error)

func unzipArchive(src, dest string, cb unzipCB) {
	r, err := zip.OpenReader(src)
	if err != nil {
		cb(err)
		return
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.MaestroErrorf("Image manager: problem closing Zip archive: %s\n", err.Error())
			//            panic(err)
		}
	}()

	//    os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			log.MaestroErrorf("Image Manager: Error opening zip archive: %s --> %s\n", f.Name, err.Error())
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				log.MaestroErrorf("Image manager: problem closing Zip archive: %s\n", err.Error())
				//            	cb(err)
			}
		}()

		var path string
		path = filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			debugging.DEBUG_OUT("IMAGE>>> extract: topTree %s %+v\n", f.Name, f.FileInfo())
			os.MkdirAll(path, f.Mode())
		} else {
			debugging.DEBUG_OUT("IMAGE>>> extract: path %s %s\n", path, f.Name)
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				log.MaestroErrorf("Image Manager: Error on extracting file from archive: %s --> %s\n", path, err.Error())
				return err
			}
			defer func(p string) {
				if err := f.Close(); err != nil {
					log.MaestroErrorf("Image manager: problem closing output file for archive %s: %s\n", p, err.Error())
					//                    panic(err)
				}
			}(path)

			_, err = io.Copy(f, rc)
			if err != nil {
				log.MaestroErrorf("Image Manager: Error on extracting file from archive (io.Copy): %s --> %s\n", path, err.Error())
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		debugging.DEBUG_OUT("IMAGE>>> EXTRACT\n")
		if err != nil {
			log.MaestroErrorf("Image Manager: Error on extracting zip): %s\n", err.Error())
			cb(err)
			return
		}
	}

	cb(nil)
	return
}

// this unzips the archive, and places it into it's destination directory
func (this *ImageManagerInstance) relocateImage(task *tasks.MaestroTask) {
	imgp, ok := this.imageops.Load(task.Id)
	if ok {
		img := imgp.(*imageop)
		// we need to determine the file name for the downloaded image. It should be a single compressed file.
		exists, err := pathExists(img.finalDir)
		if exists {
			debugging.DEBUG_OUT("IMAGE>>> Image manager will replace image at %s for Job %s\n", img.finalDir, img.imageDef.GetJobName())
			log.MaestroWarnf("Image manager will replace image at %s for Job %s\n", img.finalDir, img.imageDef.GetJobName())
			err = os.RemoveAll(img.finalDir)
			if err != nil {
				debugging.DEBUG_OUT("IMAGE>>> Image Manager: Error removing existing directory: %s - Failing Task (id:%s)\n", img.finalDir, task.Id)
				log.MaestroErrorf("Image Manager: Error removing existing directory: %s - Failing Task (id:%s)\n", img.finalDir, task.Id)
				tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 500, ErrorString: "Failed to remove existing path", Detail: "Path:" + img.finalDir})
				return
			}
		}
		debugging.DEBUG_OUT2("IMAGE>>> relocateImage #2\n")
		err = os.MkdirAll(img.finalDir, 0755) //os.ModePerm)
		if err != nil {
			debugging.DEBUG_OUT("IMAGE>>>  Image Manager: Failed to create directory for image: %s - Failing Task (id:%s)\n", img.finalDir, task.Id)
			log.MaestroErrorf("Image Manager: Failed to create directory for image: %s - Failing Task (id:%s)\n", img.finalDir, task.Id)
			tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 500, ErrorString: "Failed to create image directory", Detail: "Path:" + img.finalDir})
			return
		}
		debugging.DEBUG_OUT2("IMAGE>>> relocateImage #3\n")
		// directory created.

		// FIXME FIXME FIXME - hack until APIs in cloud finalized
		if true { //img.imageDef.GetImageType() == maestroSpecs.IMAGE_TYPE_ZIP || img.imageDef.GetImageType() == maestroSpecs.IMAGE_TYPE_DEVICEJS_ZIP
			tasks.MarkTaskStepAsExecuting(task.Id)
			// spawn a routine to unpack the archive
			debugging.DEBUG_OUT2("IMAGE>>> relocateImage #4 unzip\n")
			go unzipArchive(img.downloadFilepath, img.finalDir, func(err error) {
				if err != nil {
					log.MaestroErrorf("Image Manager: Failed to unpack arhive to: %s - Failing Task (id:%s)\n", img.finalDir, task.Id)
					tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 500, ErrorString: "Failed to unpack archive", Detail: err.Error()})
				} else {
					log.MaestroInfof("Image Manager: Archive unpacked at %s\n", img.finalDir)
					tasks.IterateTask(task.Id, nil)
					this.registerImage(img)
				}
			})
		} else {
			debugging.DEBUG_OUT2("IMAGE>>> relocateImage FAILED test ~310\n")
			// don't forget to cleanup
			tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 500, ErrorString: "Unsupported Image type", Detail: "task id:" + task.Id})
		}
	} else {
		debugging.DEBUG_OUT("IMAGE>>> doDownload() --> Could not find image op: %s\n", task.Id)
		tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 500, ErrorString: "Image not in table.", Detail: "task id:" + task.Id})
	}

}

func (this *ImageManagerInstance) cleanupTrash(task *tasks.MaestroTask) {
	imgp, ok := this.imageops.Load(task.Id)
	if ok {
		img := imgp.(*imageop)
		err := os.RemoveAll(img.tempDir)
		if err != nil {
			log.MaestroErrorf("Image Manager: Error attempting to remove temp dir: %s\n", img.tempDir)
		}
	} else {
		debugging.DEBUG_OUT("IMAGE>>> cleanupTrash() --> Could not find image op: %s\n", task.Id)
		tasks.FailTask(task.Id, &maestroSpecs.APIError{HttpStatusCode: 500, ErrorString: "Image not in table.", Detail: "task id:" + task.Id})
	}

}

func (this *ImageManagerInstance) ValidateTask(task *tasks.MaestroTask) (err error) {
	if task.Op.GetType() == maestroSpecs.OP_TYPE_IMAGE {
		requestedOp, ok := task.Op.(*maestroSpecs.ImageOpPayload)
		if ok {
			switch requestedOp.GetOp() {
			case maestroSpecs.OP_ADD:
				def := requestedOp.GetImageDefinition()
				if def != nil {
					url := def.GetURL()
					if len(url) < 1 || !IsValidDownloadURL(url) {
						apierr := new(maestroSpecs.APIError)
						apierr.HttpStatusCode = 406
						apierr.ErrorString = "Invalid image properties"
						apierr.Detail = "Bad URL"
						err = apierr
					}
					if def.GetImageType() != maestroSpecs.IMAGE_TYPE_ZIP || def.GetImageType() != maestroSpecs.IMAGE_TYPE_DEVICEJS_ZIP {
						apierr := new(maestroSpecs.APIError)
						apierr.HttpStatusCode = 406
						apierr.ErrorString = "Unsupported image type"
						apierr.Detail = "Only support \"zip\" and \"devicejs_zip\" currently"
						err = apierr
					}
				}
			case maestroSpecs.OP_REMOVE:

			case maestroSpecs.OP_UPDATE:

			default:
				apierr := new(maestroSpecs.APIError)
				apierr.HttpStatusCode = 406
				apierr.ErrorString = "Unknown op type for image op"
				apierr.Detail = "Unknown op type " + requestedOp.GetOp()
				err = apierr
			}

		} else {
			apierr := new(maestroSpecs.APIError)
			apierr.HttpStatusCode = 500
			apierr.ErrorString = "Mismatch task type"
			apierr.Detail = "internal error imageManager @ValidateTask (2)"
			err = apierr
		}
	} else {
		apierr := new(maestroSpecs.APIError)
		apierr.HttpStatusCode = 500
		apierr.ErrorString = "Mismatch task type"
		apierr.Detail = "internal error imageManager @ValidateTask"
		err = apierr
	}
	return

}

// implements TaskHandler interface:
func (this *ImageManagerInstance) SubmitTask(task *tasks.MaestroTask) (err error) {
	debugging.DEBUG_OUT("IMAGE>>>ImageManagerInstance.SubmitTask() entry>\n")
	if task.Op.GetType() == maestroSpecs.OP_TYPE_IMAGE {
		switch task.Op.GetOp() {
		case "add":
			switch task.Step {
			case 0:
				err := this.initImageDownload(task)
				if err != nil {
					tasks.FailTask(task.Id, err)
				} else {
					tasks.IterateTask(task.Id, nil)
				}
			case 1:
				debugging.DEBUG_OUT("IMAGE>>>> doing doDownload()\n")
				tasks.MarkTaskStepAsExecuting(task.Id)
				go this.doDownload(task)
			// TODO: validate checksum
			case 2:
				debugging.DEBUG_OUT("IMAGE>>>> doing relocateImage()\n")
				this.relocateImage(task)
			case 3:
				debugging.DEBUG_OUT("IMAGE>>>> doing cleanupTrash()\n")
				this.cleanupTrash(task)
				tasks.CompleteTask(task.Id)
			}
		default:
			debugging.DEBUG_OUT("IMAGE>>>ImageManagerInstance.SubmitTask() unknown Op:%s\n", task.Op.GetOp())

		}
	} else {
		debugging.DEBUG_OUT("IMAGE>>>ImageManagerInstance.SubmitTask() not correct Op type\n")
		err := new(maestroSpecs.APIError)
		err.HttpStatusCode = 500
		err.ErrorString = "op was not an ImageOperation"
		err.Detail = "in initImageDownload()"
		log.MaestroErrorf("imageManager got missplaced task %+v\n", task)
	}
	return
}

func ShutdownImageManager() {
	//	i := GetInstance()

}
