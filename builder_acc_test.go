package main

import (
	"testing"
	builderT "github.com/hashicorp/packer/helper/builder/testing"
	"fmt"
	"github.com/hashicorp/packer/packer"
	"github.com/vmware/govmomi/vim25/mo"
	"encoding/json"
	"math/rand"
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
	config := defaultConfig()
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: renderConfig(config),
		Check:    checkDefault(t, config["vm_name"].(string), config["host"].(string)),
	})
}

func defaultConfig() map[string]interface{} {
	config := map[string]interface{}{
		"vcenter_server":      "vcenter.vsphere55.test",
		"username":            "root",
		"password":            "jetbrains",
		"insecure_connection": true,

		"template": "basic",
		"host":     "esxi-1.vsphere55.test",

		"ssh_username": "jetbrains",
		"ssh_password": "jetbrains",
	}
	config["vm_name"] = fmt.Sprintf("test-%v", rand.Intn(1000))
	return config
}

func checkDefault(t *testing.T, name string, host string) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		if len(artifacts) > 1 {
			t.Fatal("more than 1 artifact")
		}

		artifactRaw := artifacts[0]
		artifact, ok := artifactRaw.(*Artifact)
		if !ok {
			t.Fatalf("unknown artifact: %#v", artifactRaw)
		}

		conn, err := testConn()
		if err != nil {
			t.Fatal("Cannot connect: ", err)
		}

		vm, err := conn.finder.VirtualMachine(conn.ctx, artifact.Name)
		if err != nil {
			t.Fatal("Cannot find VM: ", err)
		}

		var vmInfo mo.VirtualMachine
		err = vm.Properties(conn.ctx, vm.Reference(), []string{"name", "runtime.host", "resourcePool", "layoutEx.disk"}, &vmInfo)
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		if vmInfo.Name != name {
			t.Errorf("Invalid VM name: expected '%v', got '%v'", name, vmInfo.Name)
		}

		var hostInfo mo.HostSystem
		err = vm.Properties(conn.ctx, vmInfo.Runtime.Host.Reference(), []string{"name"}, &hostInfo)
		if err != nil {
			t.Fatal("Cannot read VM properties: ", err)
		}

		if hostInfo.Name != host {
			t.Errorf("Invalid host name: expected '%v', got '%v'", host, hostInfo.Name)
		}

		var rpInfo = mo.ResourcePool{}
		err = vm.Properties(conn.ctx, vmInfo.ResourcePool.Reference(), []string{"owner", "parent"}, &rpInfo)
		if err != nil {
			t.Fatalf("Cannot read resource pool properties: %v", err)
		}

		if rpInfo.Owner != *rpInfo.Parent {
			t.Error("Not a root resource pool")
		}

		if len(vmInfo.LayoutEx.Disk[0].Chain) != 1 {
			t.Error("Not a full clone")
		}

		return nil
	}
}

func TestBuilderAcc_linkedClone(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: linkedCloneConfig(),
		Check:    checkLinkedClone(t),
	})
}

func linkedCloneConfig() string {
	config := defaultConfig()
	config["linked_clone"] = true
	return renderConfig(config)
}

func checkLinkedClone(t *testing.T) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		artifactRaw := artifacts[0]
		artifact, _ := artifactRaw.(*Artifact)

		conn, err := testConn()
		if err != nil {
			t.Fatalf("Cannot connect: %v", err)
		}

		vm, err := conn.finder.VirtualMachine(conn.ctx, artifact.Name)
		if err != nil {
			t.Fatalf("Cannot find VM: %v", err)
		}

		var vmInfo mo.VirtualMachine
		err = vm.Properties(conn.ctx, vm.Reference(), []string{"layoutEx.disk"}, &vmInfo)
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		if len(vmInfo.LayoutEx.Disk[0].Chain) != 3 {
			t.Error("Not a linked clone")
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
