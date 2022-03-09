# Troubleshooting Tips for Kubernetes Cluster API Provider Bring Your Own Host (BYOH)
This section includes tips to help you to troubleshoot common problems that you might encounter when installing Kubernetes Cluster API Provider BYOH.

## Failed installation, pre-requisite not installed on the host: socat 
### Probem 
Trying to install BYOH successfully detects OS but fails on pre-requisite package precheck for the package socat.
```
I0307 05:49:30.561917   13907 installer.go:104]  "msg"="Detected"  "OS"="Ubuntu_20.04.2_x86-64"
E0307 05:49:30.562132   13907 checks.go:38]  "msg"="Failed pre-requisite packages precheck" "error"="required package(s): [socat] not found"  
E0307 05:49:30.562195   13907 cli-dev.go:141]  "msg"="unable to create installer" "error"="precheck failed" 
```
### Solution
To solve the problem the package socat needs to be installed on the host.
Example for ubuntu with apt-get:
```
sudo apt-get install socat
```

## Failed installation, pre-requisite not installed on the host: ethtool
### Probem 
Trying to install BYOH successfully detects OS but fails on pre-requisite pagackage precheck for the package ethtool.
```
I0307 05:49:30.561917   13907 installer.go:104]  "msg"="Detected"  "OS"="Ubuntu_20.04.2_x86-64"
E0307 05:49:30.562132   13907 checks.go:38]  "msg"="Failed pre-requisite packages precheck" "error"="required package(s): [ethtool] not found"  
E0307 05:49:30.562195   13907 cli-dev.go:141]  "msg"="unable to create installer" "error"="precheck failed" 
```
### Solution
To solve the problem the package ethtool needs to be installed on the host.
Example for ubuntu with apt-get:
```
sudo apt-get install ethtool
```

## Failed installation, pre-requisite not installed on the host: conntrack
### Probem 
Trying to install BYOH successfully detects OS but fails on pre-requisite pagackage precheck for the package conntrack.
```
I0307 05:55:40.203851   15309 installer.go:104]  "msg"="Detected"  "OS"="Ubuntu_20.04.2_x86-64"
E0307 05:55:40.204188   15309 checks.go:38]  "msg"="Failed pre-requisite packages precheck" "error"="required package(s): [conntrack] not found"  
E0307 05:55:40.204260   15309 cli-dev.go:141]  "msg"="unable to create installer" "error"="precheck failed"
```
### Solution
To solve the problem the package conntrack needs to be installed on the host.
Example for ubuntu with apt-get:
```
sudo apt-get install conntrack
```

## Failed installation, pre-requisite not installed on the host: ebtables
### Probem 
Trying to install BYOH successfully detects OS but fails during installation.
```
I0307 06:11:32.069667   16169 installer.go:244]  "msg"="dpkg: dependency problems prevent configuration of kubelet:\n kubelet depends on ebtables; however:\n  Package ebtables is not installed.\n\ndpkg: error processing package kubelet (--install):\n dependency problems - leaving unconfigured\nErrors were encountered while processing:\n kubelet\n"  
I0307 06:11:32.069772   16169 installer.go:244]  "msg"="exit status 1"
```
### Solution
To solve the problem the package ebtables needs to be installed on the host.
Example for ubuntu with apt-get:
```
sudo apt-get install ebtables
```

## Failed installation, multiple pre-requisites not installed on the host
### Problem
Trying to install BYOH successfully detects OS but fails on pre-requisite pagackage precheck for multiple packages. If there is more than one pre-requisite no installed on the host, all will be written in the brackets like in this example when socat, ethtool and conntrack are no found.
```
I0307 05:49:30.561917   13907 installer.go:104]  "msg"="Detected"  "OS"="Ubuntu_20.04.2_x86-64"
E0307 05:49:30.562132   13907 checks.go:38]  "msg"="Failed pre-requisite packages precheck" "error"="required package(s): [socat ethtool conntrack] not found"  
E0307 05:49:30.562195   13907 cli-dev.go:141]  "msg"="unable to create installer" "error"="precheck failed"
```
### Solution
All of the missing required packages need to be installed.
Example for this case where socat, ethtool and conntrack are not installed:
```
sudo apt-get install socat ethtool conntrack
```

## Error downloading bundle
### Problem
After successful pre-requisite package prechecks when the bundle is not found locally the installer will try to download it from the given repo but can fail with error `Error downloading bundle`.
```
I0307 06:15:24.903253   19079 installer.go:104]  "msg"="Detected"  "OS"="Ubuntu_20.04.2_x86-64"
I0307 06:15:24.903551   19079 installer.go:175]  "msg"="Current OS will be handled as"  "OS"="Ubuntu_20.04.1_x86-64"
I0307 06:15:24.903807   19079 bundle_downloader.go:69]  "msg"="Cache miss"  "path"="projects.registry.vmware.com.cluster_api_provider_bringyourownhost/v1.22.3-v0.1.0_alpha.2"
I0307 06:15:24.904267   19079 bundle_downloader.go:95]  "msg"="Downloading bundle"  "from"="projects.registry.vmware.com/cluster_api_provider_bringyourownhost/byoh-bundle-ubuntu_20.04.1_x86-64_k8s_v1.22.3:v0.1.0_alpha.2"
E0307 06:15:29.452444   19079 cli-dev.go:151]  "msg"="error installing/uninstalling" "error"="Error downloading bundle" 
```

### Solution
Check your internet connection and if you can reach the repo.

Another thing that can be attempted is to download the bundle manually using docker with the command 

`docker pull <repo>/<bundle-name>:<tag>` 

where each element with brackets needs to be replaced with the corresponding string:
```
<repo> - The address of the repo 
<bundle-name> - The name of the BYOH bundle
<tag> - The tag of the BYOH bundle
```

## Error installing/uninstalling, No k8s support for OS
### Problem
After successful pre-requisite package prechecks, the installer cannot find the BYOH bundle for the given combination of OS and K8s version.
```
I0308 05:18:54.733467   11351 installer.go:104]  "msg"="Detected"  "OS"="Ubuntu_20.04.2_x86-64"
E0308 05:18:54.733622   11351 cli-dev.go:151]  "msg"="error installing/uninstalling" "error"="No k8s support for OS"
```
### Solution
Sometimes it may happen that the OS and K8s version combination used is not supported by `BYOH` out of the box. This will require manually installing all the dependencies and using the `--skip-installation` flag. This flag will skip k8s installation attempt on the host.
