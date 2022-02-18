#!/bin/bash

<<COMMENT
ssh-copy-id <alias-name-for-this-jump-box>
> sudo bash
> chmod +w /etc/sudoers
> vi /etc/sudoers
<your-user-name>  ALL=(ALL) NOPASSWD:ALL
> exit
ssh-keygen -t rsa
docker login

COMMENT

function isCmdInstalled() {
    cmd=$1
    ${cmd} version 2>&1 | grep -q "not found"
    returnCode=$?  
    if [ ${returnCode} -eq 0 ] ; then
        echo "$cmd not found"
    else
        echo "$cmd found"
    fi
    return ${returnCode}
}

function exitIfNot(){
    code=$1
    expectedCode=$2
    prompt=$3
    if [ ${code} == ${expectedCode} ]; then
        echo ${prompt}
        exit
    fi
}

function runCmd(){
    cmd=$1
    expectCode=$2
    promptBeforeCmd=$3
    PromptAfterCmdSuc=$4
    promptAfterCmdFail=$5

    if [ -n "${promptBeforeCmd}" ]; then
        echo ${promptBeforeCmd}
    fi

    # run command
    ${cmd}
    returnCode=$?
    if [ ${returnCode} -ne ${expectCode} ]; then
        if [ -n "${promptAfterCmdFail}" ]; then
            echo ${promptAfterCmdFail}
        fi
        echo "${cmd} failed, exit...."
        exit
    fi

    if [ -n "${PromptAfterCmdSuc}" ]; then
        echo ${PromptAfterCmdSuc}
    fi
}

function runCmdUntil() {
    cmd=$1
    expectCode=$2
    waitTime=$3
    maxRunTimes=$4
    promptBeforeCmd=$5
    PromptAfterCmdSuc=$6
    promptAfterCmdFail=$7

    echo "huchen: cmd is ${cmd}"

    if [ -n "${promptBeforeCmd}" ]; then
        echo ${promptBeforeCmd}
    fi
    for((i=1;i<=${maxRunTimes};i++));  
    do   
        ${cmd}
        returnCode=$?
        if [ ${returnCode} -ne ${expectCode} ]; then
            sleep ${waitTime}
            continue
        else   
            if [ -n "${PromptAfterCmdSuc}" ]; then
                echo ${PromptAfterCmdSuc}
            fi
            return
        fi
        
    done
    if [ -n "${promptAfterCmdFail}" ]; then
        echo ${promptAfterCmdFail}
    fi
    exit
}

function isIPOccupied() {
    ip=$1
    occupiedIPList=$2
    echo "Checking if ${ip} is occupied"

    echo "huchen: occupiedIPList is ${occupiedIPList}"
    IFS=$'\n'
    for occupiedIPWithMask in ${occupiedIPList}
    do
        echo "huchen: occupiedIPWithMask is ${occupiedIPWithMask}"
        occupiedIP=`echo $occupiedIPWithMask | sed -En 's/^(.*)\/([0-9]{1,2})/\1/p'`
        echo "huchen: ip is ${ip}, occupiedIP is ${occupiedIP}"
        if [ "${ip}" == "${occupiedIP}" ]; then
             return 1
        fi
    done
    echo "${ip} is not occupied"
    return 0
}


function num2IP() {
    num=$1
    a=$((num>>24))
    b=$((num>>16&0xff))
    c=$((num>>8&0xff))
    d=$((num&0xff))
    echo "$a.$b.$c.$d"
}  

function binary2IP() {
    num=`echo "ibase=2;  $1" | bc`
    returnValue=$(num2IP $num)
    echo ${returnValue}
}


function calcControlPlaneIP() {
    subNet=`docker network inspect kind | jq -r 'map(.IPAM.Config[].Subnet) []'`
    occupiedIPList=`docker network inspect kind | jq -r 'map(.Containers[].IPv4Address) []'`

    for line in ${subNet}
    do
        echo ${line} | grep ":"
        if [ $? -ne 0 ]; then
            IPMask=${line}
            break
        fi
    done

    echo "IPMask is ${subNet}"
    echo "occupiedIPList:${occupiedIPList}"

    ip=`echo $IPMask | sed -En 's/^(.*)\/([0-9]{1,2})/\1/p'`
    ipSubNetBit=`echo $IPMask | sed -En 's/^(.*)\/([0-9]{1,2})/\2/p'`
    ipSubHostBit=$[32-${ipSubNetBit}]

    IFS=.
    hexIP=`for str in ${ip}; do printf "%02X" $str; done`
    binaryIP=`echo "ibase=16; obase=2; ${hexIP}" | bc`

    ipSubNet=`echo ${binaryIP:0:${ipSubNetBit}}`

    full0="00000000000000000000000000000010"
    full1="11111111111111111111111111111110"

    minIPBinary=${ipSubNet}`echo ${full0:${ipSubNetBit}}`
    maxIPBinary=${ipSubNet}`echo ${full1:${ipSubNetBit}}`

    minIPint=`echo "ibase=2; ${minIPBinary}"|bc`
    maxIPint=`echo "ibase=2; ${maxIPBinary}"|bc`


    for((i=${minIPint};i<=${maxIPint};i++));  
    do
        ip=$(num2IP ${i})

        isIPOccupied "${ip}" "${occupiedIPList}"
        if [ $? == 0 ]; then
            CONTROL_PLANE_ENDPOINT_IP=${ip}
            echo "Available IP is ${CONTROL_PLANE_ENDPOINT_IP}"
            return
        fi
    done
}

function configClusterctl() {
    # Write clusterctl.yaml
    writeByoh=0
    clusterCtlYamlFile="${HOME}/.cluster-api/clusterctl.yaml"
    if [ ! -f "${clusterCtlYamlFile}" ]; then
        writeByoh=1
        touch ${clusterCtlYamlFile}
    else
        grep -q "byoh" ${clusterCtlYamlFile}
        if [ $? -ne 0 ] ; then
            writeByoh=1
        fi
    fi 

    if [ ${writeByoh} -eq 1 ] ; then
        cat>>${clusterCtlYamlFile}<<EOF
providers:
  - name: byoh
    url: https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/releases/latest/infrastructure-components.yaml
    type: InfrastructureProvider
EOF
    fi

    clusterctl config repositories | grep -q byoh
    if [ $? -ne 0 ] ; then
        echo "Config clusterctl failed..."
        exit
    fi
}

function runByohAgent(){
    index=$1
    byohBinaryFile=$2
    manageClusterConfFile=$3

    runCmd "docker cp ${byohBinaryFile} host${index}:/byoh-hostagent" 0 "Copying agent binary to byoh container: host${index}..."
    runCmd "docker cp ${manageClusterConfFile} host${index}:/management-cluster.conf" 0 "Copying kubeconfig to byoh container: host${index}..."
    docker exec -d host$i sh -c "chmod +x /byoh-hostagent && /byoh-hostagent --kubeconfig management-cluster.conf --skip-installation > /agent.log 2>&1"
    if [ $? -ne 0 ] ; then
        echo "Starting the host${index} agent..."
        exit
    fi

    
    maxRunTimes=10
    waitTime=1
    for((i=1;i<=${maxRunTimes};i++));  
    do   
        kubectl get byohosts host${index} | grep -v NAME | grep -q host${index}
        if [ $? -ne 0 ]; then
            sleep ${waitTime}
            continue
        else   
            echo "byohost object(host${index}) is created successfully..."
            return
        fi
    done

    echo "Error: byohost object(host${index}) is created failed..."
    exit
}

function installDocker() {
    cmdName="docker"

    ## check  if dependency is present
    isCmdInstalled "${cmdName}"

    if [ $? -ne 0 ]; then
        return
    fi

    ## install it if it not installed
    echo "Installing docker...."
    runCmd "sudo apt update" 0
    runCmd "sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common" 0
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
    sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu  $(lsb_release -cs)  stable"
    runCmd "sudo apt-get install -y docker-ce" 0

    ## check  if denpency is installed successfully
    isCmdInstalled  "${cmdName}"
    exitIfNot $? 0 "Installing ${cmdName} failed, exit..."
    runCmd "sudo systemctl enable docker" 0
}

function enableDocker() {
    runCmd "sudo systemctl start docker" 0
    runCmd "sudo systemctl enable docker" 0

    sudo systemctl status docker | grep -q "active (running)"
    if [ $? -ne 0 ]; then
        echo "Enable docker failed"
        exit 
    fi
    echo "Enable docker success"

    # Make sure current user has permission for docker, do this if not Create the docker group.
    docker ps 2>&1 | grep -q "connect: permission denied"
    if [ $? -eq 0 ]; then

        grep -q "docker" /etc/group 
        if [ $? -ne 0 ]; then
            runCmd "sudo groupadd docker" 0
        else
            echo "group 'docker' already exists"
        fi
        USER=`whoami`
        # Add your user to the docker group.
        runCmd "sudo usermod -aG docker ${USER}" 0 "Add ${USER} to docker group"
        echo "You would need to log out and log back in so that your group membership is re-evaluated. Rerun this script after that."
        exit
    fi
    echo "current user has permission for docker"
}

function installKind() {
    cmdName="kind"
     ## check  if denpency is installed before
    isCmdInstalled "${cmdName}"

    ## install denpency if it not installed
    if [ $? -eq 0 ] ; then
        echo "Installing ${cmdName}..."
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.11.1/kind-linux-amd64 && sudo install kind /usr/local/bin/kind
        ## check  if denpency is installed successfully
        isCmdInstalled  "${cmdName}"
        exitIfNot $? 0 "Installing ${cmdName} failed, exit..."
    fi
}

function installClusterctl() {
    cmdName="clusterctl"
     ## check  if denpency is installed before
    isCmdInstalled "${cmdName}"

    ## install denpency if it not installed
    if [ $? -eq 0 ] ; then
        echo "Installing ${cmdName}..."
        curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.1.1/clusterctl-linux-amd64 -o clusterctl && sudo install clusterctl /usr/local/bin/clusterctl
        ## check  if denpency is installed successfully
        isCmdInstalled  "${cmdName}"
        exitIfNot $? 0 "Installing ${cmdName} failed, exit..."
    fi
}

function intallDependencies(){
    installDocker
    enableDocker
    installKind
    installClusterctl
    commonInstall jq "sudo apt install -y jq"
    commonInstall kubectl "sudo snap install kubectl --classic"
    commonInstall make "sudo apt install -y build-essential"
    commonInstall go "sudo snap install go --classic"
}

function commonInstall(){
    cmdName=$1
    installCmd=$2

    ## check  if denpency is installed before
    isCmdInstalled "${cmdName}"

    ## install denpency if it not installed
    if [ $? -eq 0 ] ; then
        runCmd "${installCmd}" 0 "Installing ${cmdName}..."

        ## check  if denpency is installed successfully
        isCmdInstalled  "${cmdName}"
        exitIfNot $? 0 "Installing ${cmdName} failed, exit..."
    fi
}

function createKindCluster(){
    echo  "Creating kind cluster..."

    msg=$(kind create cluster 2>&1)
    if [ $? -ne 0 ]; then
        echo "Create kind cluster failed"
        echo $msg | grep -q "You have reached your pull rate limit"
        if [ $? -eq 0 ]; then 
            echo "Suggestion: you can use \"docker login\" to avoid such an error."
            exit
        fi
    else
        echo "Create kind cluster successfully"
    fi
}

function downloadByohCode(){
    runCmd "rm -rf cluster-api-provider-bringyourownhost" 0 " Cleaning byoh code..."
    msg=$(git clone git@github.com:vmware-tanzu/cluster-api-provider-bringyourownhost.git 2>&1)
    if [ $? -ne 0 ]; then
        echo "Downloading byoh code..."
        echo $msg | grep "Please make sure you have the correct access rights"
        if [ $? -eq 0 ]; then 
            echo "Suggestion: Add an public key of this machine into \"SSH and GPG keys\" of your github setting"
            exit
        fi
    else
        echo "Download byoh code successfully"
    fi
}

export PATH=/snap/bin:${PATH}
intallDependencies 
 
runCmd "sudo swapoff -a" 0 "Turning off swap..."
swapMsg=$(sudo swapon -s)
if [ -n "${swapMsg}" ]; then
    echo "Please turn off swap first."
    exit
fi
 
# check if cluster "kind" is already exited
clusterName=`kind get clusters`
if [ "${clusterName}" != "kind" ]; then
    #runCmd "kind create cluster" 0 "Creating kind cluster..."
    createKindCluster
fi

configClusterctl

# check if init it before
kubectl get pods --all-namespaces | grep -q byoh-controller-manager
if [ $? -ne 0 ]; then
    runCmd "clusterctl init --infrastructure byoh" 0 " Transforming the Kubernetes cluster into a management cluster..."
else
    echo "clusterctl init --infrastructure byoh before"
fi

#check if byoh image is existed
docker images | grep "byoh-dev/node" | grep -q "v1.22.3"
if [ $? -ne 0 ]; then
    #runCmd "rm -rf cluster-api-provider-bringyourownhost" 0 " Cleaning byoh code..."
    #runCmd "git clone git@github.com:vmware-tanzu/cluster-api-provider-bringyourownhost.git" 0 " Downloading byoh code..."
    downloadByohCode
    runCmd "cd cluster-api-provider-bringyourownhost" 0
    # The origin one will report error:   Could not connect to apt.kubernetes.io:443 (10.25.207.164), connection timed out [IP: 10.25.207.164 443]
    echo "deb http://packages.cloud.google.com/apt/ kubernetes-xenial main" > test/e2e/kubernetes.list
    runCmd "make prepare-byoh-docker-host-image-dev" 0  "Making a byoh image: byoh-dev/node:v1.22.3 ..."
    runCmd "cd -" 0
else
    echo "byoh image \"byoh-dev/node:v1.22.3\" existed."
fi

#check if byoh binary is existed
byohBinaryFile="/tmp/byoh-hostagent-linux-amd64"
if [ ! -f "${byohBinaryFile}" ]; then
    runCmd "wget https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/releases/download/v0.1.0/byoh-hostagent-linux-amd64 -P /tmp" 0  "Downloading a byoh binary..."
else
    echo "${byohBinaryFile} existed."
fi

manageClusterConfFile="${HOME}/.kube/management-cluster.conf"
cp -f ${HOME}/.kube/config ${manageClusterConfFile}


KIND_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane) 

grep ${KIND_IP} ${manageClusterConfFile} | grep -q 6443
if [ $? -ne 0 ]; then
    sed -i 's/    server\:.*/    server\: https\:\/\/'"${KIND_IP}"'\:6443/g' ${manageClusterConfFile}
else
    echo "Already modified ${manageClusterConfFile} before"
fi

## Register BYOH host to management cluster
for i in {1..2}
do
  #Check if container "host$i" is already created.
  docker ps -a | grep -q "host${i}"
  if [ $? -ne 0 ]; then
    runCmd "docker run --detach --tty --hostname host${i} --name host${i} --privileged --security-opt seccomp=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /lib/modules:/lib/modules:ro --network kind byoh-dev/node:v1.22.3" 0 "Starting byoh container: host${i}..."
    runByohAgent ${i} ${byohBinaryFile} ${manageClusterConfFile}
  else
    echo "Container \"host${i}\" is already created"

    # check if docker status is valid
    status=`docker container inspect -f '{{.State.Status}}' host${i}`
    if [ "${status}" != "running" ]; then
        echo "Error: Status of Container \"host${i}\" is ${status}, suggest to remove it, then rerun this script"
        exit
    fi

    # check if byoh process running in this container
    docker exec host${i} sh -c "ps aux" | grep -q byoh-hostagent
    if [ $? -ne 0 ]; then
        runByohAgent ${i} ${byohBinaryFile} ${manageClusterConfFile}
    else
        echo "byoh process is already ran in this container"
    fi
  fi
done

# Find a available IP for control plane endpoint
CONTROL_PLANE_ENDPOINT_IP=""
calcControlPlaneIP
if [ -z "${CONTROL_PLANE_ENDPOINT_IP}" ]; then
    echo "Can't get an available IP for control plane endpoint, exit...."
    exit
fi
echo "CONTROL_PLANE_ENDPOINT_IP is ${CONTROL_PLANE_ENDPOINT_IP}"

clusterYamlFile="/tmp/cluster-yaml"
CONTROL_PLANE_ENDPOINT_IP=${CONTROL_PLANE_ENDPOINT_IP} clusterctl generate cluster byoh-cluster --infrastructure byoh --kubernetes-version v1.22.3 --control-plane-machine-count 1  --worker-machine-count 1 --flavor docker > "${clusterYamlFile}"
if [ $? -ne 0 ]; then
    echo "Generate ${clusterYamlFile} failed, exiting..."
    exit
fi

echo "Creating the workload cluster..."
kubectl apply -f ${clusterYamlFile} 
if [ $? -ne 0 ]; then
    echo "Create the workload cluster failed"
    exit
fi

echo "Retrieving the kubeconfig of workload cluster..."

maxRunTimes=10
waitTime=1
kubeConfigFile=/tmp/byoh-cluster-kubeconfig
for((i=1;i<=${maxRunTimes};i++));  
do   
    kubectl get secret/byoh-cluster-kubeconfig 2>&1 | grep -q "not found"
    if [ $? -eq 0 ]; then
        sleep ${waitTime}
        continue
    else 
        kubectl get secret/byoh-cluster-kubeconfig -o json | jq -r .data.value | base64 --decode  > ${kubeConfigFile}
        echo "Retrieve the kubeconfig of workload cluster successfully"
        break
    fi
done

echo "Applying a CNI for network..."

# Sometimes work cluster is not entirely ready, it reports error: Unable to connect to the server: dial tcp 172.18.0.5:6443: connect: no route to host
maxRunTimes=10
waitTime=5
cniSucc=0
for((i=1;i<=${maxRunTimes};i++));  
do   
    KUBECONFIG=${kubeConfigFile} kubectl apply -f https://docs.projectcalico.org/v3.20/manifests/calico.yaml
    if [ $? -ne 0 ]; then
        sleep ${waitTime}
        continue
    else 
        echo "Applya CNI for network successfully"
        cniSucc=1
        break
    fi
done

if [ ${cniSucc} -eq 0 ]; then
    echo "Apply a CNI for network failed"
    exit
fi

KUBECONFIG=${kubeConfigFile} kubectl get nodes | grep host
if [ $? -eq 0 ]; then
    echo "SUCCESS"
else
    echo "FAIL"
fi