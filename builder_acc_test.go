package main

import (
	"testing"
	builderT "github.com/hashicorp/packer/helper/builder/testing"
	"fmt"
	"github.com/hashicorp/packer/packer"
	"github.com/vmware/govmomi/vim25/mo"
)

func TestBuilderAcc_basic(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: testBuilderAccBasic,
		Check: checkBasic(),
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

func checkBasic() builderT.TestCheckFunc {
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

		if vm.Name() != artifact.Name {
			return fmt.Errorf("Invalid VM name: expected '%v', got '%v'", artifact.Name, vm.Name())
		}

		host, err := vm.HostSystem(conn.ctx)
		if err != nil {
			return err
		}
		var h mo.HostSystem
		err = host.Properties(conn.ctx, host.Reference(), []string{"name"}, &h)
		if err != nil {
			return err
		}

		hostname := "esxi-1.vsphere55.test"
		if h.Name != hostname {
			return fmt.Errorf("Invalid host name: expected '%v', got '%v'", hostname, h.Name)
		}

		pool, err := vm.ResourcePool(conn.ctx)
		if err != nil {
			return err
		}
		var p mo.ResourcePool
		err = pool.Properties(conn.ctx, pool.Reference(), []string{"owner", "parent"}, &p)
		if err != nil {
			return err
		}
		if p.Owner != *p.Parent {
			return fmt.Errorf("Not a root resource pool")
		}

		return nil
	}
}
func testConn() (*Driver, error) {
	config := &ConnectConfig{
		VCenterServer: "vcenter.vsphere55.test",
		Username: "root",
		Password: "jetbrains",
		InsecureConnection: true,
	}

	return NewDriver(config)
}
