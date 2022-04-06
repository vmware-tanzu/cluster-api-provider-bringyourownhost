#!/bin/bash
function isCmdInstalled() {
    local cmd=$1
    which ${cmd}
    if [ $? -eq 1 ] ; then
        echo "$cmd not found"
        return 0
    else
        echo "$cmd found"
        return 1
    fi
}

function exitIfNot(){
    local code=$1
    local expectedCode=$2
    local prompt=$3
    if [ ${code} == ${expectedCode} ]; then
        echo ${prompt}
        exit 1
    fi
}

function runCmd(){
    local cmd=$1
    local expectCode=$2
    local promptBeforeCmd=$3
    local PromptAfterCmdSuc=$4
    local promptAfterCmdFail=$5

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
        exit 1
    fi

    if [ -n "${PromptAfterCmdSuc}" ]; then
        echo ${PromptAfterCmdSuc}
    fi
}

function isIPOccupied() {
    local ip=$1
    local occupiedIPList=$2
    echo "Checking if ${ip} is occupied"

    IFS=$'\n'
    for occupiedIPWithMask in ${occupiedIPList}
    do
        occupiedIP=`echo $occupiedIPWithMask | sed -En 's/^(.*)\/([0-9]{1,2})/\1/p'`
        if [ "${ip}" == "${occupiedIP}" ]; then
             return 1
        fi
    done

    # should not use the gateway address
    for gateway in ${gateways}
    do 
        if [ "${gateway}" == "${ip}" ]; then
            return 1
        fi
    done

    echo "${ip} is not occupied"
    return 0
}



function num2IP() {
    local num=$1
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
    gateways=`docker network inspect kind | jq -r 'map(.IPAM.Config[].Gateway) []'`
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
    echo "gateways:${gateways}"

    ip=`echo $IPMask | sed -En 's/^(.*)\/([0-9]{1,2})/\1/p'`
    ipSubNetBit=`echo $IPMask | sed -En 's/^(.*)\/([0-9]{1,2})/\2/p'`
    ipSubHostBit=$[32-${ipSubNetBit}]

    IFS=.
    hexIP=`for str in ${ip}; do printf "%02X" $str; done`
    binaryIP=`echo "ibase=16; obase=2; ${hexIP}" | bc`

    ipSubNet=`echo ${binaryIP:0:${ipSubNetBit}}`

    full0="00000000000000000000000000000001"
    full1="11111111111111111111111111111110"

    minIPBinary=${ipSubNet}`echo ${full0:${ipSubNetBit}}`
    maxIPBinary=${ipSubNet}`echo ${full1:${ipSubNetBit}}`

    minIPint=`echo "ibase=2; ${minIPBinary}"|bc`
    maxIPint=`echo "ibase=2; ${maxIPBinary}"|bc`

    for((i=${minIPint};i<=${maxIPint};i++));  
    do
        ip=$(num2IP ${i})

        isIPOccupied "${ip}" "${occupiedIPList}" "${gateways}"
        if [ $? == 0 ]; then
            controlPlaneEndPointIp=${ip}
            echo "CONTROL_PLANE_ENDPOINT_IP is ${controlPlaneEndPointIp}"
            return
        fi
    done

    echo "Can't get an available IP for control plane endpoint, exit...."
    exit 1

}

function installByohProvider() {
    local maxRunTimes=40
    local waitTime=20

    runCmd "clusterctl init --infrastructure byoh" 0 " Transforming the Kubernetes cluster into a management cluster..."
    # Waiting for byoh provider is totally ready
    for((i=1;i<=${maxRunTimes};i++));  
    do  
        #byohStatus=$(kubectl get pods --all-namespaces | grep byoh-controller-manager | awk '{print $4}')
        replicas=$(kubectl get deployment byoh-controller-manager -n byoh-system -o json | jq .status.readyReplicas)
        if [ "${replicas}" == "1" ] ; then
            echo "Byoh provider is ready"
            return
        else
            echo "Waiting for byoh-provider to be ready..."
            sleep ${waitTime}
        fi
    done
    echo "Waiting too long for byoh provider, something may wrong with it."
    exit 1
}

function installDocker() {
    local cmdName="docker"

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

    ## check  if dependency is installed successfully
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
        exit 1
    fi
    echo "Enable docker success"
}

function installKind() {
    local cmdName="kind"

     ## check  if dependency is installed before
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
    local cmdName="clusterctl"

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

function commonInstall(){
    local cmdName=$1
    local installCmd=$2
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

function createKindCluster(){
    echo  "Creating kind cluster..."

    kind create cluster --name ${managerClusterName}
    if [ $? -ne 0 ]; then
        echo "Create kind cluster failed"
        exit 1
    else
        echo "Create kind cluster successfully"
    fi
}

function cleanUp(){
    local i=1

    kind delete clusters ${managerClusterName}
    for ((i=1;i<=${byohNums};i++)); 
    do
        docker rm -f host${i}
    done
}

function readArgs() {
    TEMP=`getopt -o nm:c:k: --long cni,md:,cp:,kv:`
    if [ $? != 0 ] ; then 
        echo "Terminating..." >&2 
        exit 1 
    fi
    # Note the quotes around `$TEMP': they are essential!
    while true ; do
        case "$1" in
            -n|--cni) 
                defaultCni="1"
                shift 
                ;;
            -m|--md) 
                workerCount=$2
                shift 2 
                ;;
            -c|--cp) 
                controlPlaneCount=$2
                shift 2 
                ;;
            -k|--kv) 
                kubernetesVersion=$2
                shift 2 
                ;;
            *) 
                break 
                ;;
        esac
    done

    byohNums=$[${workerCount}+${controlPlaneCount}]
}

function installCNI(){
    local maxRunTimes=40
    local waitTime=20
    local cniSucc=0
    local i=1

    # Sometimes work cluster is not entirely ready, it reports error: Unable to connect to the server: dial tcp 172.18.0.5:6443: connect: no route to host
    echo "Applying a CNI for network..."
    for((i=1;i<=${maxRunTimes};i++));  
    do
        KUBECONFIG=${kubeConfigFile} kubectl apply -f https://docs.projectcalico.org/v3.20/manifests/calico.yaml
        if [ $? -ne 0 ]; then
            sleep ${waitTime}
            continue
        else 
            echo "Apply CNI for network successfully"
            cniSucc=1
            break
        fi
    done

    if [ ${cniSucc} -eq 0 ]; then
        echo "Apply a CNI for network failed"
        exit 1
    fi
}

function retrieveKubeConfig() {
    local maxRunTimes=10
    local waitTime=1

    echo "Retrieving the kubeconfig of workload cluster..."
    for((i=1;i<=${maxRunTimes};i++));  
    do   
        kubectl get secret/${workerClusterName}-kubeconfig 2>&1 | grep -q "not found"
        if [ $? -eq 0 ]; then
            sleep ${waitTime}
            continue
        else 
            kubectl get secret/${workerClusterName}-kubeconfig -o json | jq -r .data.value | base64 --decode  > ${kubeConfigFile}
            echo "Retrieve the kubeconfig of workload cluster successfully"
            return
        fi
    done

    echo "Retrieve the kubeconfig of workload cluster failed"
    exit 1
}

function checkNodeStatus() {
    local maxRunTimes=40
    local waitTime=30
    local i=1
    local j=1
    local ready=0

    for((i=1;i<=${byohNums};i++)); 
    do
        ready=0
        for((j=1;j<=${maxRunTimes};j++));  
        do   
            KUBECONFIG=${kubeConfigFile} kubectl get nodes host${i} | grep -q "not found"
            if [ $? -eq 0 ]; then
                sleep ${waitTime}
                continue
            fi
            if [ "${defaultCni}" == "1" ]; then
                status=`KUBECONFIG=${kubeConfigFile} kubectl get nodes host${i} | grep -v NAME | awk '{print $2}'`
                if [ "${status}" != "Ready" ]; then
                    sleep ${waitTime}
                    continue
                fi
            fi
            ready=1
            echo "node \"host${i}\" is ready"
            break

        done

        if [ ${ready} -eq 0 ]; then
            echo "FAIL! node \"host${i}\" is not ready"
            exit 1
        fi
    done
}


function prepareImageAndBinary() {
    runCmd "cd ${reposDir}" 0

    # Build byoh image
    # Check if byoh image is existed
    image=$(docker images ${byohImageName}:${byohImageTag} | grep -v REPOSITORY)
    if [ -z "${image}" ]; then
        cp -f ${reposDir}/test/e2e/kubernetes.list ${reposDir}/test/e2e/kubernetes.list.bak
        # The origin one will report error:   Could not connect to apt.kubernetes.io:443 (10.25.207.164), connection timed out [IP: 10.25.207.164 443]
        echo "deb http://packages.cloud.google.com/apt/ kubernetes-xenial main" > ${reposDir}/test/e2e/kubernetes.list
        runCmd "make prepare-byoh-docker-host-image" 0  "Making a byoh image: ${byohImageName}:${byohImageTag} ..."
        mv -f ${reposDir}/test/e2e/kubernetes.list.bak ${reposDir}/test/e2e/kubernetes.list
    else
        echo "byoh image \"${byohImageName}:${byohImageTag}\" existed."
    fi

    # Build byoh binary
    # Check if byoh binary is existed
    if [ ! -f ${byohBinaryFile} ]; then
        runCmd "make host-agent-binaries" 0  "Making byoh binary: ${byohBinaryFile} ..."
    else
        echo "byoh binary \"${byohBinaryFile}\" existed."
    fi

    runCmd "cd -" 0
}

function bringUpByoHost(){
    local i=1
    local t=1
    local maxRunTimes=10
    local waitTime=1
    local KIND_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ${managerClusterName}-control-plane) 
    local ok=0

    cp -f ${HOME}/.kube/config ${manageClusterConfFile}
    sed -i 's/    server\:.*/    server\: https\:\/\/'"${KIND_IP}"'\:6443/g' ${manageClusterConfFile}

    for (( i=1; i<=${byohNums}; i++ ))
    do
        runCmd "docker run --detach --tty --hostname host${i} --name host${i} --privileged --security-opt seccomp=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /lib/modules:/lib/modules:ro --network kind ${byohImageName}:${byohImageTag}" 0 "Starting byoh container: host${i}..."

        runCmd "docker cp ${byohBinaryFile} host${i}:/byoh-hostagent" 0 "Copying agent binary to byoh container: host${i}..."
        runCmd "docker cp ${manageClusterConfFile} host${i}:/management-cluster.conf" 0 "Copying kubeconfig to byoh container: host${i}..."

        echo "Starting the host${i} agent..."
        docker exec -d host${i} sh -c "chmod +x /byoh-hostagent && /byoh-hostagent --kubeconfig /management-cluster.conf > /agent.log 2>&1"

        ok=0
        for((t=1;t<=${maxRunTimes};t++));  
        do
            kubectl get byohosts host${i} | grep -q "not found" 2>/dev/null
            if [ $? -eq 0 ]; then
                sleep ${waitTime}
                continue
            else   
                echo "byohost object(host${i}) is created successfully..."
                ok=1
                break
            fi
        done
        if [ $ok -eq 0 ]; then
            echo "Error: byohost object(host${i}) is created failed..."
            exit 1
        fi
    done
}

function createWorkloadCluster() {
    local clusterYamlFile="/tmp/cluster-yaml"

    # Find a available IP for control plane endpoint
    calcControlPlaneIP

    CONTROL_PLANE_ENDPOINT_IP=${controlPlaneEndPointIp} clusterctl generate cluster ${workerClusterName} --infrastructure byoh --kubernetes-version ${kubernetesVersion} --control-plane-machine-count ${controlPlaneCount}  --worker-machine-count ${workerCount} --flavor docker > "${clusterYamlFile}"
    if [ $? -ne 0 ]; then
        echo "Generate ${clusterYamlFile} failed, exiting..."
        exit 1
    fi

    echo "Creating the workload cluster..."
    kubectl apply -f ${clusterYamlFile} 
    if [ $? -ne 0 ]; then
        echo "Create the workload cluster failed"
        exit 1
    fi
}

function swapOff() {
    runCmd "sudo swapoff -a" 0 "Turning off swap..."
    swapMsg=$(sudo swapon -s)
    if [ -n "${swapMsg}" ]; then
        echo "Please turn off swap first."
        exit 1
    fi
}


function userConfirmation() {
    local warning='
#####################################################################################################
** WARNING **
This modifys system settings - and do **NOT** revert them at the end of the test.
It locally will change the following host config
- disable swap, but it can revert back if rebooting vm
- use "sudo apt-get update" command to download package information from all configured sources.
- install docker, and enable it as service if not
- install kind, clusterctl, jq, kubectl, build-essential and go, if not
- create a kind cluster as manager cluster, byoh clustr as worker cluster
#####################################################################################################'

    echo "${warning}"
	read -p "Do you want to proceed [Y/N]?" REPLY; 
	if [[ ${REPLY} != "Y" && ${REPLY} != "y" ]]; then 
        echo "Aborting..."
        exit 1
    fi
}

export PATH=/snap/bin:${PATH}
byohImageName="byoh/node"
byohImageTag="e2e"
managerClusterName="kind-byoh"
workerClusterName="worker-byoh"
controlPlaneEndPointIp=""
workerCount=1
controlPlaneCount=1
byohNums=2
defaultCni="0"
manageClusterConfFile="${HOME}/.kube/management-cluster.conf"
kubeConfigFile=/tmp/byoh-cluster-kubeconfig
reposDir=$(dirname $0)/../
byohBinaryFile=${reposDir}/bin/byoh-hostagent-linux-amd64
kubernetesVersion="v1.23.5"

readArgs $@
userConfirmation
swapOff
intallDependencies 
cleanUp
createKindCluster
installByohProvider
prepareImageAndBinary
bringUpByoHost
createWorkloadCluster
retrieveKubeConfig

if [ "${defaultCni}" == "1" ]; then
    installCNI
fi

checkNodeStatus

if [ "${defaultCni}" == "0" ]; then
    echo "Byoh cluster \"${workerClusterName}\" is successfully created, next step is to apply a CNI of your choice."
fi

