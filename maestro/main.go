package main

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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	//	"reflect"
	"net/http"
	"time"

	"github.com/armPelionEdge/httprouter"
	. "github.com/armPelionEdge/maestro"
	"github.com/armPelionEdge/maestro/configMgr"
	"github.com/armPelionEdge/maestro/debugging"
	Log "github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/logconfig"
	"github.com/armPelionEdge/maestro/servicectl"
	"github.com/armPelionEdge/maestro/maestroConfig"
	"github.com/armPelionEdge/maestro/maestroutils"
	"github.com/armPelionEdge/maestro/mdns"
	"github.com/armPelionEdge/maestro/networking"
	"github.com/armPelionEdge/maestro/processes"
	"github.com/armPelionEdge/maestro/storage"
	"github.com/armPelionEdge/maestro/sysstats"
	"github.com/armPelionEdge/maestro/tasks"
	maestroTime "github.com/armPelionEdge/maestro/time"
	"github.com/armPelionEdge/maestro/watchdog"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestroSpecs/templates"
	"github.com/op/go-logging"

	// Platforms
	"github.com/armPelionEdge/maestro/platforms"
	// platform_rp200 "github.com/armPelionEdge/maestro/platforms/rp200"
	// platform_rp200_edge "github.com/armPelionEdge/maestro/platforms/rp200_edge"
	// platform_wwrelayA10 "github.com/armPelionEdge/maestro/platforms/wwrelayA10"
	// platform_softRelay "github.com/armPelionEdge/maestro/platforms/softRelay"
	// platform_testplatform "github.com/armPelionEdge/maestro/platforms/testplatform"
	_ "net/http/pprof"
)

var log = logging.MustGetLogger("maestro")

var (
	NumWorkers = os.Getenv("MAESTRO_LOG_WORKERS")
)

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func main() {

	if len(NumWorkers) < 1 {
		fmt.Printf("MAESTRO_LOG_WORKERS not set.\n")
	}

	log.Info("maestro starting.")

	// Debug function that runs on USR1 signal
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGUSR1)
		for {
			<-c
			fmt.Println("[*] SIGUSR1 received")

			fmt.Println("    maestro version: maestroutils.Version()")
			fmt.Println("    meta vars: {{VARNAME}} = [[VALUE]]")
			dict := maestroConfig.GetGlobalConfigDictionary()
			for varname, val := range dict.Map {
				fmt.Printf("        {{%s}} = [[%s]]\n", varname, val)
			}
		}
	}()

	configFlag := flag.String("config", "./maestro.config", "Config path")
	dumpMetaVars := flag.Bool("dump_meta_vars", false, "Dump config file meta variables only")
	versionFlag := flag.Bool("version", false, "Dump version information")
	debugServerFlag := flag.Bool("debug_loopback", true, "Start a debug loopback on http://127.0.0.1:6060")
	debugMemory := flag.Bool("debug_mem", true, "Debugging memory stats")
	flag.Parse()

	debugging.DebugPprof(*debugServerFlag)
	if *debugMemory {
		debugging.DumpMemStats()
		go debugging.RuntimeMemStats(300)
	}

	if *versionFlag {
		s := maestroutils.Version()
		fmt.Printf("%s\n", s)
		fmt.Printf("%s\n", logconfig.GetLogLibVersion())
		os.Exit(0)
	}

	if configFlag != nil {
		debugging.DEBUG_OUT("config file:", *configFlag)
	}

	// Initialization starts off with reading in the entire config file,
	// which also creates and populated the config macro variable dictionary,
	// which is used in different parts of the configs

	config := new(maestroConfig.YAMLMaestroConfig)
	err := config.LoadFromFile(*configFlag)

	if err != nil {
		log.Errorf("Critical error. Config file parse failed --> %s\n", err.Error())
		os.Exit(1)
	}

	for _, platform := range config.PlatformReaders {
		logger := Log.NewPrefixedLogger("platform_reader " + platform.Platform)
		if len(platform.Params) > 0 {
			platforms.SetPlatformReaderOpts(platform.Params, platform.Platform, platform.Opts, logger)
		}
		err := platforms.ReadWithPlatformReader(maestroConfig.GetGlobalConfigDictionary(), platform.Platform, platform.Opts, logger)
		// // Is the string a path? If so - load as a plugin
		// if strings.HasPrefix(platform.Platform,"plugin:") {
		// 	s := strings.Split(platform.Platform,":")
		// 	err = platforms.ReadWithPlatformReader(maestroConfig.GetGlobalConfigDictionary(),s[1])
		// } else {
		// 	logger := Log.NewPrefixedLogger("platform_reader "+platform.Platform)
		// 	// else - see if its a known internal reader
		// 	switch platform.Platform {
		// 	case "testplatform":
		// 		err = platform_testplatform.GetPlatformVars(maestroConfig.GetGlobalConfigDictionary(), logger)
		// 	case "rp200":
		// 		err = platform_rp200.GetPlatformVars(maestroConfig.GetGlobalConfigDictionary(), logger)
		// 	case "rp200_edge":
		// 		err = platform_rp200_edge.GetPlatformVars(maestroConfig.GetGlobalConfigDictionary(), logger)
		// 	case "wwrelayA10":
		// 		err = platform_wwrelayA10.GetPlatformVars(maestroConfig.GetGlobalConfigDictionary(), logger)
		// 	case "softRelay":
		// 		err = platform_softRelay.GetPlatformVars(maestroConfig.GetGlobalConfigDictionary(), logger)
		// 	default:
		// 		log.Errorf("Unknown plaform referred to: %s  Skipping.\n",platform.Platform)
		// 	}
		// }
		if err != nil {
			log.Errorf("Error reading platform information: %s\n", err.Error())
		}
	}

	config.FinalizeConfig()

	if *dumpMetaVars {
		fmt.Printf(" Format: {{VARNAME}} = [[VALUE]]\n\n")
		dict := maestroConfig.GetGlobalConfigDictionary()
		for varname, val := range dict.Map {
			fmt.Printf("{{%s}} = [[%s]]\n", varname, val)
		}
		os.Exit(0)
	}

	// do debug stuff at start
	if config.DebugOpts != nil {
		// write out PID file if
		if len(config.DebugOpts.PidFile) > 0 {
			var err error
			var f *os.File
			var pidstr string
			if config.DebugOpts.KeepPids {
				// If the file doesn't exist, create it, or append to the file
				f, err = os.OpenFile(config.DebugOpts.PidFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			} else {
				f, err = os.OpenFile(config.DebugOpts.PidFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
			}
			if err == nil {
				if config.DebugOpts.PidFileDates {
					t := time.Now()
					st := t.Format("2006-01-02T15:04:05.999999-07:00")
					pidstr = fmt.Sprintf("%d : %s\n", os.Getpid(), st)
				} else {
					pidstr = fmt.Sprintf("%d\n", os.Getpid())
				}
				if _, err = f.Write([]byte(pidstr)); err != nil {
					fmt.Fprintln(os.Stderr, "Error writing to pid file:", err)
				}
				if err = f.Close(); err != nil {
					fmt.Fprintln(os.Stderr, "Error closing pid file:", err)
				}
			} else {
				fmt.Fprintln(os.Stderr, "Error opening pid file:", err)
			}
		}
	}

	if err == nil {
		debugging.DEBUG_OUT("Loaded config file")
	} else {
		fmt.Fprintln(os.Stderr, "Error loading config:", err)
	}

	//	fmt.Println("Using sink socket:",config.ApiUnixDgramSocket);
	fmt.Println("Using API socket:", config.HttpUnixSocket)

	var waitGroup sync.WaitGroup

	// start Watchdog ASAP
	if config.Watchdog != nil {
		err := watchdog.LoadWatchdog(config.Watchdog)
		if err != nil {
			log.Errorf("Failed to load watchdog!! %s\n", err.Error())
		}
	}

	// Start up storage driver
	dbpath := config.GetDBPath()
	fmt.Printf("Opening config database:%s\n", dbpath)
	DB, err := storage.InitStorage(dbpath)

	if err != nil {
		log.Errorf("!!! ERROR - storage driver failed! %s\n", err.Error())
	}

	/*********************************************/
	/*             Logger setup                  */
	/*********************************************/
	// We've completed the minimal setup required to bring up logging.
	// The logger deals with all syslog() calls, all kernel logs,
	// and all stdout / stderr from processes Maestro starts
	// This processes the logger config, and sets up where logs should go
	logerr := logconfig.InitLogManager(config)
	if logerr != nil {
		Log.MaestroErrorf("Error starting log subsystem! %s\n", logerr.Error())
		log.Errorf("Error starting log subsystem! %s\n", logerr.Error())
	}

	servicectlerr := servicectl.InitServicectlManager(config)
	if servicectlerr != nil {
		Log.MaestroErrorf("Error starting servicectl subsystem! %s\n", servicectlerr.Error())
		log.Errorf("Error starting servicectl subsystem! %s\n", servicectlerr.Error())
	}

	tasks.InitTaskManager()
	processes.InitProcessMgmt(config.Processes)

	if DB != nil {
		// register all Job's in the existing config database.
		DB.ForEachContainerTemplate(func(templ maestroSpecs.ContainerTemplate) {
			err := processes.RegisterContainer(templ)
			if err != nil {
				log.Errorf("Container templ [%s] failed to register: %s\n", templ.GetName())
			} else {
				debugging.DEBUG_OUT("Container templ found in DB: %+v\n", templ)
			}
		})

		// register all Job's in the existing config database.
		DB.ForEachJob(func(job maestroSpecs.JobDefinition) {
			err := processes.RegisterJob(job)
			if err != nil {
				log.Errorf("Job [%s] failed to register job: %s\n", job.GetJobName())
			} else {
				debugging.DEBUG_OUT("Job found in DB: %+v\n", job)
			}
		})
	}

	/*********************************************/
	/*             System stats                  */
	/*********************************************/
	if config.SysStats != nil {
		sysStatMgr := sysstats.GetManager()
		ok, err := sysStatMgr.ReadConfig(config.SysStats)
		if ok {
			Log.MaestroDebug("sysstats read config ok. Starting...")
			sysStatMgr.Start()
		} else {
			Log.MaestroErrorf("sysstats - error reading config: %s\n", err.Error())
			log.Errorf("sysstats - error reading config: %s\n", err.Error())
		}
	}

	/*********************************************/
	/*             Process template files        */
	/*********************************************/
	// template files are processed now that we have read in all the config
	// data. These are usually config files needed by different software components
	// we will later start

	// Process static config files before we start any processes
	debugging.DEBUG_OUT("Processing static file generators...\n")
	for _, fileop := range config.StaticFileGenerators {
		if (len(fileop.TemplateFile) > 0 || len(fileop.TemplateString) > 0) && len(fileop.OutputFile) > 0 {
			op := templates.NewFileOp(fileop.Name, fileop.TemplateFile, fileop.TemplateString, fileop.OutputFile, maestroConfig.GetGlobalConfigDictionary())
			err := op.ProcessTemplateFile()
			if err != nil {
				log.Errorf("Error processing template file %s: %s\n", fileop.TemplateFile, err.Error())
			} else {
				err, wrote, checksum := op.MaybeGenerateFile()
				if err != nil {
					log.Errorf("Error creating generated file (%s) %s: %s\n", fileop.Name, fileop.OutputFile, err.Error())
				} else {
					debugging.DEBUG_OUT("Static file generator, template %d - checksum is %s\n", fileop.Name, checksum)
					if wrote {
						debugging.DEBUG_OUT("Static file generator - wrote out new file for template %s: %s\n", fileop.Name, fileop.OutputFile)
					} else {
						debugging.DEBUG_OUT("Static file generator - skipping file, no change for template %s: %s\n", fileop.Name, fileop.OutputFile)
					}
				}
			}
		} else {
			log.Errorf("Poorly formed 'static_file_generators' entry in config file. Skipping %s\n", fileop.Name)
		}
	}
	debugging.DEBUG_OUT("Done with static file generators.\n")

	/*********************************************/
	/*             Events Manager start          */
	/*********************************************/

	// starts automatically.

	/*********************************************/
	/*             Task Manager start            */
	/*********************************************/
	// the TaskManager is needed to process tasks and
	// is used by the JobsManager and the NetworkManager

	debugging.DEBUG_OUT("doing task.StartTaskManager()\n")
	tasks.StartTaskManager()

	/*********************************************/
	/*             Network startup               */
	/*********************************************/

	bringUpIfs := func() {
		// wait a few seconds to start interface bring up. we want the logging to be working, and
		// other serivces ready.
		time.Sleep(time.Second * 2)
		Log.MaestroInfo("Maestro startup: Bringing up existing network interfaces.")
		log.Info("Maestro startup: Bringing up existing network interfaces.")
		networking.GetInstance().SetupExistingInterfaces()

		/*********************************************/
		/*               Set date-time               */
		/*********************************************/
		// many things break if the time is not set correctly
		// getting the time is dependant on the network
		if config.TimeServer != nil {
			log.Info("Maestro startup: getting time from server.")
			ok, timeclient, err := maestroTime.NewClient(config.TimeServer)
			if ok {
				// wait for time to be set, and then
				// do certain things after the time is set.
				go func() {
					log.Info("top of time go routine")

					errors := 0
					for {
						ok2, ch := timeclient.StatusChannel()
						if ok2 {
							code := <-ch
							switch code {
							case maestroTime.SetTimeOk:
								log.Info("time set ok.")
								// TODO: do things when time set
								return
							case maestroTime.TimedOut:
								log.Error("time server error. Timed out.")
								errors++
							case maestroTime.BadResponse:
								log.Error("time server error. Bad Response")
								errors++
							default:
								log.Errorf("time server error. %d\n", code)
								errors++
							}
							if errors > 50 {
								log.Error("Time server is failing. Maestro will start services that needed time verification anyway.")
								break
							}
						} else {
							log.Warning("Waiting for time subsytem status channel to come up. 2 seconds..")
							time.Sleep(time.Second * 2)
						}
					}
					// TODO: do things even if get time failed
					return
				}()

				timeclient.Run()
			} else {
				log.Error("Maestro time server client failed to start.")
				if err != nil {
					log.Errorf("Maestro time server client failure details: %s\n", err.Error())
				}
			}
		} else {
			log.Info("Maestro time server not set. Not setting system time.")
		}

		/*********************************************/
		/*          Start Mdns server (optional)     */
		/*********************************************/
		if config.Mdns != nil {
			if !config.Mdns.Disable {
				mdnsMgr := mdns.GetInstance()
				if len(config.Mdns.StaticRecords) > 0 {
					ok, errs := mdnsMgr.LoadFromConfigFile(config.Mdns.StaticRecords)
					if !ok {
						for n, err := range errs {
							// if that published record had an error...
							if err != nil {
								Log.MaestroErrorf("MDNS config: static record in config file - record %d - error: %s\n", n, err.Error())
								log.Errorf("MDNS config: static record in config file - record %d - error: %s\n", n, err.Error())
							}
						}
					}
				}
			} else {
				Log.MaestroWarn("MDNS server is disabled in config file. Skipping.")
			}
		} else {
			// default: Start service, but publish nothing.
			_ = mdns.GetInstance()
		}

	}

	neterr := networking.InitNetworkManager(config.Network, config.DDBConnConfig)
	if neterr != nil {
		Log.MaestroErrorf("Error starting networking subsystem! %s\n", neterr.Error())
		log.Errorf("Error starting networking subsystem! %s\n", neterr.Error())
	} else {
		go bringUpIfs()
	}

	/*********************************************/
	/*               Jobs startup                */
	/*********************************************/

	debugging.DEBUG_OUT("doing InitImageManager()\n")
	InitImageManager(config.GetScratchPath(), config.GetImagePath())
	debugging.DEBUG_OUT("doing StartJobConfigManager()\n")
	configMgr.StartJobConfigManager(config)

	debugging.DEBUG_OUT("doing processes.InitProcessEvents()\n")
	processes.InitProcessEvents(config.Stats)

	debugging.DEBUG_OUT("starts:%+v\n", config.JobStarts)
	debugging.DEBUG_OUT("container_templates:%+v\n", config.ContainerTemplates)

	// First load existing Jobs / Templates from database:

	// config file Jobs / Templates take precedence
	for i, _ := range config.ContainerTemplates {
		var container maestroSpecs.ContainerTemplate
		container = &config.ContainerTemplates[i]
		if container.IsMutable() {
			err := DB.UpsertContainerTemplate(container)
			if err != nil {
				log.Errorf("Error on saving job in DB: %s\n", err.Error())
			}
		} else {
			debugging.DEBUG_OUT("Immutable: Not storing config file Container: %s\n", container.GetName())
		}
		processes.RegisterContainer(&config.ContainerTemplates[i])
	}

	for i, _ := range config.JobStarts {
		// _job := new(*JobStartRequestConfig)
		// *_job = &job
		var job maestroSpecs.JobDefinition
		job = &config.JobStarts[i]
		if job.IsMutable() {
			err := DB.UpsertJob(job)
			if err != nil {
				log.Errorf("Error on saving job in DB: %s\n", err.Error())
			}
		} else {
			debugging.DEBUG_OUT("Immutable: Not storing config file Job: %s\n", job.GetJobName())
		}
		processes.RegisterJob(&config.JobStarts[i])
	}

	depErr := processes.ValidateJobs()

	if depErr != nil {
		fmt.Printf("Error in jobs configuration: %s\n", depErr.Error())
		os.Exit(1)
	}

	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)
	AddProcessRoutes(router)

	unixEndpoint := new(UnixHttpEndpoint)
	err = unixEndpoint.Init(config.GetHttpUnixSocket())
	if err != nil {
		log.Error("Error on sink start: %s\n", err.Error())
	} else {
		defer unixEndpoint.Start(router, &waitGroup)
		debugging.DEBUG_OUT("Started unix socket HTTP endpoint.")
		//		sink.Shutdown()
		//		waitGroup.Wait()
	}

	defer processes.StartAllAutoStartJobs()
}

/**

Unix socket endpoint:

Basic tests can be done like:

echo -e "GET /hello/John HTTP/1.1\r\nHost: 127.0.0.1\r\n" | socat unix-connect:/tmp/maestroapi.sock STDIO

**/
