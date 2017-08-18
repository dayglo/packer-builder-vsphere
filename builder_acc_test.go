package main

import (
	"testing"
	builderT "github.com/hashicorp/packer/helper/builder/testing"
	"fmt"
	"github.com/hashicorp/packer/packer"
	"github.com/vmware/govmomi/vim25/mo"
	"encoding/json"
)

func TestBuilderAcc_basic(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: testBuilderAccBasic,
	})
}

const testBuilderAccBasic = `
{
	"builders": [{
		"type": "test",

		"vcenter_server": "vcenter.vsphere55.test",
		"username": "root",
		"password": "jetbrains",
		"insecure_connection": true,

		"template": "basic",
		"vm_name": "test-1",
		"host": "esxi-1.vsphere55.test",

		"ssh_username": "jetbrains",
		"ssh_password": "jetbrains"
	}]
}
`

func TestBuilderAcc_default(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: renderConfig(defaultConfig()),
		Check:    checkDefault(defaultConfig()),
	})
}

func defaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"vcenter_server":      "vcenter.vsphere55.test",
		"username":            "root",
		"password":            "jetbrains",
		"insecure_connection": true,

		"template": "basic",
		"vm_name":  "test-1",
		"host":     "esxi-1.vsphere55.test",

		"ssh_username": "jetbrains",
		"ssh_password": "jetbrains",
	}
}

func checkDefault(config map[string]interface{}) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		if len(artifacts) > 1 {
			return fmt.Errorf("more than 1 artifact")
		}

		artifactRaw := artifacts[0]
		artifact, ok := artifactRaw.(*Artifact)
		if !ok {
			return fmt.Errorf("unknown artifact: %#v", artifactRaw)
		}

		conn, err := testConn()
		if err != nil {
			return err
		}

		vm, err := conn.finder.VirtualMachine(conn.ctx, artifact.Name)
		if err != nil {
			return err
		}

		var vmInfo mo.VirtualMachine
		err = vm.Properties(conn.ctx, vm.Reference(), []string{"name", "runtime.host", "resourcePool", "layoutEx.disk"}, &vmInfo)
		if err != nil {
			return err
		}

		if vmInfo.Name != config["vm_name"] {
			return fmt.Errorf("Invalid VM name: expected '%v', got '%v'", config["vm_name"], vmInfo.Name)
		}

		var hostInfo mo.HostSystem
		err = vm.Properties(conn.ctx, vmInfo.Runtime.Host.Reference(), []string{"name"}, &hostInfo)
		if err != nil {
			return err
		}

		if hostInfo.Name != config["host"] {
			return fmt.Errorf("Invalid host name: expected '%v', got '%v'", config["host"], hostInfo.Name)
		}

		var rpInfo = mo.ResourcePool{}
		err = vm.Properties(conn.ctx, vmInfo.ResourcePool.Reference(), []string{"owner", "parent"}, &rpInfo)
		if err != nil {
			return err
		}

		if rpInfo.Owner != *rpInfo.Parent {
			return fmt.Errorf("Not a root resource pool")
		}

		if len(vmInfo.LayoutEx.Disk[0].Chain) != 1 {
			return fmt.Errorf("Not a full clone")
		}

		return nil
	}
}

func renderConfig(config map[string]interface{}) string {
	t := map[string][]map[string]interface{}{
		"builders": {
			map[string]interface{}{
				"type": "test",
			},
		},
	}
	for k, v := range config {
		t["builders"][0][k] = v
	}

	j, _ := json.Marshal(t)
	return string(j)
}

func testConn() (*Driver, error) {
	config := &ConnectConfig{
		VCenterServer:      "vcenter.vsphere55.test",
		Username:           "root",
		Password:           "jetbrains",
		InsecureConnection: true,
	}

	return NewDriver(config)
}
