# maestro-specs
Maestro RMI payload specifications

### Overview

Maestro provides managment of the following:
* **Job Management**: Jobs are usually one or more pieces of software which can be started & stopped. <br>See [jobsAndImages.go](https://github.com/armPelionEdge/maestroSpecs/blob/master/jobsAndImages.go#L329)

* **Software image management**: Jobs need software to run. Image management handles the movement, installation, deletion and version control of software used by a Job. Job definitions may reside, but an image may be updated. If the image is updated, the Job is simply started, the image is updated, and then the Job is restarted. <br>See [jobsAndImages.go](https://github.com/armPelionEdge/maestroSpecs/blob/master/jobsAndImages.go#L299)

* **Config management**: Jobs usually have a 'configuration' piece. This config may be a simple string of data, or may be one or many complex configuration files. These configs need to be moved, stored and made available to a Job. <br>See [config.go](https://github.com/armPelionEdge/maestroSpecs/blob/master/configs.go#L10)

* **Templates & containers**: Many jobs use similar forms and require similar resources. Templates allow defining of common procedures such as start commands, resource requirements, and security settings. Example: A deviceJS script requires a standard way of starting, so deviceJS jobs use a particular template.  Capabilities such as `chroot()`ing a process, limiting memory and other resources, and running a Job under a particular user ID are available through templates.<br>See [jobsAndImages.go](https://github.com/armPelionEdge/maestroSpecs/blob/master/jobsAndImages.go#L256)<br><br>
Some Container Templates are predefined, such as `devicejs`. These templates are defined in the maestro config file.


### A 'Running Config'
---

The simple way to update an entire configuration with Maestro is to send the daemon a 'running config' which is a larger JSON payload which states exactly what *Jobs*, *Container Templates* they use, *Images* they use, and *Configs* they use, as one large task for the daemon to setup.

**Example:**

Setup two deviceJS processes, each with multiple modules a piece. Each module is defined as a Job, but the Jobs can share the same process via their *composite process ID* which can be retried with `GetCompositeProcess()`. This example also sets up a single Job which just launches a proprietary binary.

*As JSON*

```
{
   "config" : "running",
   "date": 7329847329,
   "jobs":
     [
        {
          "job": "zigbeeHA",
          "composite_id": "devicejs-core-modules",
          "container_template":"devicejs",
          "config":"zigbeeHA-edge500v1"
        },
        {
          "job": "UPnP",
          "composite_id": "devicejs-core-modules",
          "container_template":"devicejs",
          "config":"std",
          "disabled":true
        },
        {
          "job": "testscript2",
          "container_template":"python-wrapper-debug",
          "exec_cmd":"{{IMAGE}}/testscript2.py"
        }        
     ],
  "templates":
     [
        {
          "name":"python-wrapper-debug",
          "exec_cmd":"/usr/bin/python",
          "exec_pre_args":[ "-v" ],
          "restart": true,
          "restart_limit":10,
          "uses_ok_string":true,
          "cgroup": {
             "mem_limit":10485100
          },
          "inherit_env": true,
          "add_env": [ "SPECIALVAR=stuff" ]
        }
     ],
  "images": 
     [
        {
          "job":"testscript2",
          "version":"0.0.1",
          "url":"https://www.wigwag.io/images/testscript2.tar.gz",
          "size": 10922,
          "checksum" : {
            "type":"sha256sum",
            "value":"c01b39c7a35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b"
          },
          "image_type":"tarball"
        },
        {
          "job":"zigbeeHA",
          "version":"1.0.8",
          "url":"https://www.wigwag.io/images/wigwag-core-modules_zigbeeha_aarch64.tar.gz",
          "size": 419181,
          "checksum" : {
            "type":"sha256sum",
            "value":"defbabca35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b"
          },
          "image_type":"devicejs_tarball"
        },
        {
          "job":"UPnP",
          "version":"1.0.1",
          "url":"https://www.wigwag.io/images/wigwag-core-modules_upnp_aarch64.tar.gz",
          "size": 10991,
          "checksum" : {
            "type":"sha256sum",
            "value":"09fba37a35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b"
          },
          "image_type":"devicejs_tarball"
        }
     ],
  "configs":
     [
        {
           "name":"standard-edge500",
           "job":"zigbeeHA",
           "data": "{\n    \"siodev\": \"/dev/ttyUSB0\",\n    \"devType\": 0,\n    \"newNwk\": false,\n    \"channelMask\": 20,\n    \"baudRate\": 115200,\n    \"log_level\": 2,\n    \"networkRefreshDuration\": 0,\n    \"panIdSelection\": \"randomInRange\",\n    \"panId\": 16,\n    \"platform\": \"\",\n    \"logLevel\": 2\n}",
           "encoding":"utf8"
        },
        {
           "name":"testconfig",
           "job":"testscript2",
           "files": [ {
              "url":"https://www.wigwag.io/configfiles/testconfig.dat",
              "dest_relative_path":"/config/testconfig.dat",
              "checksum": {
                "type":"sha256sum",
                "value":"09fba37a35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b"
              },
              "size": 9101
           } ]
        }        
     ]
}
```

##### Notes

- In the above example `UPnP` is `disabled` `true` This means maestro will have the job in it's database, but the job will be stopped. If this job is currently running, Maestro will stop the job.


