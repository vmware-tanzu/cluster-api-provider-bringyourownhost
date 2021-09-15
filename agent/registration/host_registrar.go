package registration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"

	"github.com/jackpal/gateway"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Register struct {
	ByoHostName        string `json:"byoHostName,omitempty"`
	ByoHostNameSpace   string `json:"ByoHostNameSpace,omitempty"`
	DefaultNetworkName string `json:"defaultNetworkName,omitempty"`
}
type HostRegistrar struct {
	K8sClient               client.Client
	ByoHostRegsiterFileName string
	RegisterInfo            Register
	ctx                     context.Context
}

const (
	ByoHostNameSuffixLength = 5
	filePermission          = 0644
)

func (hr *HostRegistrar) Register() error {
	hr.ctx = context.TODO()
	err := hr.GetByoHostRegsiterFileName()
	if err != nil {
		klog.Errorf("error getting ByoHost Register File, err=%v", err)
		return err
	}

	byoHost := &infrastructurev1alpha4.ByoHost{}

	/*
		check if /var/run/byohost.json is existed
		if it existed, it means there is already a byohost object, read its name from this file.
		if it not existed, it means this is a new byohost needs to be created.
		The format of /var/run/byohost.json is as followed:
		{
			"byoHostName": "byohost",
			"ByoHostNameSpace": "default",
			"defaultNetworkName": "eth0"
		}

	*/
	isExisted, err := common.IsFileExists(hr.ByoHostRegsiterFileName)
	if err != nil {
		klog.Errorf("error stating file %s, err=%v", hr.ByoHostRegsiterFileName, err)
		return err
	}

	if isExisted {
		// This is for byohost restart situation
		err = hr.ReadObjFromRegisterFile()
		if err != nil {
			klog.Errorf("ReadObjFromRegisterFile %s data, err=%v", hr.ByoHostRegsiterFileName, err)
			return err
		}

		klog.Infof("ByoHostName=%s, DefaultNetworkName=%s", hr.RegisterInfo.ByoHostName, hr.RegisterInfo.DefaultNetworkName)

		if hr.RegisterInfo.ByoHostName == "" || hr.RegisterInfo.ByoHostNameSpace == "" {
			errMsg := fmt.Sprintf("%s corrupted, fix it first", hr.ByoHostRegsiterFileName)
			klog.Errorf(errMsg)
			return errors.New(errMsg)
		}

		err = hr.K8sClient.Get(hr.ctx, types.NamespacedName{Name: hr.RegisterInfo.ByoHostName, Namespace: hr.RegisterInfo.ByoHostNameSpace}, byoHost)
		if err != nil {
			if apierrors.IsNotFound(err) {
				errMsg := fmt.Sprintf("This host is not clean status, it may be can fixed by delete %s", hr.ByoHostRegsiterFileName)
				klog.Errorf(errMsg)
				return errors.New(errMsg)
			}
			klog.Errorf("error getting host %s in namespace %s, err=%v", hr.RegisterInfo.ByoHostName, hr.RegisterInfo.ByoHostNameSpace, err)
			return err
		}
	} else {
		// This is for byohost new-create situation
		// report error when there are two hosts with same hostname
		err = hr.K8sClient.Get(hr.ctx, types.NamespacedName{Name: hr.RegisterInfo.ByoHostName, Namespace: hr.RegisterInfo.ByoHostNameSpace}, byoHost)
		if err == nil {
			errMsg := fmt.Sprintf("Byohost %s in namespace %s already existed, please rename your hostname", hr.RegisterInfo.ByoHostName, hr.RegisterInfo.ByoHostNameSpace)
			klog.Errorf(errMsg)
			return errors.New(errMsg)
		}

		byoHost, err = hr.CreateByoHost()
		if err != nil {
			return err
		}

		// update register file with ByoHostName and ByoHostNamespace instantly
		err = hr.UpdateRegisterFile(Register{ByoHostName: hr.RegisterInfo.ByoHostName}, false)
		if err != nil {
			klog.Errorf("error updating register file  %s, err=%v", hr.ByoHostRegsiterFileName, err)
			return err
		}
	}

	// run it at startup or reboot
	err = hr.UpdateNetwork(byoHost)
	if err != nil {
		klog.Errorf("error updating network, err=%v", err)
		return err
	}

	// update register file with DefaultNetworkName instantly
	if hr.RegisterInfo.DefaultNetworkName == "" {
		klog.Errorf("error getting default network name")
		return errors.New("error getting default network name")
	}
	return hr.UpdateRegisterFile(Register{DefaultNetworkName: hr.RegisterInfo.DefaultNetworkName}, true)
}

func (hr *HostRegistrar) CreateByoHost() (*infrastructurev1alpha4.ByoHost, error) {
	byoHost := &infrastructurev1alpha4.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hr.RegisterInfo.ByoHostName,
			Namespace: hr.RegisterInfo.ByoHostNameSpace,
			Labels: map[string]string{
				clusterv1.WatchLabel: hr.RegisterInfo.ByoHostName,
			},
		},
		Spec:   infrastructurev1alpha4.ByoHostSpec{},
		Status: infrastructurev1alpha4.ByoHostStatus{},
	}
	err := hr.K8sClient.Create(hr.ctx, byoHost)
	if err != nil {
		klog.Errorf("error creating host %s in namespace %s, err=%v", hr.RegisterInfo.ByoHostName, hr.RegisterInfo.ByoHostNameSpace, err)
		return nil, err
	}

	return byoHost, nil
}

func (hr *HostRegistrar) GetByoHostRegsiterFileName() error {
	u, err := user.Current()
	if err != nil {
		klog.Errorf("error gettig current user, err=%v", err)
		return err
	}
	hr.ByoHostRegsiterFileName = filepath.Join(u.HomeDir, "byohost.json")
	klog.Infof("byoHost register file %s", hr.ByoHostRegsiterFileName)
	return nil
}

func (hr *HostRegistrar) UpdateRegisterFile(data Register, isRegisterFileExisted bool) error {
	if isRegisterFileExisted {
		err := hr.ReadObjFromRegisterFile()
		if err != nil {
			klog.Errorf("ReadObjFromRegisterFile %s data, err=%v", hr.ByoHostRegsiterFileName, err)
			return err
		}
	}

	if len(data.ByoHostName) > 0 {
		hr.RegisterInfo.ByoHostName = data.ByoHostName
	}

	if len(data.DefaultNetworkName) > 0 {
		hr.RegisterInfo.DefaultNetworkName = data.DefaultNetworkName
	}

	return hr.WriteObjToRegisterFile(!isRegisterFileExisted)
}

func (hr *HostRegistrar) WriteObjToRegisterFile(isCreated bool) error {
	if isCreated {
		f, err1 := os.Create(hr.ByoHostRegsiterFileName)
		if err1 != nil {
			klog.Errorf("error creating %s, err=%v", hr.ByoHostRegsiterFileName, err1)
			return err1
		}
		defer f.Close()
	}

	fileContent, err := json.MarshalIndent(&hr.RegisterInfo, "", " ")
	if err != nil {
		klog.Errorf("error marshaling registerInfo, err=%v", err)
		return err
	}

	err = ioutil.WriteFile(hr.ByoHostRegsiterFileName, fileContent, filePermission)
	if err != nil {
		klog.Errorf("error writing file %s, err=%v", hr.ByoHostRegsiterFileName, err)
		return err
	}

	return nil
}

func (hr *HostRegistrar) ReadObjFromRegisterFile() error {
	byteValue, err := ioutil.ReadFile(hr.ByoHostRegsiterFileName)
	if err != nil {
		klog.Errorf("error reading file %s, err=%v", hr.ByoHostRegsiterFileName, err)
		return err
	}
	if err = json.Unmarshal(byteValue, &hr.RegisterInfo); err != nil {
		klog.Errorf("error unmarshaling data, err=%v", err)
		return err
	}

	klog.Infof("ByoHostName=%s, DefaultNetworkName=%s", hr.RegisterInfo.ByoHostName, hr.RegisterInfo.DefaultNetworkName)

	return nil
}

func (hr *HostRegistrar) UpdateNetwork(byoHost *infrastructurev1alpha4.ByoHost) error {
	helper, err := patch.NewHelper(byoHost, hr.K8sClient)
	if err != nil {
		return err
	}

	byoHost.Status.Network = hr.GetNetworkStatus()

	return helper.Patch(hr.ctx, byoHost)
}

func (hr *HostRegistrar) GetNetworkStatus() []infrastructurev1alpha4.NetworkStatus {
	Network := []infrastructurev1alpha4.NetworkStatus{}

	defaultIP, err := gateway.DiscoverInterface()
	if err != nil {
		return Network
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return Network
	}

	for _, iface := range ifaces {
		netStatus := infrastructurev1alpha4.NetworkStatus{}

		if iface.Flags&net.FlagUp > 0 {
			netStatus.Connected = true
		}

		netStatus.MACAddr = iface.HardwareAddr.String()
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		netStatus.NetworkName = iface.Name
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.String() == defaultIP.String() {
				netStatus.IsDefault = true
				hr.RegisterInfo.DefaultNetworkName = netStatus.NetworkName
			}
			netStatus.IPAddrs = append(netStatus.IPAddrs, addr.String())
		}
		Network = append(Network, netStatus)
	}
	return Network
}
