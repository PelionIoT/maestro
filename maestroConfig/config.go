package maestroConfig

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
	//	"fmt"

	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v2"
	//	"github.com/armPelionEdge/mustache"
	"github.com/armPelionEdge/greasego"
	"github.com/armPelionEdge/maestro/configs"
	"github.com/armPelionEdge/maestro/debugging"
	. "github.com/armPelionEdge/maestro/defaults"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestro/mdns"
	"github.com/armPelionEdge/maestro/sysstats"
	"github.com/armPelionEdge/maestro/time"
	"github.com/armPelionEdge/maestro/wwrmi"
	"github.com/armPelionEdge/maestroSpecs"
	"github.com/armPelionEdge/maestroSpecs/templates"
	"github.com/kardianos/osext" // not needed in go1.8 - see their README
	logging "github.com/op/go-logging"
)

// use for startup errors
var golog = logging.MustGetLogger("maestro")

/**
 * These are used by the config file, and in some cases by
 * the API
 */

type YAMLMaestroConfig struct {
	UnixLogSocket        string `yaml:"unixLogSocket"`
	SyslogSocket         string `yaml:"sysLogSocket"`
	LinuxKernelLog       bool   `yaml:"linuxKernelLog"`
	LinuxKernelLogLegacy bool   `yaml:"linuxKernelLogLegacy"`
	//	ApiUnixDgramSocket string `yaml:"apiUnixDgramSocket"` // not used yet
	HttpUnixSocket       string                                  `yaml:"httpUnixSocket"`
	VarDefs              []SubstVars                             `yaml:"var_defs"`
	TimeServer           *time.ClientConfig                      `yaml:"time_server"`
	Mdns                 *MdnsSetup                              `yaml:"mdns"`
	Watchdog             *maestroSpecs.WatchdogConfig            `yaml:"watchdog"`
	Symphony             *wwrmi.ClientConfig                     `yaml:"symphony"`
	SysStats             *sysstats.StatsConfig                   `yaml:"sys_stats"`
	Tags                 []string                                `yaml:"tags"`
	Targets              []maestroSpecs.LogTarget                `yaml:"targets"`
	ClientId             string                                  `yaml:"clientId"`
	ConfigDBPath         string                                  `yaml:"configDBPath"` // where Maestro should look for it's database
	Stats                maestroSpecs.StatsConfigPayload         `yaml:"stats"`
	JobStarts            []maestroSpecs.JobDefinitionPayload     `yaml:"jobs"`
	ContainerTemplates   []maestroSpecs.ContainerTemplatePayload `yaml:"container_templates"`
	ImagePath            string                                  `yaml:"imagePath"`   // where we should place Job images
	ScratchPath          string                                  `yaml:"scratchPath"` // where we should place temporary file, downloads, etc.
	StaticFileGenerators []StaticFileOp                          `yaml:"static_file_generators"`
	PlatformReaders      []PlatformReader                        `yaml:"platform_readers"`
	Plugins              []Plugin                                `yaml:"plugins"`
	Network              *maestroSpecs.NetworkConfigPayload      `yaml:"network"`
	Grm                  *GrmConfig                              `yaml:"grm`
	Processes            *configs.ProcessesConfig                `yaml:"processes"`
	DebugOpts            *DebugOptions                           `yaml:"debug_opts"`
	DDBConnConfig        *DeviceDBConnConfig                     `yaml:"devicedb_conn_config"`
	ConfigEnd            bool                                    `yaml:"config_end"`
}

type MdnsSetup struct {
	Disable       bool                `yaml:"disable"`
	StaticRecords []*mdns.ConfigEntry `yaml:"static_records"`
}

type DebugOptions struct {
	// PidFile if provided will write a file with the current PID
	PidFile string `yaml:"pid_file"`
	// KeepPids if true, will append the PID number instead of truncate and write
	KeepPids bool `yaml:"keep_pids"`
	// PidFileDara prints the current date/time next to the pid number in the file upod pid file creation
	PidFileDates bool `yaml:"pid_file_dates"`
}

type GrmConfig struct {
	EdgeCoreSocketPath string `yaml:"edge_core_socketpath"`
	FluentbitConfigFilePath string `yaml:"fluentbit_config_filepath"`
	FluentbitConfigObjectId int `yaml:"fluentbit_config_lwm2mobjectid"`
	
}
type DeviceDBConnConfig struct {
	// The URI of the relay's local DeviceDB instance
	DeviceDBUri string `yaml:"devicedb_uri"`
	// The prefix where keys related to configuration are stored
	DeviceDBPrefix string `yaml:"devicedb_prefix"`
	// The devicedb bucket where configurations are stored
	DeviceDBBucket string `yaml:"devicedb_bucket"`
	// The ID of the relay whose configuration should be monitored
	RelayId string `yaml:"relay_id"`
	// The file path to a PEM encoded CA chain used to validate the server certificate used by the DeviceDB instance
	CaChainCert string `yaml:"ca_chain"`
}

type PlatformReader struct {
	Platform string `yaml:"platform"`
	// Opts is an optional PluginOpts config
	Opts   *maestroSpecs.PluginOpts `yaml:"opts"`
	Params map[string]interface{}   `yaml:"params,omitempty"`
}

type Plugin struct {
	// Id is an identifier used to call out this generic plugin in the
	// logging system and in errors. Generic plugins are plugins which are
	// not a platform or watchdog plugin. Id is like a name
	ID string `yaml:"id"`
	// A path to the plugin .so file
	Path string `yaml:"path"`
	// Opts is an optional PluginOpts config
	Opts *maestroSpecs.PluginOpts `yaml:"opts"`
}

type SubstVars struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type StaticFileOp struct {
	Name           string `yaml:"name"`
	TemplateFile   string `yaml:"template_file"`
	TemplateString string `yaml:"template"`
	OutputFile     string `yaml:"output_file"`
}

var re_this_dir *regexp.Regexp // {{thisdir}}
var re_cwd *regexp.Regexp      // {{cwd}}
// var CWD string
// var THIS_DIR string

var macroVarMap *templates.TemplateVarDictionary

func GetGlobalConfigDictionary() (ret *templates.TemplateVarDictionary) {
	ret = macroVarMap
	return
}

func updateMacroVars() {
	cwd, err := os.Getwd()
	if err == nil {
		macroVarMap.Add("cwd", cwd)
		//		macroVarMap.Map["cwd"] = cwd
	}
}

func subInEnvVars(in string, varmap map[string]string) (out string) {
	m := re_findEnvSubs.FindAllStringSubmatch(in, -1)
	if m != nil {
		for _, match := range m {
			inner := re_innerEnv.FindStringSubmatch(match[1])
			if inner != nil {
				dfault := inner[2]
				subvar := inner[1]
				// fmt.Printf(">>%s<< >>%s<<  ))%s((\n",match,subvar,dfault);
				sub, ok := varmap[subvar]
				if ok {
					in = strings.Replace(in, match[0], sub, -1)
				} else {
					if len(dfault) > 0 {
						in = strings.Replace(in, match[0], dfault, -1)
					} else {
						in = strings.Replace(in, match[0], "", -1)
					}
				}
			} else {
				// no match? then it is not a valid expression, just leave alone
			}
		}

	}
	out = in
	return
}

// This goes through the "var_defs" section,
// and swaps out any ${SOMEENVVAR} for and environemntal variable
// if that env var exists. It also allows: ${SOMEENVVAR|default}
// where if SOMEENVVAR is not set, then 'default' is instead
// replaced.
// When done it puts the vars into the macroVarMap
func (ysc *YAMLMaestroConfig) mixInSubstConfigVars() {
	for _, el := range ysc.VarDefs {
		//		macroVarMap.Map[el.Key] = el.Value
		macroVarMap.Add(el.Key, GetInterpolatedConfigString(subInEnvVars(el.Value, osEnviron)))
	}
}

func _inspectAndInterpolate(s interface{}) {
	debugging.DEBUG_OUT2("ONE  struct/ptr struct {\n")

	kind := reflect.ValueOf(s).Kind()

	var reflectType reflect.Type
	var reflectValue reflect.Value

	if kind == reflect.Ptr {
		reflectType = reflect.TypeOf(s).Elem()
		debugging.DEBUG_OUT2("ONE.1 (ptr)\n")
		reflectValue = reflect.ValueOf(s).Elem()
		debugging.DEBUG_OUT2("TWO (ptr)\n")
	} else {
		reflectType = reflect.TypeOf(s)
		debugging.DEBUG_OUT2("ONE.1\n")
		reflectValue = reflect.ValueOf(s)
		debugging.DEBUG_OUT2("TWO\n")
	}

	// if reflectType.Kind() == reflect.Ptr{
	// 	reflectType = reflectType.Elem()
	// 	reflectValue = reflectValue.Elem()
	// }
	for i := 0; i < reflectType.NumField(); i++ {
		typeName := reflectType.Field(i).Name
		if reflectValue.Field(i).IsValid() {
			valueType := reflectValue.Field(i).Type()
			valueValue := reflectValue.Field(i)

			switch reflectValue.Field(i).Kind() {
			case reflect.String:
				if reflectValue.Field(i).CanSet() {
					s := valueValue.String()
					debugging.DEBUG_OUT2("**************> IN>> %s OUt>> %s\n", s, GetInterpolatedConfigString(s))
					reflectValue.Field(i).SetString(GetInterpolatedConfigString(s))
				} else {
					fmt.Printf("**************> %s(%s) - CanSet FALSE - can't interpolate\n", typeName, valueType)
				}
				debugging.DEBUG_OUT2("**************> %s : %s(%s)\n", typeName, reflectValue.Field(i).Interface(), valueType)
			// case reflect.Int32:
			//     fmt.Printf("%s : %i(%s)\n", typeName, valueValue, valueType)
			case reflect.Struct:
				debugging.DEBUG_OUT2("**************> %s : it is %s\n", typeName, valueType)
				debugging.DEBUG_OUT2("**************> %+v  { \n", valueValue)
				// if valueValue != nil {
				_inspectAndInterpolate(valueValue)
				// }
			case reflect.Slice:
				alen := valueValue.Len()
				debugging.DEBUG_OUT2("**************> SLICE len(%d) : it is %s\n", alen, reflectType.Field(i).Name)
				if alen > 0 {
					if valueValue.Index(0).Kind() == reflect.String {
						debugging.DEBUG_OUT2("**************>  SLICE has strings...\n")
						// newstrings := reflect.MakeSlice(reflect.TypeOf([]string{}), alen, alen)
						for e := 0; e < alen; e++ {
							valueValue.Index(e).Set(reflect.ValueOf(GetInterpolatedConfigString(valueValue.Index(e).String())))
							debugging.DEBUG_OUT2("**************>   slice[%d]=\"%s\"\n", e, valueValue.Index(e).String())
						}
						// valueValue.Set(newstrings)
					} else if valueValue.Index(0).Kind() == reflect.Struct {
						debugging.DEBUG_OUT2("**************>  SLICE has structs...\n")
						for e := 0; e < alen; e++ {
							debugging.DEBUG_OUT2("**************>  SLICE [%d]...\n", e)
							_inspectAndInterpolate(valueValue.Index(e).Addr().Interface())
						}
					} else if valueValue.Index(0).Kind() == reflect.Ptr {
						for e := 0; e < alen; e++ {
							if !valueValue.Index(e).IsNil() {
								debugging.DEBUG_OUT2("**************>  SLICE (ptr) [%d]...\n", e)
								_inspectAndInterpolate(valueValue.Index(e).Elem().Addr().Interface())
							}
						}
					}
				}
			case reflect.Ptr:
				if typeName != "typ" {
					debugging.DEBUG_OUT2("**************> PTR %s : it is %s\n", typeName, reflectType.Field(i).Name)
					if !valueValue.IsNil() {
						k := valueValue.Elem().Type().Kind()
						if k == reflect.Struct {
							_inspectAndInterpolate(valueValue.Elem().Addr().Interface())
							//				            _inspectAndInterpolate(valueValue.Elem().Interface())
						} else {
							debugging.DEBUG_OUT2("**************> PTR was not to struct{} skipping\n")
						}
					} else {
						debugging.DEBUG_OUT2("**************> PTR was nil\n")
					}
				}

			}
		}
	}
	debugging.DEBUG_OUT2("**************> } end struct\n")
}

func (ysc *YAMLMaestroConfig) InterpolateAllStrings() {

	_inspectAndInterpolate(ysc)

}

const _re_findEnvSubs = "\\${(.*)}"
const _re_innerEnv = "([^|]+)\\|?(.*)"

var re_findEnvSubs *regexp.Regexp
var re_innerEnv *regexp.Regexp
var osEnviron map[string]string

func init() {
	// get environmental variables on startup
	env := os.Environ()
	osEnviron = make(map[string]string)
	for _, envs := range env {
		splt := strings.Split(envs, "=")
		osEnviron[splt[0]] = splt[1]
	}

	re_findEnvSubs = regexp.MustCompile(_re_findEnvSubs)
	re_innerEnv = regexp.MustCompile(_re_innerEnv)

	macroVarMap = templates.NewTemplateVarDictionary()

	folderPath, err2 := osext.ExecutableFolder()
	if err2 == nil {
		//		macroVarMap.Map["thisdir"] = folderPath
		macroVarMap.Add("thisdir", folderPath)
	} else {
		fmt.Printf("ERROR - could not get maestro executable folder. {{thisdir}} macro will fail.")
	}
	updateMacroVars()
}

// replaces the {{VAR}} macro strings with their appropriate names values
// Currently the config files supports:
// {{thisdir}} - the directory where the maestro exec resides
// {{cwd}} the 'current working directory' of the Maestro process
func GetInterpolatedConfigString(s string) string {
	// ret = re_this_dir.ReplaceAllString(s, THIS_DIR)
	// ret = re_cwd.ReplaceAllString(ret, CWD)
	//	return mustache.Render(s,macroVarMap.Map)
	return macroVarMap.Render(s)
}

func (ysc *YAMLMaestroConfig) GetDBPath() (ret string) {
	if len(ysc.ConfigDBPath) < 1 {
		ysc.ConfigDBPath = DefaultConfigDBPath
	}
	ret = GetInterpolatedConfigString(ysc.ConfigDBPath)
	return
}

func (ysc *YAMLMaestroConfig) GetScratchPath() (ret string) {
	if len(ysc.ScratchPath) < 1 {
		ysc.ScratchPath = DefaultScratchPath
	}
	ret = GetInterpolatedConfigString(ysc.ScratchPath)
	return
}

func (ysc *YAMLMaestroConfig) GetImagePath() (ret string) {
	if len(ysc.ImagePath) < 1 {
		ysc.ImagePath = DefaultImagePath
	}
	ret = GetInterpolatedConfigString(ysc.ImagePath)
	return
}

func (ysc *YAMLMaestroConfig) GetHttpUnixSocket() (ret string) {
	if len(ysc.HttpUnixSocket) < 1 {
		ysc.HttpUnixSocket = DefaultHttpUnixSocket
	}
	ret = GetInterpolatedConfigString(ysc.HttpUnixSocket)
	return
}

func (ysc *YAMLMaestroConfig) GetUnixLogSocket() (ret string) {
	if len(ysc.UnixLogSocket) < 1 {
		ysc.UnixLogSocket = DefaultUnixLogSocketSink
	}
	ret = GetInterpolatedConfigString(ysc.UnixLogSocket)
	return
}

func (ysc *YAMLMaestroConfig) GetSyslogSocket() (ret string) {
	if len(ysc.SyslogSocket) < 1 {
		return ""
		// ysc.SyslogSocket = DefaultSyslogSocket
	}
	ret = GetInterpolatedConfigString(ysc.SyslogSocket)
	return
}

// FillInDefaults goes through specific parts of the config and puts in defaults if strings
// were missing or empty.
func (ysc *YAMLMaestroConfig) FillInDefaults() {
	for n, _ := range ysc.JobStarts {
		_job := maestroSpecs.JobDefinition(&ysc.JobStarts[n])
		if len(ysc.JobStarts[n].ConfigName) < 1 {
			ysc.JobStarts[n].ConfigName = _job.GetConfigName()
		}
	}
}

// FinalizeConfig Interpolates all config strings and fills in defaults
func (ysc *YAMLMaestroConfig) FinalizeConfig() {
	ysc.mixInSubstConfigVars()
	ysc.InterpolateAllStrings()
	ysc.FillInDefaults()
}

// ConfigError is the error type when there is an issue in the config parsing
type ConfigError struct {
	code        int
	errorString string
}

// Error implements error.Error
func (err *ConfigError) Error() string {
	return err.errorString
}

// LoadFromFile load the config file
func (ysc *YAMLMaestroConfig) LoadFromFile(file string) error {
	log.MaestroInfo("Loading config file:", file)
	rawConfig, err := ioutil.ReadFile(file)

	if err != nil {
		log.MaestroError("Failed to load config file:", err)
		return err
	}

	err = yaml.Unmarshal(rawConfig, ysc)

	if err != nil {
		log.MaestroError("Failed to parse config:", err)
		return err
	}

	if ysc.ConfigEnd != true {
		golog.Error("Did not see a \"config_end: true\" statement at end. Possible bad parse??")
		//    	return &ConfigError{1,"Did not see a \"config_end: true\" statement at end. Bad parse."}
	}

	// if len(ysc.UnixLogSocket) < 1 {
	// 	return errors.New("Must have a unixLogSocket path, for the primary log sink.")
	// }

	return nil
}

// @param levels is a string of one or more comma separated value for levels,
// like: "warn, error"
func ConvertLevelStringToUint32Mask(levels string) uint32 {
	ret := uint32(0)
	parts := strings.Split(levels, ",")
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if strings.Compare(s, "all") == 0 {
			return uint32(greasego.GREASE_ALL_LEVELS)
		} else {
			ret |= greasego.DefaultLevelMap[s]
		}
	}
	return ret
}

func ConvertLevelUint32MaskToString(mask uint32) string {
	if mask == uint32(greasego.GREASE_ALL_LEVELS) {
		return "all"
	}

	var ret string
	for k, v := range greasego.DefaultLevelMap {
		if v&mask == v {
			if len(ret) > 0 {
				ret += ","
			}
			ret += k
		}
	}

	return ret
}

func ConvertTagStringToUint32(tag string) uint32 {
	return greasego.DefaultTagMap[strings.TrimSpace(tag)]
}

func ConvertTagUint32ToString(tag uint32) string {
	var ret string
	for k, v := range greasego.DefaultTagMap {
		if v&tag == v {
			if len(ret) > 0 {
				ret += ","
			}
			ret += k
		}
	}

	return ret
}

func ConfigGetMinDiskSpaceScratch() uint64 {
	// TODO: get from config
	return MIN_DISK_SPACE_SCRATCH_DISK
}
