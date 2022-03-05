module github.com/PelionIoT/maestro

go 1.16

replace (
	github.com/shirou/gopsutil v2.16.12+incompatible => github.com/PelionIoT/gopsutil v2.16.12+incompatible
	golang.org/x/net => golang.org/x/net v0.0.0-20210520170846-37e1c6afe023 // Required to fix CVE-2021-33194
)

require (
	github.com/PelionIoT/httprouter v0.0.0-20170104185816-8a45e95fc75c
	github.com/PelionIoT/maestro-plugins-template v0.0.0-20190409060302-5b22b1f86b05
	github.com/PelionIoT/maestroSpecs v2.4.0+incompatible
	github.com/PelionIoT/mustache v0.0.0-20160804235033-6375acf62c69 // indirect
	github.com/PelionIoT/netlink v0.0.0-20190409055558-1cddec0bf368
	github.com/PelionIoT/stow v2.2.1-0.20170902223607-b0e83fd073d4+incompatible
	github.com/PelionIoT/structmutate v0.0.0-20190409035007-57b37c3ac1d9
	github.com/PelionIoT/zeroconf v0.0.0-20180517163623-78d055b6305c
	github.com/ThomasRooney/gexpect v0.0.0-20161231170123-5482f0350944 // indirect
	github.com/armPelionEdge/dhcp4 v0.0.0-20180917122751-7f54fcf15bd7
	github.com/armPelionEdge/dhcp4client v0.0.0-20190409055833-be924652a34a
	github.com/armPelionEdge/netlink v0.0.0-20190409055558-1cddec0bf368 // indirect
	github.com/armPelionEdge/wpa-connect v0.0.0-20200827001433-10297d26dfc8
	github.com/boltdb/bolt v1.3.2-0.20180302180052-fd01fc79c553
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/d2g/dhcp4 v0.0.0-20170904100407-a1d1b6c41b1c // indirect
	github.com/d2g/dhcp4client v1.0.0 // indirect
	github.com/deckarep/golang-set v1.8.0
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/frankban/quicktest v1.14.2 // indirect
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gorilla/websocket v1.5.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/mholt/archiver v3.1.2-0.20181212200041-4e41ad6272dc+incompatible
	github.com/miekg/dns v1.1.46 // indirect
	github.com/nwaples/rardecode v1.1.2 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/shirou/gopsutil v2.16.12+incompatible
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/vishvananda/netns v0.0.0-20211101163701-50045581ed74 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9
	gopkg.in/yaml.v2 v2.4.0
)
