package vpn

import (
	"fmt"
	"io"
	"net/http"

	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/cloud-builder/terraform"
	"github.com/mevansam/gocloud/cloud"
)

type vpnConfig struct {	
	name    string
	vpnType string
	data    []byte
}

func NewStaticConfigData(tgt *target.Target, user, passwd string) (*vpnConfig, error) {

	var (
		err error
		ok  bool

		instance      *target.ManagedInstance
		instanceState cloud.InstanceState

		output terraform.Output
		
		vpcName string

		client *http.Client
		url    string

		req     *http.Request
		resp    *http.Response
		resBody []byte
	)
	c := &vpnConfig{}

	// validate target bastion
	if !tgt.Recipe.IsBastion() {
		return nil, fmt.Errorf(fmt.Sprintf("target \"%s\" is not a bastion node", tgt.Key()))
	}
	if tgt.Status() != target.Running {
		return nil, fmt.Errorf("target is not running")
	}
	instance = tgt.ManagedInstance("bastion")
	if instance == nil {
		return nil, fmt.Errorf("unable to find a bastion instance to connect to")
	}
	if instanceState, err = instance.Instance.State(); err != nil {
		return nil, err
	}
	if instanceState != cloud.StateRunning {
		return nil, fmt.Errorf("bastion instance is not running")
	}
	if client, url, err = instance.HttpsClient(); err != nil {
		return nil, err
	}

	// get a name for the space vpn
	if output, ok = (*tgt. Output)["cb_vpc_name"]; !ok {
		return nil, fmt.Errorf("the vpc name was not present in the sandbox build output")
	}
	if vpcName, ok = output.Value.(string); !ok {
		return nil, fmt.Errorf("target's \"cb_vpc_name\" output was not a string %#v", output)
	}
	c.name = fmt.Sprintf(
		"%s.conf",
		vpcName,
	)
	url = fmt.Sprintf(
		"%s/static/~%s/%s",
		url, user, c.name,
	)

	// get the vpn type
	if output, ok = (*tgt.Output)["cb_vpn_type"]; !ok {
		return nil, fmt.Errorf("the vpn type was not present in the sandbox build output")
	}
	if c.vpnType, ok = output.Value.(string); !ok {
		return nil, fmt.Errorf("target's \"cb_vpn_type\" output was not a string: %#v", output)
	}

	if req, err = http.NewRequest("GET", url, nil); err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, passwd)
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error retrieving vpn config from bastion instance: %s", resp.Status)
	}
	if resBody, err = io.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	c.data = resBody
	return c, nil
}

func (c *vpnConfig) Name() string {	
	return c.name
}

func (c *vpnConfig) VPNType() string {	
	return c.vpnType
}

func (c *vpnConfig) Data() []byte {	
	return c.data
}
