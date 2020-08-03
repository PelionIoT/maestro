MESSAGE ("Building standard x86_64 desktop Linux target")
SET (OS_BRAND Linux)
SET (MBED_CLOUD_CLIENT_DEVICE x86_x64)
SET (PAL_TARGET_DEVICE x86_x64)

SET (PAL_FS_MOUNT_POINT_PRIMARY "\"/home/vagrant/work/gostuff/src/github.com/armPelionEdge/mbed-edge/mcc_config\"")
SET (PAL_FS_MOUNT_POINT_SECONDARY "\"/home/vagrant/work/gostuff/src/github.com/armPelionEdge/mbed-edge/mcc_config\"")
SET (PAL_UPDATE_FIRMWARE_DIR "\"/home/vagrant/work/gostuff/src/github.com/armPelionEdge/mbed-edge/upgrades\"")
SET (PAL_USER_DEFINED_CONFIGURATION "\"${CMAKE_CURRENT_SOURCE_DIR}/config/sotp_fs_linux.h\"")

if (${FIRMWARE_UPDATE})
  SET (MBED_CLOUD_CLIENT_UPDATE_STORAGE "ARM_UCP_LINUX_GENERIC")
endif()

